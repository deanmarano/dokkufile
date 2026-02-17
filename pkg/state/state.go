package state

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

// CommandRunner abstracts command execution so tests can use fakes.
type CommandRunner interface {
	Run(args ...string) (string, error)
}

// StdinRunner extends CommandRunner with stdin support.
type StdinRunner interface {
	CommandRunner
	RunWithStdin(stdin io.Reader, args ...string) (string, error)
}

// FileRunner abstracts file I/O for reading/writing files on the server.
type FileRunner interface {
	ReadFile(path string) (string, error)
	WriteFile(path string, content []byte, perm os.FileMode) error
}

// ExecRunner shells out to the real dokku binary.
type ExecRunner struct{}

func (r *ExecRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("dokku", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *ExecRunner) RunWithStdin(stdin io.Reader, args ...string) (string, error) {
	cmd := exec.Command("dokku", args...)
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ExecFileRunner implements FileRunner using the local filesystem.
type ExecFileRunner struct{}

func (r *ExecFileRunner) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func (r *ExecFileRunner) WriteFile(path string, content []byte, perm os.FileMode) error {
	return os.WriteFile(path, content, perm)
}

// Reader reads live state from a dokku server.
type Reader interface {
	Read() (*schema.Dokkufile, error)
}

// DokkuReader reads state by shelling out to dokku commands.
type DokkuReader struct {
	Runner     CommandRunner
	FileRunner FileRunner
}

// serviceTypes lists the backing service plugins to scan.
var serviceTypes = []string{
	"postgres", "redis", "mysql", "mariadb", "mongo",
	"clickhouse", "couchdb", "elasticsearch", "memcached",
	"meilisearch", "nats", "rabbitmq", "rethinkdb", "solr", "typesense",
}

// Read shells out to dokku to build the current server state.
func (r *DokkuReader) Read() (*schema.Dokkufile, error) {
	df := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{},
		Services: map[string]schema.Service{},
	}

	// Read services first (needed for link detection).
	// Map of service type -> list of service names.
	allServices := map[string][]string{}
	for _, svcType := range serviceTypes {
		out, err := r.Runner.Run(svcType + ":list")
		if err != nil {
			continue // plugin not installed
		}
		names := parseServiceList(out)
		for _, name := range names {
			df.Services[name] = schema.Service{Type: svcType}
		}
		allServices[svcType] = names
	}

	// Read mail services.
	if out, err := r.Runner.Run("mail:list"); err == nil {
		mailNames := parseServiceList(out)
		if len(mailNames) > 0 {
			df.MailServices = map[string]schema.MailService{}
		}
		for _, name := range mailNames {
			svc := schema.MailService{}
			if info, err := r.Runner.Run("mail:info", name); err == nil {
				svc.Provider = parseReportField(info, "Provider")
				cfg := parseReportConfigFields(info)
				if len(cfg) > 0 {
					svc.Config = cfg
				}
			}
			df.MailServices[name] = svc
		}
	}

	// Read auth directories.
	if out, err := r.Runner.Run("auth:list"); err == nil {
		dirNames := parseServiceList(out)
		if len(dirNames) > 0 {
			df.AuthDirectories = map[string]schema.AuthDirectory{}
		}
		for _, name := range dirNames {
			dir := schema.AuthDirectory{}
			if info, err := r.Runner.Run("auth:info", name); err == nil {
				dir.Provider = parseReportField(info, "Provider")
				cfg := parseReportConfigFields(info)
				if len(cfg) > 0 {
					dir.Config = cfg
				}
			}
			df.AuthDirectories[name] = dir
		}
	}

	// Read auth frontends.
	if out, err := r.Runner.Run("auth:frontend:list"); err == nil {
		feNames := parseServiceList(out)
		if len(feNames) > 0 {
			df.AuthFrontends = map[string]schema.AuthFrontend{}
		}
		for _, name := range feNames {
			fe := schema.AuthFrontend{}
			if info, err := r.Runner.Run("auth:frontend:info", name); err == nil {
				fe.Provider = parseReportField(info, "Provider")
				fe.Directory = parseReportField(info, "Directory")
				apps := parseReportField(info, "Protected apps")
				if apps != "" {
					fe.ProtectedApps = strings.Fields(apps)
				}
				cfg := parseReportConfigFields(info)
				if len(cfg) > 0 {
					fe.Config = cfg
				}
			}
			// OIDC
			if oidcOut, err := r.Runner.Run("auth:oidc:list", name); err == nil {
				clients := parseOIDCClients(oidcOut)
				if len(clients) > 0 {
					fe.OIDCEnabled = true
					fe.OIDCClients = clients
				}
			}
			df.AuthFrontends[name] = fe
		}
	}

	// Read plugins.
	if out, err := r.Runner.Run("plugin:list"); err == nil {
		plugins := parsePluginList(out)
		if len(plugins) > 0 {
			df.Plugins = plugins
		}
	}

	// Read apps.
	appsOut, err := r.Runner.Run("apps:list")
	if err != nil {
		return nil, err
	}
	appNames := parseAppsList(appsOut)

	for _, appName := range appNames {
		app := schema.App{}

		// Image
		if out, err := r.Runner.Run("git:report", appName, "--git-source-image"); err == nil {
			app.Image = strings.TrimSpace(out)
		}

		// Git config
		if out, err := r.Runner.Run("git:report", appName); err == nil {
			branch := parseReportField(out, "Git deploy branch")
			keepGitDir := parseReportField(out, "Git keep git dir")
			if branch != "" || keepGitDir == "true" {
				git := &schema.GitConfig{}
				if branch != "" {
					git.Branch = branch
				}
				if keepGitDir == "true" {
					git.KeepGitDir = true
				}
				app.Git = git
			}
		}

		// Domains
		if out, err := r.Runner.Run("domains:report", appName, "--domains-app-vhosts"); err == nil {
			trimmed := strings.TrimSpace(out)
			if trimmed != "" {
				app.Domains = strings.Fields(trimmed)
			}
		}

		// Ports
		if out, err := r.Runner.Run("ports:list", appName); err == nil {
			ports := parsePortsList(out)
			if len(ports) > 0 {
				app.Ports = ports
			}
		}

		// Env
		if out, err := r.Runner.Run("config:export", appName); err == nil {
			env := parseExportLines(out)
			if len(env) > 0 {
				app.Env = env
			}
		}

		// Storage (deploy mounts)
		if out, err := r.Runner.Run("storage:report", appName); err == nil {
			mounts := splitCommaList(parseReportField(out, "Storage deploy mounts"))
			if len(mounts) > 0 {
				app.Storage = mounts
			}
		}

		// Docker Options
		if out, err := r.Runner.Run("docker-options:report", appName); err == nil {
			build := splitCommaList(parseReportField(out, "Docker options build"))
			deploy := splitCommaList(parseReportField(out, "Docker options deploy"))
			run := splitCommaList(parseReportField(out, "Docker options run"))
			if len(build) > 0 || len(deploy) > 0 || len(run) > 0 {
				app.DockerOptions = schema.DockerOptions{
					Build:  build,
					Deploy: deploy,
					Run:    run,
				}
			}
		}

		// Scale
		if out, err := r.Runner.Run("ps:scale", appName); err == nil {
			scale := parseScaleOutput(out)
			if len(scale) > 0 {
				app.Scale = scale
			}
		}

		// Network config
		if out, err := r.Runner.Run("network:report", appName); err == nil {
			net := &schema.NetworkConfig{
				AttachPostCreate:  parseReportField(out, "Network attach post create"),
				AttachPostDeploy:  parseReportField(out, "Network attach post deploy"),
				BindAllInterfaces: parseReportField(out, "Network bind all interfaces") == "true",
				InitialNetwork:    parseReportField(out, "Network initial network"),
				StaticWebListener: parseReportField(out, "Network static web listener"),
				TLD:               parseReportField(out, "Network tld"),
			}
			if net.AttachPostCreate != "" || net.AttachPostDeploy != "" || net.BindAllInterfaces ||
				net.InitialNetwork != "" || net.StaticWebListener != "" || net.TLD != "" {
				app.Network = net
			}
		}

		// Nginx config
		if out, err := r.Runner.Run("nginx:report", appName); err == nil {
			nginx := &schema.NginxConfig{
				HSTS:                  parseReportField(out, "Nginx hsts") == "true",
				HSTSIncludeSubdomains: parseReportField(out, "Nginx hsts include subdomains") == "true",
				HSTSPreload:           parseReportField(out, "Nginx hsts preload") == "true",
			}
			if maxAge := parseReportField(out, "Nginx hsts max age"); maxAge != "" {
				fmt.Sscanf(maxAge, "%d", &nginx.HSTSMaxAge)
			}
			// Parse extended nginx properties
			props := parseNginxProperties(out)
			if len(props) > 0 {
				nginx.Properties = props
			}
			if nginx.HSTS || nginx.HSTSIncludeSubdomains || nginx.HSTSMaxAge > 0 || nginx.HSTSPreload || len(nginx.Properties) > 0 {
				app.Nginx = nginx
			}
		}

		// Proxy config
		if out, err := r.Runner.Run("proxy:report", appName); err == nil {
			proxy := &schema.ProxyConfig{
				Enabled: parseReportField(out, "Proxy enabled") == "true",
				Type:    parseReportField(out, "Proxy type"),
			}
			if proxy.Enabled || proxy.Type != "" {
				app.Proxy = proxy
			}
		}

		// SSL certs
		if out, err := r.Runner.Run("certs:report", appName); err == nil {
			sslPresent := parseReportField(out, "Ssl cert present")
			if sslPresent == "true" {
				app.SSL = &schema.SSLConfig{}
			}
		}

		// Resource limits
		if out, err := r.Runner.Run("resource:report", appName); err == nil {
			resources := parseResourceReport(out)
			if len(resources) > 0 {
				app.Resources = resources
			}
		}

		// Checks (zero-downtime deploy)
		if out, err := r.Runner.Run("checks:report", appName); err == nil {
			checks := &schema.ChecksConfig{}
			if disabled := parseReportField(out, "Checks disabled list"); disabled != "" {
				checks.Disabled = strings.Fields(disabled)
			}
			if skipped := parseReportField(out, "Checks skipped list"); skipped != "" {
				checks.Skipped = strings.Fields(skipped)
			}
			if wtr := parseReportField(out, "Checks wait to retire"); wtr != "" {
				fmt.Sscanf(wtr, "%d", &checks.WaitToRetire)
			}
			if len(checks.Disabled) > 0 || len(checks.Skipped) > 0 || checks.WaitToRetire > 0 {
				app.Checks = checks
			}
		}

		// Builder settings
		if out, err := r.Runner.Run("builder:report", appName); err == nil {
			builder := &schema.BuilderConfig{
				Selected: parseReportField(out, "Builder selected"),
				BuildDir: parseReportField(out, "Builder build dir"),
			}
			if builder.Selected != "" || builder.BuildDir != "" {
				app.Builder = builder
			}
		}

		// Registry settings
		if out, err := r.Runner.Run("registry:report", appName); err == nil {
			reg := &schema.RegistryConfig{
				Server:        parseReportField(out, "Registry server"),
				ImageRepo:     parseReportField(out, "Registry image repo"),
				PushOnRelease: parseReportField(out, "Registry push on release") == "true",
				PushExtraTags: parseReportField(out, "Registry push extra tags"),
			}
			if reg.Server != "" || reg.ImageRepo != "" || reg.PushOnRelease || reg.PushExtraTags != "" {
				app.Registry = reg
			}
		}

		// Maintenance mode
		if out, err := r.Runner.Run("maintenance:report", appName); err == nil {
			app.Maintenance = parseReportField(out, "Maintenance enabled") == "true"
		}

		// Process management
		if out, err := r.Runner.Run("ps:report", appName); err == nil {
			proc := &schema.ProcessConfig{
				RestartPolicy: parseReportField(out, "Ps restart policy"),
				ProcfilePath:  parseReportField(out, "Ps procfile path"),
			}
			if proc.RestartPolicy != "" || proc.ProcfilePath != "" {
				app.Process = proc
			}
		}

		// Log configuration
		if out, err := r.Runner.Run("logs:report", appName); err == nil {
			logs := &schema.LogConfig{
				MaxSize:       parseReportField(out, "Logs max size"),
				VectorImage:   parseReportField(out, "Logs vector image"),
				VectorSink:    parseReportField(out, "Logs vector sink"),
				AppLabelAlias: parseReportField(out, "Logs app label alias"),
			}
			if logs.MaxSize != "" || logs.VectorImage != "" || logs.VectorSink != "" || logs.AppLabelAlias != "" {
				app.Logs = logs
			}
		}

		// Scheduler configuration
		{
			sched := &schema.SchedulerConfig{}
			if out, err := r.Runner.Run("scheduler:report", appName); err == nil {
				sched.Selected = parseReportField(out, "Scheduler selected")
			}
			if out, err := r.Runner.Run("scheduler-docker-local:report", appName); err == nil {
				sched.DockerLocalInitProcess = parseReportField(out, "Scheduler docker local init process")
				sched.DockerLocalParallelScheduleCount = parseReportField(out, "Scheduler docker local parallel schedule count")
			}
			if sched.Selected != "" || sched.DockerLocalInitProcess != "" || sched.DockerLocalParallelScheduleCount != "" {
				app.Scheduler = sched
			}
		}

		// Buildpacks
		if out, err := r.Runner.Run("buildpacks:list", appName); err == nil {
			bps := parseBuildpacksList(out)
			if len(bps) > 0 {
				app.Buildpacks = bps
			}
		}

		// Builder sub-plugin properties
		if out, err := r.Runner.Run("builder-dockerfile:report", appName); err == nil {
			dfPath := parseReportField(out, "Builder dockerfile dockerfile path")
			if dfPath != "" {
				if app.Builder == nil {
					app.Builder = &schema.BuilderConfig{}
				}
				app.Builder.DockerfilePath = dfPath
			}
		}
		if out, err := r.Runner.Run("builder-pack:report", appName); err == nil {
			ptPath := parseReportField(out, "Builder pack projecttoml path")
			if ptPath != "" {
				if app.Builder == nil {
					app.Builder = &schema.BuilderConfig{}
				}
				app.Builder.PackProjecttomlPath = ptPath
			}
		}
		if out, err := r.Runner.Run("builder-nixpacks:report", appName); err == nil {
			npPath := parseReportField(out, "Builder nixpacks nixpackstoml path")
			if npPath != "" {
				if app.Builder == nil {
					app.Builder = &schema.BuilderConfig{}
				}
				app.Builder.NixpacksTomlPath = npPath
			}
		}
		if out, err := r.Runner.Run("builder-herokuish:report", appName); err == nil {
			allowed := parseReportField(out, "Builder herokuish allowed")
			if allowed != "" {
				if app.Builder == nil {
					app.Builder = &schema.BuilderConfig{}
				}
				app.Builder.HerokuishAllowed = allowed
			}
		}

		// Deploy locking
		if _, err := r.Runner.Run("apps:locked", appName); err == nil {
			app.Locked = true
		}

		// Nginx template (via FileRunner)
		if r.FileRunner != nil {
			sigilPath := fmt.Sprintf("/home/dokku/%s/nginx.conf.sigil", appName)
			if content, err := r.FileRunner.ReadFile(sigilPath); err == nil && content != "" {
				app.NginxTemplate = content
			}
		}

		// App.json (healthchecks + cron + scripts)
		if r.FileRunner != nil {
			appJSONPath := fmt.Sprintf("/home/dokku/%s/app.json", appName)
			if content, err := r.FileRunner.ReadFile(appJSONPath); err == nil && content != "" {
				app.Healthchecks, app.Cron, app.Scripts = parseAppJSON(content)
			}
		}

		// LetsEncrypt
		_, err := r.Runner.Run("letsencrypt:active", appName)
		app.LetsEncrypt = (err == nil)

		// Links — check each service to see if it's linked to this app
		links := map[string]string{}
		for svcType, names := range allServices {
			for _, svcName := range names {
				_, err := r.Runner.Run(svcType+":linked", svcName, appName)
				if err == nil {
					links[svcType] = svcName
				}
			}
		}
		if len(links) > 0 {
			app.Links = links
		}

		df.Apps[appName] = app
	}

	return df, nil
}
