package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

// Action describes the type of change to make.
type Action string

const (
	CreateApp           Action = "create_app"
	DestroyApp          Action = "destroy_app"
	UpdateApp           Action = "update_app"
	CreateService       Action = "create_service"
	DestroyService      Action = "destroy_service"
	UpdateService       Action = "update_service"
	CreateMailService   Action = "create_mail_service"
	DestroyMailService  Action = "destroy_mail_service"
	UpdateMailService   Action = "update_mail_service"
	CreateAuthDirectory Action = "create_auth_directory"
	DestroyAuthDirectory Action = "destroy_auth_directory"
	UpdateAuthDirectory Action = "update_auth_directory"
	CreateAuthFrontend  Action = "create_auth_frontend"
	DestroyAuthFrontend Action = "destroy_auth_frontend"
	UpdateAuthFrontend  Action = "update_auth_frontend"
	InstallPlugin       Action = "install_plugin"
	UninstallPlugin     Action = "uninstall_plugin"
	UpdatePlugin        Action = "update_plugin"
	UpdateGlobal        Action = "update_global"
)

// Step is a single planned change.
type Step struct {
	Action      Action `json:"action"`
	App         string `json:"app,omitempty"`
	Service     string `json:"service,omitempty"`
	ServiceType string `json:"service_type,omitempty"`
	Field       string `json:"field,omitempty"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
}

// Plan holds the list of steps needed to converge actual state to desired state.
type Plan struct {
	Steps []Step `json:"steps"`
}

// String returns a human-readable summary of the plan.
func (p *Plan) String() string {
	if len(p.Steps) == 0 {
		return "No changes needed."
	}
	var b strings.Builder
	for _, s := range p.Steps {
		switch s.Action {
		case CreateApp:
			fmt.Fprintf(&b, "+ app %q\n", s.App)
		case DestroyApp:
			fmt.Fprintf(&b, "- app %q\n", s.App)
		case UpdateApp:
			fmt.Fprintf(&b, "~ app %q: %s %q → %q\n", s.App, s.Field, s.OldValue, s.NewValue)
		case CreateService:
			fmt.Fprintf(&b, "+ service %q\n", s.Service)
		case DestroyService:
			fmt.Fprintf(&b, "- service %q\n", s.Service)
		case UpdateService:
			fmt.Fprintf(&b, "~ service %q: %s %q → %q\n", s.Service, s.Field, s.OldValue, s.NewValue)
		case CreateMailService:
			fmt.Fprintf(&b, "+ mail service %q\n", s.Service)
		case DestroyMailService:
			fmt.Fprintf(&b, "- mail service %q\n", s.Service)
		case UpdateMailService:
			fmt.Fprintf(&b, "~ mail service %q\n", s.Service)
		case CreateAuthDirectory:
			fmt.Fprintf(&b, "+ auth directory %q\n", s.Service)
		case DestroyAuthDirectory:
			fmt.Fprintf(&b, "- auth directory %q\n", s.Service)
		case UpdateAuthDirectory:
			fmt.Fprintf(&b, "~ auth directory %q\n", s.Service)
		case CreateAuthFrontend:
			fmt.Fprintf(&b, "+ auth frontend %q\n", s.Service)
		case DestroyAuthFrontend:
			fmt.Fprintf(&b, "- auth frontend %q\n", s.Service)
		case UpdateAuthFrontend:
			fmt.Fprintf(&b, "~ auth frontend %q\n", s.Service)
		case InstallPlugin:
			fmt.Fprintf(&b, "+ plugin %q\n", s.Service)
		case UninstallPlugin:
			fmt.Fprintf(&b, "- plugin %q\n", s.Service)
		case UpdatePlugin:
			fmt.Fprintf(&b, "~ plugin %q\n", s.Service)
		case UpdateGlobal:
			fmt.Fprintf(&b, "~ global: %s\n", s.Field)
		}
	}
	return b.String()
}

// Diff computes the plan to go from actual to desired state.
func Diff(desired, actual *schema.Dokkufile) *Plan {
	var steps []Step

	steps = append(steps, diffServices(desired, actual)...)
	steps = append(steps, diffMailServices(desired, actual)...)
	steps = append(steps, diffAuthDirectories(desired, actual)...)
	steps = append(steps, diffAuthFrontends(desired, actual)...)
	steps = append(steps, diffPlugins(desired, actual)...)
	// Plugins should be installed before apps that may depend on them
	steps = append(steps, diffGlobal(desired, actual)...)
	steps = append(steps, diffApps(desired, actual)...)

	return &Plan{Steps: steps}
}

func diffServices(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredSvc := desired.Services
	actualSvc := actual.Services

	if desiredSvc == nil {
		desiredSvc = map[string]schema.Service{}
	}
	if actualSvc == nil {
		actualSvc = map[string]schema.Service{}
	}

	for name, svc := range desiredSvc {
		if actual, exists := actualSvc[name]; !exists {
			steps = append(steps, Step{
				Action:      CreateService,
				Service:     name,
				ServiceType: svc.Type,
			})
		} else if svc.ImageVersion != "" && svc.ImageVersion != actual.ImageVersion {
			steps = append(steps, Step{
				Action:      UpdateService,
				Service:     name,
				ServiceType: svc.Type,
				Field:       "image_version",
				OldValue:    actual.ImageVersion,
				NewValue:    svc.ImageVersion,
			})
		}
	}

	return steps
}

func diffMailServices(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredMail := desired.MailServices
	actualMail := actual.MailServices
	if desiredMail == nil {
		desiredMail = map[string]schema.MailService{}
	}
	if actualMail == nil {
		actualMail = map[string]schema.MailService{}
	}

	for name, d := range desiredMail {
		if a, exists := actualMail[name]; !exists {
			steps = append(steps, Step{Action: CreateMailService, Service: name})
		} else if d.Provider != a.Provider || !mapEqual(d.Config, a.Config) {
			steps = append(steps, Step{Action: UpdateMailService, Service: name})
		}
	}
	return steps
}

func diffAuthDirectories(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredDir := desired.AuthDirectories
	actualDir := actual.AuthDirectories
	if desiredDir == nil {
		desiredDir = map[string]schema.AuthDirectory{}
	}
	if actualDir == nil {
		actualDir = map[string]schema.AuthDirectory{}
	}

	for name, d := range desiredDir {
		if a, exists := actualDir[name]; !exists {
			steps = append(steps, Step{Action: CreateAuthDirectory, Service: name})
		} else if d.Provider != a.Provider || !mapEqual(d.Config, a.Config) {
			steps = append(steps, Step{Action: UpdateAuthDirectory, Service: name})
		}
	}
	return steps
}

func diffAuthFrontends(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredFE := desired.AuthFrontends
	actualFE := actual.AuthFrontends
	if desiredFE == nil {
		desiredFE = map[string]schema.AuthFrontend{}
	}
	if actualFE == nil {
		actualFE = map[string]schema.AuthFrontend{}
	}

	for name, d := range desiredFE {
		if a, exists := actualFE[name]; !exists {
			steps = append(steps, Step{Action: CreateAuthFrontend, Service: name})
		} else if !authFrontendEqual(d, a) {
			steps = append(steps, Step{Action: UpdateAuthFrontend, Service: name})
		}
	}
	return steps
}

func authFrontendEqual(a, b schema.AuthFrontend) bool {
	if a.Provider != b.Provider || a.Directory != b.Directory || a.OIDCEnabled != b.OIDCEnabled {
		return false
	}
	if !mapEqual(a.Config, b.Config) {
		return false
	}
	if !sliceEqual(a.ProtectedApps, b.ProtectedApps) {
		return false
	}
	if len(a.OIDCClients) != len(b.OIDCClients) {
		return false
	}
	for i := range a.OIDCClients {
		if a.OIDCClients[i] != b.OIDCClients[i] {
			return false
		}
	}
	return true
}

func diffPlugins(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredPlugins := desired.Plugins
	actualPlugins := actual.Plugins
	if desiredPlugins == nil {
		desiredPlugins = map[string]schema.Plugin{}
	}
	if actualPlugins == nil {
		actualPlugins = map[string]schema.Plugin{}
	}

	for name := range desiredPlugins {
		if _, exists := actualPlugins[name]; !exists {
			steps = append(steps, Step{Action: InstallPlugin, Service: name})
		}
		// Don't diff URL/Committish — dokku doesn't expose the install URL,
		// so actual plugins always have empty URL. Only detect missing plugins.
	}
	return steps
}

func diffGlobal(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	dg := desired.Global
	ag := actual.Global
	// If no global section in desired dokkufile, skip global diff entirely.
	// Global config should only be managed when explicitly declared.
	if dg == nil {
		return nil
	}
	if ag == nil {
		ag = &schema.GlobalConfig{}
	}

	if !sliceEqual(dg.Domains, ag.Domains) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "domains"})
	}
	if !nginxConfigEqual(dg.Nginx, ag.Nginx) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "nginx"})
	}
	if !proxyConfigEqual(dg.Proxy, ag.Proxy) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "proxy"})
	}
	if !networkConfigEqual(dg.Network, ag.Network) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "network"})
	}
	if !builderEqual(dg.Builder, ag.Builder) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "builder"})
	}
	if !registryEqual(dg.Registry, ag.Registry) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "registry"})
	}
	if !logConfigEqual(dg.Logs, ag.Logs) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "logs"})
	}
	if !schedulerConfigEqual(dg.Scheduler, ag.Scheduler) {
		steps = append(steps, Step{Action: UpdateGlobal, Field: "scheduler"})
	}

	return steps
}

func diffApps(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredApps := desired.Apps
	actualApps := actual.Apps

	if desiredApps == nil {
		desiredApps = map[string]schema.App{}
	}
	if actualApps == nil {
		actualApps = map[string]schema.App{}
	}

	// New apps
	for name, dApp := range desiredApps {
		if _, exists := actualApps[name]; !exists {
			steps = append(steps, Step{
				Action:   CreateApp,
				App:      name,
				NewValue: dApp.Image,
			})
			continue
		}
		// Existing app — diff fields
		aApp := actualApps[name]
		steps = append(steps, diffApp(name, dApp, aApp)...)
	}

	return steps
}

func diffApp(name string, desired, actual schema.App) []Step {
	var steps []Step

	if desired.Image != actual.Image {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "image",
			OldValue: actual.Image,
			NewValue: desired.Image,
		})
	}

	if !sliceEqual(desired.Domains, actual.Domains) {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "domains",
			OldValue: strings.Join(actual.Domains, ","),
			NewValue: strings.Join(desired.Domains, ","),
		})
	}

	// Only diff env if the dokkufile explicitly declares an env section.
	// nil means "unmanaged" (don't touch), empty map means "clear all".
	if desired.Env != nil && !envEqual(desired.Env, actual.Env, desired.Secrets) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "env",
		})
	}

	if !sliceEqual(desired.Secrets, actual.Secrets) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "secrets",
		})
	}

	if !mapEqual(desired.Links, actual.Links) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "links",
		})
	}

	// Only diff ports if the dokkufile explicitly declares them.
	// nil means "unmanaged" (don't touch).
	if desired.Ports != nil && !portsEqual(desired.Ports, actual.Ports) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "ports",
		})
	}

	if !sliceEqual(desired.Storage, actual.Storage) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "storage",
		})
	}

	// Only diff scale if the dokkufile explicitly declares it.
	// nil means "unmanaged" (don't touch).
	if desired.Scale != nil && !mapIntEqual(desired.Scale, actual.Scale) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "scale",
		})
	}

	if !sliceEqual(desired.DockerOptions.Build, actual.DockerOptions.Build) ||
		!sliceEqual(desired.DockerOptions.Deploy, actual.DockerOptions.Deploy) ||
		!sliceEqual(desired.DockerOptions.Run, actual.DockerOptions.Run) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "docker_options",
		})
	}

	if desired.LetsEncrypt != actual.LetsEncrypt {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "letsencrypt",
		})
	}

	// Git config
	if !gitConfigEqual(desired.Git, actual.Git) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "git",
		})
	}

	// Network config
	if !networkConfigEqual(desired.Network, actual.Network) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "network",
		})
	}

	// Nginx config
	if !nginxConfigEqual(desired.Nginx, actual.Nginx) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "nginx",
		})
	}

	// Proxy config — only diff if explicitly declared.
	// nil means "unmanaged" (don't touch).
	if desired.Proxy != nil && !proxyConfigEqual(desired.Proxy, actual.Proxy) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "proxy",
		})
	}

	// Mail link
	if desired.Mail != actual.Mail {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "mail",
			OldValue: actual.Mail,
			NewValue: desired.Mail,
		})
	}

	// Auth config
	if !authConfigEqual(desired.Auth, actual.Auth) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "auth",
		})
	}

	// SSL config
	if !sslConfigEqual(desired.SSL, actual.SSL) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "ssl",
		})
	}

	// Healthchecks — only diff if explicitly declared.
	// nil means "unmanaged" (don't touch).
	if desired.Healthchecks != nil && !healthchecksEqual(desired.Healthchecks, actual.Healthchecks) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "healthchecks",
		})
	}

	// Cron
	if !cronEqual(desired.Cron, actual.Cron) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "cron",
		})
	}

	// Nginx template
	if desired.NginxTemplate != actual.NginxTemplate {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "nginx_template",
			OldValue: actual.NginxTemplate,
			NewValue: desired.NginxTemplate,
		})
	}

	// Resources
	if !resourcesEqual(desired.Resources, actual.Resources) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "resources",
		})
	}

	// Checks
	if !checksEqual(desired.Checks, actual.Checks) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "checks",
		})
	}

	// Builder
	if !builderEqual(desired.Builder, actual.Builder) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "builder",
		})
	}

	// Registry
	if !registryEqual(desired.Registry, actual.Registry) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "registry",
		})
	}

	// Maintenance
	if desired.Maintenance != actual.Maintenance {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "maintenance",
		})
	}

	// Scripts (deployment tasks)
	if !scriptsEqual(desired.Scripts, actual.Scripts) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "scripts",
		})
	}

	// Locked
	if desired.Locked != actual.Locked {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "locked",
		})
	}

	// Logs
	if !logConfigEqual(desired.Logs, actual.Logs) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "logs",
		})
	}

	// Scheduler
	if !schedulerConfigEqual(desired.Scheduler, actual.Scheduler) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "scheduler",
		})
	}

	// Buildpacks
	if !buildpacksEqual(desired.Buildpacks, actual.Buildpacks) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "buildpacks",
		})
	}

	// Process management
	if !processEqual(desired.Process, actual.Process) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "process",
		})
	}

	return steps
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sorted := func(s []string) []string {
		c := make([]string, len(s))
		copy(c, s)
		sort.Strings(c)
		return c
	}
	sa, sb := sorted(a), sorted(b)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// portsEqual compares port maps after normalizing to a common format.
// The YAML schema uses {"http": "80:3001"} (scheme -> host:container).
// The state reader uses {"http:80": "3001"} (scheme:host -> container).
// This function normalizes both to the state reader format before comparing.
func portsEqual(a, b map[string]string) bool {
	return mapEqual(normalizePorts(a), normalizePorts(b))
}

// normalizePorts converts port maps to "scheme:host" -> "container" format.
func normalizePorts(ports map[string]string) map[string]string {
	if ports == nil {
		return nil
	}
	result := map[string]string{}
	for k, v := range ports {
		if !strings.Contains(k, ":") {
			// Short form: scheme -> "host:container"
			parts := strings.SplitN(v, ":", 2)
			if len(parts) == 2 {
				result[k+":"+parts[0]] = parts[1]
			} else {
				result[k] = v
			}
		} else {
			// Already normalized: "scheme:host" -> "container"
			result[k] = v
		}
	}
	return result
}

func mapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func gitConfigEqual(a, b *schema.GitConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Skip Repo comparison — dokku doesn't store the source URL,
	// so actual.Repo is always empty. Only compare Branch and KeepGitDir.
	return a.Branch == b.Branch && a.KeepGitDir == b.KeepGitDir
}

func networkConfigEqual(a, b *schema.NetworkConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func nginxConfigEqual(a, b *schema.NginxConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.HSTS == b.HSTS &&
		a.HSTSIncludeSubdomains == b.HSTSIncludeSubdomains &&
		a.HSTSMaxAge == b.HSTSMaxAge &&
		a.HSTSPreload == b.HSTSPreload &&
		mapEqual(a.Properties, b.Properties)
}

func proxyConfigEqual(a, b *schema.ProxyConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Enabled == b.Enabled &&
		a.Type == b.Type &&
		mapEqual(a.Caddy, b.Caddy) &&
		mapEqual(a.HAProxy, b.HAProxy) &&
		mapEqual(a.Traefik, b.Traefik)
}

func authConfigEqual(a, b *schema.AuthConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Directory == b.Directory && a.Protected == b.Protected
}

func sslConfigEqual(a, b *schema.SSLConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// CertFile and KeyFile are desired-only (state reader can't read file paths).
	// If both are non-nil, SSL is present on both sides — treat as equal.
	return true
}

func healthchecksEqual(a, b map[string][]schema.HealthcheckConfig) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
	}
	return true
}

func cronEqual(a, b []schema.CronJob) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func resourcesEqual(a, b map[string]schema.ResourceConfig) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || av != bv {
			return false
		}
	}
	return true
}

func checksEqual(a, b *schema.ChecksConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return sliceEqual(a.Disabled, b.Disabled) && sliceEqual(a.Skipped, b.Skipped) && a.WaitToRetire == b.WaitToRetire
}

func builderEqual(a, b *schema.BuilderConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func registryEqual(a, b *schema.RegistryConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func mapIntEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func scriptsEqual(a, b *schema.ScriptsConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func processEqual(a, b *schema.ProcessConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func logConfigEqual(a, b *schema.LogConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func schedulerConfigEqual(a, b *schema.SchedulerConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// envEqual compares env maps, excluding keys that are in the secrets list.
// Secret env vars are managed separately via the secrets mechanism, so
// we don't flag them as env drift.
func envEqual(desired, actual map[string]string, secrets []string) bool {
	secretSet := map[string]bool{}
	for _, s := range secrets {
		secretSet[s] = true
	}
	// Count non-secret keys in each
	dCount, aCount := 0, 0
	for k := range desired {
		if !secretSet[k] {
			dCount++
		}
	}
	for k := range actual {
		if !secretSet[k] {
			aCount++
		}
	}
	if dCount != aCount {
		return false
	}
	for k, v := range desired {
		if secretSet[k] {
			continue
		}
		if actual[k] != v {
			return false
		}
	}
	for k := range actual {
		if secretSet[k] {
			continue
		}
		if _, ok := desired[k]; !ok {
			return false
		}
	}
	return true
}

func buildpacksEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
