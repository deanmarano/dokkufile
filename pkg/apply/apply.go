package apply

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
)

// Executor applies a plan by shelling out to dokku commands.
type Executor struct {
	Runner     state.CommandRunner
	FileRunner state.FileRunner
	DryRun     bool
	// EnvGetter reads environment variables for secrets. Defaults to os.Getenv.
	EnvGetter func(string) string
	// CheckpointDir is the directory to store checkpoint files. Empty disables checkpointing.
	CheckpointDir string
	// DokkufilePath is the path to the Dokkufile, used for checkpoint hash validation.
	DokkufilePath string
}

func (e *Executor) getEnv(key string) string {
	if e.EnvGetter != nil {
		return e.EnvGetter(key)
	}
	return os.Getenv(key)
}

// Execute runs each step in the plan, using the desired and actual state for context.
// If CheckpointDir is set, completed steps are saved on failure so that a subsequent
// Execute call can resume from where it left off.
func (e *Executor) Execute(p *plan.Plan, desired, actual *schema.Dokkufile) error {
	// Load checkpoint if available
	completedKeys, dokkufileHash, err := e.loadResumeState()
	if err != nil {
		return err
	}

	var completed []CompletedStep

	for _, step := range p.Steps {
		key := StepKey(step)

		// Skip steps completed in a previous run
		if completedKeys[key] {
			fmt.Printf("Skipping (already done): %s\n", describeStep(step))
			continue
		}

		cmds, err := e.commandsForStep(step, desired, actual)
		if err != nil {
			e.saveFailureCheckpoint(completed, dokkufileHash, key, err)
			return fmt.Errorf("planning commands for %s: %w", describeStep(step), err)
		}

		if err := e.runCommands(cmds); err != nil {
			e.saveFailureCheckpoint(completed, dokkufileHash, key, err)
			return err
		}

		completed = append(completed, CompletedStep{
			StepKey:  key,
			Commands: flattenCommands(cmds),
		})
	}

	// Success — clear checkpoint if one exists
	if e.CheckpointDir != "" {
		if err := clearCheckpoint(e.CheckpointDir); err != nil {
			fmt.Printf("Warning: failed to clear checkpoint: %v\n", err)
		}
	}
	return nil
}

// runCommands executes a list of commands, handling synthetic file-write commands and dry-run mode.
func (e *Executor) runCommands(cmds [][]string) error {
	for _, cmd := range cmds {
		// Handle synthetic file-write commands
		if len(cmd) >= 3 && cmd[0] == "__write-app-json" {
			if e.DryRun {
				fmt.Printf("[dry-run] write /home/dokku/%s/app.json\n", cmd[1])
				continue
			}
			if e.FileRunner != nil {
				path := fmt.Sprintf("/home/dokku/%s/app.json", cmd[1])
				fmt.Printf("Writing: %s\n", path)
				if err := e.FileRunner.WriteFile(path, []byte(cmd[2]), 0644); err != nil {
					return fmt.Errorf("writing app.json: %w", err)
				}
			}
			continue
		}
		if e.DryRun {
			fmt.Printf("[dry-run] dokku %s\n", strings.Join(cmd, " "))
			continue
		}
		fmt.Printf("Running: dokku %s\n", strings.Join(cmd, " "))
		out, err := e.Runner.Run(cmd...)
		if err != nil {
			if isAlreadyExistsError(cmd, out) {
				fmt.Printf("  (already exists, continuing)\n")
				continue
			}
			return fmt.Errorf("dokku %s: %s: %w", strings.Join(cmd, " "), out, err)
		}
	}
	return nil
}

// isAlreadyExistsError returns true if the error from a create command indicates
// the resource already exists. This allows resumed applies to skip past creates
// that succeeded in a previous partial run.
func isAlreadyExistsError(cmd []string, output string) bool {
	if len(cmd) < 2 {
		return false
	}
	lowerOut := strings.ToLower(output)
	isCreate := cmd[0] == "apps:create" ||
		strings.HasSuffix(cmd[0], ":create") ||
		cmd[0] == "mail:create" ||
		cmd[0] == "auth:create" ||
		cmd[0] == "auth:frontend:create"
	return isCreate && strings.Contains(lowerOut, "already exists")
}

// loadResumeState loads checkpoint data if checkpointing is enabled and a valid checkpoint exists.
// Returns the set of completed step keys, the dokkufile hash (for saving), and any error.
func (e *Executor) loadResumeState() (map[string]bool, string, error) {
	completedKeys := map[string]bool{}
	var dokkufileHash string

	if e.CheckpointDir == "" {
		return completedKeys, dokkufileHash, nil
	}

	// Compute current dokkufile hash
	if e.DokkufilePath != "" {
		var err error
		dokkufileHash, err = hashFile(e.DokkufilePath)
		if err != nil {
			return completedKeys, dokkufileHash, fmt.Errorf("hashing dokkufile: %w", err)
		}
	}

	cp, err := loadCheckpoint(e.CheckpointDir)
	if err != nil {
		return completedKeys, dokkufileHash, fmt.Errorf("loading checkpoint: %w", err)
	}
	if cp == nil {
		return completedKeys, dokkufileHash, nil
	}

	// Validate checkpoint against current dokkufile
	if dokkufileHash != "" && cp.DokkufileHash != dokkufileHash {
		fmt.Println("Warning: Dokkufile changed since last run. Ignoring checkpoint.")
		if err := clearCheckpoint(e.CheckpointDir); err != nil {
			fmt.Printf("Warning: failed to clear stale checkpoint: %v\n", err)
		}
		return completedKeys, dokkufileHash, nil
	}

	for _, cs := range cp.CompletedSteps {
		completedKeys[cs.StepKey] = true
	}
	fmt.Printf("Resuming from checkpoint (%d steps already completed)\n", len(cp.CompletedSteps))

	return completedKeys, dokkufileHash, nil
}

// saveFailureCheckpoint saves a checkpoint recording completed steps and the failed step.
func (e *Executor) saveFailureCheckpoint(completed []CompletedStep, dokkufileHash, failedKey string, failErr error) {
	if e.CheckpointDir == "" || e.DryRun {
		return
	}
	cp := newCheckpoint(dokkufileHash)
	cp.CompletedSteps = completed
	cp.FailedStep = &StepRecord{StepKey: failedKey, Error: failErr.Error()}
	if err := saveCheckpoint(e.CheckpointDir, cp); err != nil {
		fmt.Printf("Warning: failed to save checkpoint: %v\n", err)
		return
	}
	fmt.Printf("Checkpoint saved. Re-run apply to resume from step %d.\n", len(completed)+1)
}

// flattenCommands converts a slice of command arg slices into a slice of command strings.
func flattenCommands(cmds [][]string) []string {
	result := make([]string, len(cmds))
	for i, cmd := range cmds {
		result[i] = strings.Join(cmd, " ")
	}
	return result
}

// commandsForStep translates a plan step into one or more dokku command arg slices.
func (e *Executor) commandsForStep(s plan.Step, desired, actual *schema.Dokkufile) ([][]string, error) {
	switch s.Action {
	case plan.CreateService:
		return e.createServiceCommands(s, desired)

	case plan.UpdateService:
		return e.updateServiceCommands(s, desired)

	case plan.DestroyService:
		return [][]string{{s.ServiceType + ":destroy", s.Service, "--force"}}, nil

	case plan.CreateApp:
		return e.createAppCommands(s.App, desired)

	case plan.DestroyApp:
		return [][]string{{"apps:destroy", s.App, "--force"}}, nil

	case plan.UpdateApp:
		return e.updateAppCommands(s, desired, actual)

	case plan.CreateMailService:
		return e.createMailServiceCommands(s.Service, desired)

	case plan.DestroyMailService:
		return [][]string{{"mail:destroy", s.Service, "--force"}}, nil

	case plan.UpdateMailService:
		return e.updateMailServiceCommands(s.Service, desired)

	case plan.CreateAuthDirectory:
		return e.createAuthDirectoryCommands(s.Service, desired)

	case plan.DestroyAuthDirectory:
		return [][]string{{"auth:destroy", s.Service, "--force"}}, nil

	case plan.UpdateAuthDirectory:
		return e.updateAuthDirectoryCommands(s.Service, desired)

	case plan.CreateAuthFrontend:
		return e.createAuthFrontendCommands(s.Service, desired)

	case plan.DestroyAuthFrontend:
		return [][]string{{"auth:frontend:destroy", s.Service, "--force"}}, nil

	case plan.UpdateAuthFrontend:
		return e.updateAuthFrontendCommands(s.Service, desired, actual)

	case plan.InstallPlugin:
		return e.installPluginCommands(s.Service, desired)

	case plan.UninstallPlugin:
		return [][]string{{"plugin:uninstall", s.Service}}, nil

	case plan.UpdatePlugin:
		return e.installPluginCommands(s.Service, desired)

	case plan.UpdateGlobal:
		return e.globalCommands(s, desired)

	default:
		return nil, fmt.Errorf("unknown action: %s", s.Action)
	}
}

// createAppCommands generates all commands to create and fully configure a new app.
func (e *Executor) createAppCommands(appName string, desired *schema.Dokkufile) ([][]string, error) {
	app, ok := desired.Apps[appName]
	if !ok {
		return nil, fmt.Errorf("app %q not found in desired state", appName)
	}

	var cmds [][]string
	cmds = append(cmds, []string{"apps:create", appName})

	// Set domains (before deploy so nginx config is correct)
	if len(app.Domains) > 0 {
		cmds = append(cmds, append([]string{"domains:set", appName}, app.Domains...))
	}

	// Set env
	if len(app.Env) > 0 {
		cmds = append(cmds, configSetArgs(appName, app.Env))
	}

	// Set secrets from host environment
	if len(app.Secrets) > 0 {
		secretEnv := e.resolveSecrets(app.Secrets)
		if len(secretEnv) > 0 {
			cmds = append(cmds, configSetArgs(appName, secretEnv))
		}
	}

	// Set ports
	if len(app.Ports) > 0 {
		cmds = append(cmds, portsSetArgs(appName, app.Ports))
	}

	// Set storage
	for _, mount := range app.Storage {
		cmds = append(cmds, []string{"storage:mount", appName, mount})
	}

	// Set docker options
	cmds = append(cmds, dockerOptionsAddCmds(appName, "build", app.DockerOptions.Build)...)
	cmds = append(cmds, dockerOptionsAddCmds(appName, "deploy", app.DockerOptions.Deploy)...)
	cmds = append(cmds, dockerOptionsAddCmds(appName, "run", app.DockerOptions.Run)...)

	// Set scale
	if len(app.Scale) > 0 {
		cmds = append(cmds, scaleSetArgs(appName, app.Scale))
	}

	// Link services (before deploy so DATABASE_URL etc. are available)
	for svcType, svcName := range app.Links {
		cmds = append(cmds, []string{svcType + ":link", svcName, appName, "--no-restart"})
	}

	// Scheduler config (before deploy so init_process etc. take effect)
	if app.Scheduler != nil {
		cmds = append(cmds, schedulerCommands(appName, app.Scheduler)...)
	}

	// App.json (healthchecks, cron, scripts — before deploy so checks take effect)
	if len(app.Healthchecks) > 0 || len(app.Cron) > 0 || app.Scripts != nil {
		ajCmds, err := e.appJsonCommands(appName, app)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, ajCmds...)
	}

	// Deploy image (must come after all config so the container starts correctly)
	if app.Image != "" {
		cmds = append(cmds, []string{"git:from-image", appName, app.Image})
	}

	// DNS (after deploy, before letsencrypt)
	if app.DNS {
		cmds = append(cmds, []string{"dns:apps:enable", appName})
		cmds = append(cmds, []string{"dns:apps:sync", appName})
	}

	// Letsencrypt email (before enable)
	if app.LetsEncryptEmail != "" {
		cmds = append(cmds, []string{"letsencrypt:set", appName, "email", app.LetsEncryptEmail})
	}

	// Letsencrypt (after DNS so HTTP-01 challenge works)
	if app.LetsEncrypt {
		cmds = append(cmds, []string{"letsencrypt:enable", appName})
	}

	// Git config
	if app.Git != nil {
		cmds = append(cmds, gitConfigCommands(appName, app.Git)...)
	}

	// Network config
	if app.Network != nil {
		cmds = append(cmds, networkConfigCommands(appName, app.Network)...)
	}

	// Nginx config
	if app.Nginx != nil {
		cmds = append(cmds, nginxConfigCommands(appName, app.Nginx)...)
	}

	// Proxy config
	if app.Proxy != nil {
		cmds = append(cmds, proxyConfigCommands(appName, app.Proxy)...)
	}

	// Resources
	if len(app.Resources) > 0 {
		cmds = append(cmds, resourceCommands(appName, app.Resources)...)
	}

	// Checks
	if app.Checks != nil {
		cmds = append(cmds, checksCommands(appName, app.Checks)...)
	}

	// Builder
	if app.Builder != nil {
		cmds = append(cmds, builderCommands(appName, app.Builder)...)
	}

	// Registry
	if app.Registry != nil {
		cmds = append(cmds, registryCommands(appName, app.Registry)...)
	}

	// Maintenance
	if app.Maintenance {
		cmds = append(cmds, []string{"maintenance:enable", appName})
	}

	// Process management
	if app.Process != nil {
		cmds = append(cmds, processCommands(appName, app.Process)...)
	}

	// Locked
	if app.Locked {
		cmds = append(cmds, []string{"apps:lock", appName})
	}

	// Logs
	if app.Logs != nil {
		cmds = append(cmds, logCommands(appName, app.Logs)...)
	}

	// Buildpacks
	if len(app.Buildpacks) > 0 {
		cmds = append(cmds, buildpacksCommands(appName, app.Buildpacks)...)
	}

	// Mail link
	if app.Mail != "" {
		cmds = append(cmds, []string{"mail:link", app.Mail, appName})
	}

	// Auth link + protection
	if app.Auth != nil {
		if app.Auth.Directory != "" {
			cmds = append(cmds, []string{"auth:link", app.Auth.Directory, appName})
		}
		if app.Auth.Protected != "" {
			cmds = append(cmds, []string{"auth:frontend:protect", app.Auth.Protected, appName})
		}
	}

	return cmds, nil
}

// updateAppCommands generates commands for a single field update on an existing app.
func (e *Executor) updateAppCommands(s plan.Step, desired, actual *schema.Dokkufile) ([][]string, error) {
	dApp := desired.Apps[s.App]
	aApp := actual.Apps[s.App]

	switch s.Field {
	case "image":
		return [][]string{{"git:from-image", s.App, dApp.Image}}, nil

	case "domains":
		return [][]string{append([]string{"domains:set", s.App}, dApp.Domains...)}, nil

	case "env":
		return envUpdateCommands(s.App, dApp.Env, aApp.Env), nil

	case "links":
		return linkUpdateCommands(s.App, dApp.Links, aApp.Links), nil

	case "ports":
		var cmds [][]string
		cmds = append(cmds, []string{"ports:clear", s.App})
		if len(dApp.Ports) > 0 {
			cmds = append(cmds, portsSetArgs(s.App, dApp.Ports))
		}
		return cmds, nil

	case "storage":
		return storageUpdateCommands(s.App, dApp.Storage, aApp.Storage), nil

	case "docker_options":
		return dockerOptionsUpdateCommands(s.App, dApp.DockerOptions, aApp.DockerOptions), nil

	case "scale":
		return [][]string{scaleSetArgs(s.App, dApp.Scale)}, nil

	case "letsencrypt":
		var cmds [][]string
		if dApp.LetsEncryptEmail != "" {
			cmds = append(cmds, []string{"letsencrypt:set", s.App, "email", dApp.LetsEncryptEmail})
		}
		if dApp.LetsEncrypt {
			cmds = append(cmds, []string{"letsencrypt:enable", s.App})
		} else {
			cmds = append(cmds, []string{"letsencrypt:disable", s.App})
		}
		return cmds, nil

	case "letsencrypt_email":
		if dApp.LetsEncryptEmail != "" {
			return [][]string{{"letsencrypt:set", s.App, "email", dApp.LetsEncryptEmail}}, nil
		}
		return [][]string{{"letsencrypt:set", s.App, "email", ""}}, nil

	case "dns":
		if dApp.DNS {
			return [][]string{
				{"dns:apps:enable", s.App},
				{"dns:apps:sync", s.App},
			}, nil
		}
		return [][]string{{"dns:apps:disable", s.App}}, nil

	case "git":
		if dApp.Git == nil {
			return nil, nil
		}
		return gitConfigCommands(s.App, dApp.Git), nil

	case "network":
		if dApp.Network == nil {
			return nil, nil
		}
		return networkConfigCommands(s.App, dApp.Network), nil

	case "nginx":
		if dApp.Nginx == nil {
			return nil, nil
		}
		return nginxConfigCommands(s.App, dApp.Nginx), nil

	case "proxy":
		if dApp.Proxy == nil {
			return nil, nil
		}
		return proxyConfigCommands(s.App, dApp.Proxy), nil

	case "ssl":
		return e.sslCommands(s.App, dApp.SSL, aApp.SSL)

	case "healthchecks", "cron":
		return e.appJsonCommands(s.App, dApp)

	case "nginx_template":
		return e.nginxTemplateCommands(s.App, dApp.NginxTemplate)

	case "resources":
		return resourceCommands(s.App, dApp.Resources), nil

	case "checks":
		return checksCommands(s.App, dApp.Checks), nil

	case "builder":
		if dApp.Builder == nil {
			return nil, nil
		}
		return builderCommands(s.App, dApp.Builder), nil

	case "registry":
		if dApp.Registry == nil {
			return nil, nil
		}
		return registryCommands(s.App, dApp.Registry), nil

	case "maintenance":
		if dApp.Maintenance {
			return [][]string{{"maintenance:enable", s.App}}, nil
		}
		return [][]string{{"maintenance:disable", s.App}}, nil

	case "scripts":
		return e.appJsonCommands(s.App, dApp)

	case "locked":
		if dApp.Locked {
			return [][]string{{"apps:lock", s.App}}, nil
		}
		return [][]string{{"apps:unlock", s.App}}, nil

	case "process":
		if dApp.Process == nil {
			return nil, nil
		}
		return processCommands(s.App, dApp.Process), nil

	case "logs":
		if dApp.Logs == nil {
			return nil, nil
		}
		return logCommands(s.App, dApp.Logs), nil

	case "scheduler":
		if dApp.Scheduler == nil {
			return nil, nil
		}
		return schedulerCommands(s.App, dApp.Scheduler), nil

	case "buildpacks":
		return buildpacksCommands(s.App, dApp.Buildpacks), nil

	case "secrets":
		secretEnv := e.resolveSecrets(dApp.Secrets)
		if len(secretEnv) > 0 {
			return [][]string{configSetArgs(s.App, secretEnv)}, nil
		}
		return nil, nil

	case "mail":
		return mailLinkCommands(s.App, dApp.Mail, aApp.Mail), nil

	case "auth":
		return authLinkCommands(s.App, dApp.Auth, aApp.Auth), nil

	default:
		return nil, fmt.Errorf("unknown field: %s", s.Field)
	}
}

func gitConfigCommands(appName string, git *schema.GitConfig) [][]string {
	var cmds [][]string
	if git.Branch != "" {
		cmds = append(cmds, []string{"git:set", appName, "deploy-branch", git.Branch})
	}
	if git.KeepGitDir {
		cmds = append(cmds, []string{"git:set", appName, "keep-git-dir", "true"})
	} else {
		cmds = append(cmds, []string{"git:set", appName, "keep-git-dir", "false"})
	}
	if git.Repo != "" {
		cmd := []string{"git:sync", "--build", appName, git.Repo}
		if git.Branch != "" {
			cmd = append(cmd, git.Branch)
		}
		cmds = append(cmds, cmd)
	}
	return cmds
}

func networkConfigCommands(appName string, net *schema.NetworkConfig) [][]string {
	var cmds [][]string
	props := []struct {
		name  string
		value string
	}{
		{"attach-post-create", net.AttachPostCreate},
		{"attach-post-deploy", net.AttachPostDeploy},
		{"bind-all-interfaces", strconv.FormatBool(net.BindAllInterfaces)},
		{"initial-network", net.InitialNetwork},
		{"static-web-listener", net.StaticWebListener},
		{"tld", net.TLD},
	}
	for _, p := range props {
		if p.value != "" {
			cmds = append(cmds, []string{"network:set", appName, p.name, p.value})
		}
	}
	return cmds
}

func nginxConfigCommands(appName string, nginx *schema.NginxConfig) [][]string {
	var cmds [][]string
	cmds = append(cmds, []string{"nginx:set", appName, "hsts", strconv.FormatBool(nginx.HSTS)})
	cmds = append(cmds, []string{"nginx:set", appName, "hsts-include-subdomains", strconv.FormatBool(nginx.HSTSIncludeSubdomains)})
	if nginx.HSTSMaxAge > 0 {
		cmds = append(cmds, []string{"nginx:set", appName, "hsts-max-age", strconv.Itoa(nginx.HSTSMaxAge)})
	}
	cmds = append(cmds, []string{"nginx:set", appName, "hsts-preload", strconv.FormatBool(nginx.HSTSPreload)})
	// Extended properties
	for _, k := range sortedKeys(nginx.Properties) {
		cmds = append(cmds, []string{"nginx:set", appName, k, nginx.Properties[k]})
	}
	return cmds
}

func proxyConfigCommands(appName string, proxy *schema.ProxyConfig) [][]string {
	var cmds [][]string
	// Only emit enable/disable if explicitly true.
	// Zero value (false) means "don't change" — use maintenance mode to disable proxy.
	if proxy.Enabled {
		cmds = append(cmds, []string{"proxy:enable", appName})
	}
	if proxy.Type != "" {
		cmds = append(cmds, []string{"proxy:set", appName, proxy.Type})
	}
	// Caddy properties
	for _, k := range sortedKeys(proxy.Caddy) {
		cmds = append(cmds, []string{"caddy:set", appName, k, proxy.Caddy[k]})
	}
	// HAProxy properties
	for _, k := range sortedKeys(proxy.HAProxy) {
		cmds = append(cmds, []string{"haproxy:set", appName, k, proxy.HAProxy[k]})
	}
	// Traefik properties
	for _, k := range sortedKeys(proxy.Traefik) {
		cmds = append(cmds, []string{"traefik:set", appName, k, proxy.Traefik[k]})
	}
	return cmds
}

func (e *Executor) sslCommands(appName string, desired, actual *schema.SSLConfig) ([][]string, error) {
	if desired == nil || (desired.CertFile == "" && desired.KeyFile == "") {
		// Remove SSL
		return [][]string{{"certs:remove", appName}}, nil
	}

	// Read cert and key files, build tar, pipe to certs:add via StdinRunner.
	if e.FileRunner == nil {
		return [][]string{{"certs:add", appName}}, nil
	}

	certContent, err := e.FileRunner.ReadFile(desired.CertFile)
	if err != nil {
		return nil, fmt.Errorf("reading cert file %s: %w", desired.CertFile, err)
	}
	keyContent, err := e.FileRunner.ReadFile(desired.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("reading key file %s: %w", desired.KeyFile, err)
	}

	tarBuf, err := buildCertTar([]byte(certContent), []byte(keyContent))
	if err != nil {
		return nil, fmt.Errorf("building cert tar: %w", err)
	}

	// Use StdinRunner if available to pipe the tar
	if sr, ok := e.Runner.(state.StdinRunner); ok {
		if e.DryRun {
			fmt.Printf("[dry-run] dokku certs:add %s < <cert-tar>\n", appName)
		} else {
			fmt.Printf("Running: dokku certs:add %s < <cert-tar>\n", appName)
			out, err := sr.RunWithStdin(tarBuf, "certs:add", appName)
			if err != nil {
				return nil, fmt.Errorf("dokku certs:add %s: %s: %w", appName, out, err)
			}
		}
		return nil, nil // already executed directly
	}

	// Fallback: just emit the command name
	return [][]string{{"certs:add", appName}}, nil
}

// buildCertTar creates an in-memory tar archive containing server.crt and server.key.
func buildCertTar(cert, key []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	files := []struct {
		name string
		data []byte
	}{
		{"server.crt", cert},
		{"server.key", key},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0600,
			Size: int64(len(f.data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.data); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func (e *Executor) appJsonCommands(appName string, app schema.App) ([][]string, error) {
	appJSON := buildAppJSON(app)
	jsonBytes, err := json.Marshal(appJSON)
	if err != nil {
		return nil, fmt.Errorf("marshaling app.json: %w", err)
	}
	// Return a synthetic command that the executor will intercept to write the file.
	// We can't write directly here because during CreateApp, the /home/dokku/<app>/
	// directory may not exist yet (apps:create hasn't run).
	return [][]string{{"__write-app-json", appName, string(jsonBytes)}}, nil
}

func buildAppJSON(app schema.App) map[string]interface{} {
	result := map[string]interface{}{}

	if len(app.Healthchecks) > 0 {
		result["healthchecks"] = app.Healthchecks
	}

	if len(app.Cron) > 0 {
		cron := make([]map[string]string, len(app.Cron))
		for i, c := range app.Cron {
			cron[i] = map[string]string{
				"command":  c.Command,
				"schedule": c.Schedule,
			}
		}
		result["cron"] = cron
	}

	if app.Scripts != nil {
		dokkuScripts := map[string]string{}
		if app.Scripts.Predeploy != "" {
			dokkuScripts["predeploy"] = app.Scripts.Predeploy
		}
		if app.Scripts.Postdeploy != "" {
			dokkuScripts["postdeploy"] = app.Scripts.Postdeploy
		}
		if len(dokkuScripts) > 0 {
			result["scripts"] = map[string]interface{}{
				"dokku": dokkuScripts,
			}
		}
	}

	return result
}

func (e *Executor) nginxTemplateCommands(appName, template string) ([][]string, error) {
	if e.FileRunner != nil {
		path := fmt.Sprintf("/home/dokku/%s/nginx.conf.sigil", appName)
		if template == "" {
			// Template removed — we'd delete the file, but just rebuild config
		} else {
			if err := e.FileRunner.WriteFile(path, []byte(template), 0644); err != nil {
				return nil, fmt.Errorf("writing nginx template: %w", err)
			}
		}
	}
	return [][]string{{"nginx:build-config", appName}}, nil
}

// Mail service commands
func (e *Executor) createMailServiceCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	svc, ok := desired.MailServices[name]
	if !ok {
		return nil, fmt.Errorf("mail service %q not found in desired state", name)
	}
	var cmds [][]string
	cmds = append(cmds, []string{"mail:create", name})
	if svc.Provider != "" {
		cmds = append(cmds, []string{"mail:provider:set", name, svc.Provider})
	}
	for _, k := range sortedKeys(svc.Config) {
		cmds = append(cmds, []string{"mail:provider:config", name, fmt.Sprintf("%s=%s", k, svc.Config[k])})
	}
	if svc.Provider != "" {
		cmds = append(cmds, []string{"mail:provider:apply", name})
	}
	return cmds, nil
}

func (e *Executor) updateMailServiceCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	svc, ok := desired.MailServices[name]
	if !ok {
		return nil, fmt.Errorf("mail service %q not found in desired state", name)
	}
	var cmds [][]string
	if svc.Provider != "" {
		cmds = append(cmds, []string{"mail:provider:set", name, svc.Provider})
	}
	for _, k := range sortedKeys(svc.Config) {
		cmds = append(cmds, []string{"mail:provider:config", name, fmt.Sprintf("%s=%s", k, svc.Config[k])})
	}
	if svc.Provider != "" {
		cmds = append(cmds, []string{"mail:provider:apply", name})
	}
	return cmds, nil
}

// Auth directory commands
func (e *Executor) createAuthDirectoryCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	dir, ok := desired.AuthDirectories[name]
	if !ok {
		return nil, fmt.Errorf("auth directory %q not found in desired state", name)
	}
	var cmds [][]string
	cmds = append(cmds, []string{"auth:create", name})
	if dir.Provider != "" {
		cmds = append(cmds, []string{"auth:provider:set", name, dir.Provider})
	}
	for _, k := range sortedKeys(dir.Config) {
		cmds = append(cmds, []string{"auth:provider:config", name, fmt.Sprintf("%s=%s", k, dir.Config[k])})
	}
	if dir.Provider != "" {
		cmds = append(cmds, []string{"auth:provider:apply", name})
	}
	return cmds, nil
}

func (e *Executor) updateAuthDirectoryCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	dir, ok := desired.AuthDirectories[name]
	if !ok {
		return nil, fmt.Errorf("auth directory %q not found in desired state", name)
	}
	var cmds [][]string
	if dir.Provider != "" {
		cmds = append(cmds, []string{"auth:provider:set", name, dir.Provider})
	}
	for _, k := range sortedKeys(dir.Config) {
		cmds = append(cmds, []string{"auth:provider:config", name, fmt.Sprintf("%s=%s", k, dir.Config[k])})
	}
	if dir.Provider != "" {
		cmds = append(cmds, []string{"auth:provider:apply", name})
	}
	return cmds, nil
}

// Auth frontend commands
func (e *Executor) createAuthFrontendCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	fe, ok := desired.AuthFrontends[name]
	if !ok {
		return nil, fmt.Errorf("auth frontend %q not found in desired state", name)
	}
	var cmds [][]string
	cmds = append(cmds, []string{"auth:frontend:create", name})
	if fe.Provider != "" {
		cmds = append(cmds, []string{"auth:frontend:provider:set", name, fe.Provider})
	}
	if fe.Directory != "" {
		cmds = append(cmds, []string{"auth:frontend:use-directory", name, fe.Directory})
	}
	for _, k := range sortedKeys(fe.Config) {
		cmds = append(cmds, []string{"auth:frontend:config", name, fmt.Sprintf("%s=%s", k, fe.Config[k])})
	}
	if fe.Provider != "" {
		cmds = append(cmds, []string{"auth:frontend:apply", name})
	}
	for _, app := range fe.ProtectedApps {
		cmds = append(cmds, []string{"auth:frontend:protect", name, app})
	}
	if fe.OIDCEnabled {
		cmds = append(cmds, []string{"auth:oidc:enable", name})
		for _, client := range fe.OIDCClients {
			cmd := []string{"auth:oidc:add-client", name, client.ID}
			if client.Secret != "" {
				cmd = append(cmd, client.Secret)
			}
			if client.RedirectURI != "" {
				cmd = append(cmd, client.RedirectURI)
			}
			cmds = append(cmds, cmd)
		}
	}
	return cmds, nil
}

func (e *Executor) updateAuthFrontendCommands(name string, desired, actual *schema.Dokkufile) ([][]string, error) {
	fe, ok := desired.AuthFrontends[name]
	if !ok {
		return nil, fmt.Errorf("auth frontend %q not found in desired state", name)
	}
	var cmds [][]string

	if fe.Provider != "" {
		cmds = append(cmds, []string{"auth:frontend:provider:set", name, fe.Provider})
	}
	if fe.Directory != "" {
		cmds = append(cmds, []string{"auth:frontend:use-directory", name, fe.Directory})
	}
	for _, k := range sortedKeys(fe.Config) {
		cmds = append(cmds, []string{"auth:frontend:config", name, fmt.Sprintf("%s=%s", k, fe.Config[k])})
	}
	if fe.Provider != "" {
		cmds = append(cmds, []string{"auth:frontend:apply", name})
	}

	// Unprotect removed apps, protect new apps
	actualFE := actual.AuthFrontends[name]
	actualProtected := toSet(actualFE.ProtectedApps)
	desiredProtected := toSet(fe.ProtectedApps)
	for _, app := range actualFE.ProtectedApps {
		if !desiredProtected[app] {
			cmds = append(cmds, []string{"auth:frontend:unprotect", name, app})
		}
	}
	for _, app := range fe.ProtectedApps {
		if !actualProtected[app] {
			cmds = append(cmds, []string{"auth:frontend:protect", name, app})
		}
	}

	// OIDC
	if fe.OIDCEnabled && !actualFE.OIDCEnabled {
		cmds = append(cmds, []string{"auth:oidc:enable", name})
	} else if !fe.OIDCEnabled && actualFE.OIDCEnabled {
		cmds = append(cmds, []string{"auth:oidc:disable", name})
	}

	if fe.OIDCEnabled {
		// Remove old clients not in desired
		actualClients := map[string]bool{}
		for _, c := range actualFE.OIDCClients {
			actualClients[c.ID] = true
		}
		desiredClients := map[string]bool{}
		for _, c := range fe.OIDCClients {
			desiredClients[c.ID] = true
		}
		for _, c := range actualFE.OIDCClients {
			if !desiredClients[c.ID] {
				cmds = append(cmds, []string{"auth:oidc:remove-client", name, c.ID})
			}
		}
		for _, c := range fe.OIDCClients {
			if !actualClients[c.ID] {
				cmd := []string{"auth:oidc:add-client", name, c.ID}
				if c.Secret != "" {
					cmd = append(cmd, c.Secret)
				}
				if c.RedirectURI != "" {
					cmd = append(cmd, c.RedirectURI)
				}
				cmds = append(cmds, cmd)
			}
		}
	}

	return cmds, nil
}

// globalCommands generates commands for global setting updates.
func (e *Executor) globalCommands(s plan.Step, desired *schema.Dokkufile) ([][]string, error) {
	g := desired.Global
	if g == nil {
		g = &schema.GlobalConfig{}
	}

	switch s.Field {
	case "domains":
		if len(g.Domains) > 0 {
			return [][]string{append([]string{"domains:set", "--global"}, g.Domains...)}, nil
		}
		return [][]string{{"domains:clear", "--global"}}, nil

	case "nginx":
		if g.Nginx == nil {
			return nil, nil
		}
		return globalNginxCommands(g.Nginx), nil

	case "proxy":
		if g.Proxy == nil {
			return nil, nil
		}
		var cmds [][]string
		if g.Proxy.Type != "" {
			cmds = append(cmds, []string{"proxy:set", "--global", g.Proxy.Type})
		}
		return cmds, nil

	case "network":
		if g.Network == nil {
			return nil, nil
		}
		return globalNetworkCommands(g.Network), nil

	case "builder":
		if g.Builder == nil {
			return nil, nil
		}
		return globalBuilderCommands(g.Builder), nil

	case "registry":
		if g.Registry == nil {
			return nil, nil
		}
		return globalRegistryCommands(g.Registry), nil

	case "logs":
		if g.Logs == nil {
			return nil, nil
		}
		return globalLogCommands(g.Logs), nil

	case "scheduler":
		if g.Scheduler == nil {
			return nil, nil
		}
		var cmds [][]string
		if g.Scheduler.Selected != "" {
			cmds = append(cmds, []string{"scheduler:set", "--global", "selected", g.Scheduler.Selected})
		}
		return cmds, nil

	default:
		return nil, fmt.Errorf("unknown global field: %s", s.Field)
	}
}

func globalNginxCommands(nginx *schema.NginxConfig) [][]string {
	var cmds [][]string
	cmds = append(cmds, []string{"nginx:set", "--global", "hsts", strconv.FormatBool(nginx.HSTS)})
	cmds = append(cmds, []string{"nginx:set", "--global", "hsts-include-subdomains", strconv.FormatBool(nginx.HSTSIncludeSubdomains)})
	if nginx.HSTSMaxAge > 0 {
		cmds = append(cmds, []string{"nginx:set", "--global", "hsts-max-age", strconv.Itoa(nginx.HSTSMaxAge)})
	}
	cmds = append(cmds, []string{"nginx:set", "--global", "hsts-preload", strconv.FormatBool(nginx.HSTSPreload)})
	for _, k := range sortedKeys(nginx.Properties) {
		cmds = append(cmds, []string{"nginx:set", "--global", k, nginx.Properties[k]})
	}
	return cmds
}

func globalNetworkCommands(net *schema.NetworkConfig) [][]string {
	var cmds [][]string
	props := []struct {
		name  string
		value string
	}{
		{"attach-post-create", net.AttachPostCreate},
		{"attach-post-deploy", net.AttachPostDeploy},
		{"bind-all-interfaces", strconv.FormatBool(net.BindAllInterfaces)},
		{"initial-network", net.InitialNetwork},
		{"static-web-listener", net.StaticWebListener},
		{"tld", net.TLD},
	}
	for _, p := range props {
		if p.value != "" {
			cmds = append(cmds, []string{"network:set", "--global", p.name, p.value})
		}
	}
	return cmds
}

func globalBuilderCommands(builder *schema.BuilderConfig) [][]string {
	var cmds [][]string
	if builder.Selected != "" {
		cmds = append(cmds, []string{"builder:set", "--global", "selected", builder.Selected})
	}
	if builder.BuildDir != "" {
		cmds = append(cmds, []string{"builder:set", "--global", "build-dir", builder.BuildDir})
	}
	return cmds
}

func globalRegistryCommands(reg *schema.RegistryConfig) [][]string {
	var cmds [][]string
	if reg.Server != "" {
		cmds = append(cmds, []string{"registry:set", "--global", "server", reg.Server})
	}
	if reg.ImageRepo != "" {
		cmds = append(cmds, []string{"registry:set", "--global", "image-repo", reg.ImageRepo})
	}
	if reg.PushOnRelease {
		cmds = append(cmds, []string{"registry:set", "--global", "push-on-release", "true"})
	}
	if reg.PushExtraTags != "" {
		cmds = append(cmds, []string{"registry:set", "--global", "push-extra-tags", reg.PushExtraTags})
	}
	return cmds
}

func globalLogCommands(logs *schema.LogConfig) [][]string {
	var cmds [][]string
	if logs.MaxSize != "" {
		cmds = append(cmds, []string{"logs:set", "--global", "max-size", logs.MaxSize})
	}
	if logs.VectorImage != "" {
		cmds = append(cmds, []string{"logs:set", "--global", "vector-image", logs.VectorImage})
	}
	if logs.VectorSink != "" {
		cmds = append(cmds, []string{"logs:set", "--global", "vector-sink", logs.VectorSink})
	}
	return cmds
}

// resolveSecrets reads secret values from the host environment.
func (e *Executor) resolveSecrets(secrets []string) map[string]string {
	result := map[string]string{}
	for _, key := range secrets {
		val := e.getEnv(key)
		if val != "" {
			result[key] = val
		}
	}
	return result
}

// mailLinkCommands generates mail:link/unlink commands for an app.
func mailLinkCommands(appName, desired, actual string) [][]string {
	var cmds [][]string
	if actual != "" && actual != desired {
		cmds = append(cmds, []string{"mail:unlink", actual, appName})
	}
	if desired != "" && desired != actual {
		cmds = append(cmds, []string{"mail:link", desired, appName})
	}
	return cmds
}

// authLinkCommands generates auth:link/unlink and auth:frontend:protect/unprotect commands for an app.
func authLinkCommands(appName string, desired, actual *schema.AuthConfig) [][]string {
	var cmds [][]string
	oldDir := ""
	newDir := ""
	oldProtected := ""
	newProtected := ""
	if actual != nil {
		oldDir = actual.Directory
		oldProtected = actual.Protected
	}
	if desired != nil {
		newDir = desired.Directory
		newProtected = desired.Protected
	}
	if oldDir != "" && oldDir != newDir {
		cmds = append(cmds, []string{"auth:unlink", oldDir, appName})
	}
	if newDir != "" && newDir != oldDir {
		cmds = append(cmds, []string{"auth:link", newDir, appName})
	}
	if oldProtected != "" && oldProtected != newProtected {
		cmds = append(cmds, []string{"auth:frontend:unprotect", oldProtected, appName})
	}
	if newProtected != "" && newProtected != oldProtected {
		cmds = append(cmds, []string{"auth:frontend:protect", newProtected, appName})
	}
	return cmds
}

// configSetArgs builds: config:set --no-restart <app> KEY1=VALUE1 KEY2=VALUE2 ...
func configSetArgs(appName string, env map[string]string) []string {
	args := []string{"config:set", "--no-restart", appName}
	for _, k := range sortedKeys(env) {
		args = append(args, fmt.Sprintf("%s=%s", k, env[k]))
	}
	return args
}

// envUpdateCommands computes config:set and config:unset commands for env drift.
func envUpdateCommands(appName string, desired, actual map[string]string) [][]string {
	var cmds [][]string

	// Find vars to set (new or changed)
	toSetMap := map[string]string{}
	for k, v := range desired {
		if actual[k] != v {
			toSetMap[k] = v
		}
	}
	if len(toSetMap) > 0 {
		cmds = append(cmds, configSetArgs(appName, toSetMap))
	}

	// Find vars to unset (removed)
	var toUnset []string
	for k := range actual {
		if _, ok := desired[k]; !ok {
			toUnset = append(toUnset, k)
		}
	}
	if len(toUnset) > 0 {
		sort.Strings(toUnset)
		cmds = append(cmds, append([]string{"config:unset", "--no-restart", appName}, toUnset...))
	}

	return cmds
}

// linkUpdateCommands computes link/unlink commands for service link drift.
func linkUpdateCommands(appName string, desired, actual map[string]string) [][]string {
	var cmds [][]string

	// Unlink removed services
	for svcType, svcName := range actual {
		if desired[svcType] != svcName {
			cmds = append(cmds, []string{svcType + ":unlink", svcName, appName})
		}
	}

	// Link new services
	for svcType, svcName := range desired {
		if actual[svcType] != svcName {
			cmds = append(cmds, []string{svcType + ":link", svcName, appName})
		}
	}

	return cmds
}

// storageUpdateCommands computes mount/unmount commands for storage drift.
func storageUpdateCommands(appName string, desired, actual []string) [][]string {
	var cmds [][]string

	desiredSet := toSet(desired)
	actualSet := toSet(actual)

	// Unmount removed
	for _, m := range actual {
		if !desiredSet[m] {
			cmds = append(cmds, []string{"storage:unmount", appName, m})
		}
	}

	// Mount new
	for _, m := range desired {
		if !actualSet[m] {
			cmds = append(cmds, []string{"storage:mount", appName, m})
		}
	}

	return cmds
}

// dockerOptionsUpdateCommands clears old options and sets new ones per phase.
func dockerOptionsUpdateCommands(appName string, desired, actual schema.DockerOptions) [][]string {
	var cmds [][]string

	phases := []struct {
		name        string
		desiredOpts []string
		actualOpts  []string
	}{
		{"build", desired.Build, actual.Build},
		{"deploy", desired.Deploy, actual.Deploy},
		{"run", desired.Run, actual.Run},
	}

	for _, phase := range phases {
		desiredSet := toSet(phase.desiredOpts)
		actualSet := toSet(phase.actualOpts)

		// Remove old options
		for _, opt := range phase.actualOpts {
			if !desiredSet[opt] {
				cmds = append(cmds, []string{"docker-options:remove", appName, phase.name, opt})
			}
		}

		// Add new options
		for _, opt := range phase.desiredOpts {
			if !actualSet[opt] {
				cmds = append(cmds, []string{"docker-options:add", appName, phase.name, opt})
			}
		}
	}

	return cmds
}

// portsSetArgs builds: ports:set <app> scheme:host:container ...
func portsSetArgs(appName string, ports map[string]string) []string {
	args := []string{"ports:set", appName}
	for _, k := range sortedKeys(ports) {
		args = append(args, fmt.Sprintf("%s:%s", k, ports[k]))
	}
	return args
}

// scaleSetArgs builds: ps:scale <app> proc=count ...
func scaleSetArgs(appName string, scale map[string]int) []string {
	args := []string{"ps:scale", appName}
	keys := make([]string, 0, len(scale))
	for k := range scale {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, fmt.Sprintf("%s=%d", k, scale[k]))
	}
	return args
}

func dockerOptionsAddCmds(appName, phase string, opts []string) [][]string {
	var cmds [][]string
	for _, opt := range opts {
		cmds = append(cmds, []string{"docker-options:add", appName, phase, opt})
	}
	return cmds
}

func describeStep(s plan.Step) string {
	switch s.Action {
	case plan.CreateApp:
		return fmt.Sprintf("create app %q (image: %s)", s.App, s.NewValue)
	case plan.DestroyApp:
		return fmt.Sprintf("destroy app %q", s.App)
	case plan.UpdateApp:
		return fmt.Sprintf("update app %q field %s", s.App, s.Field)
	case plan.CreateService:
		return fmt.Sprintf("create %s service %q", s.ServiceType, s.Service)
	case plan.DestroyService:
		return fmt.Sprintf("destroy %s service %q", s.ServiceType, s.Service)
	case plan.UpdateService:
		return fmt.Sprintf("update %s service %q", s.ServiceType, s.Service)
	case plan.CreateMailService:
		return fmt.Sprintf("create mail service %q", s.Service)
	case plan.DestroyMailService:
		return fmt.Sprintf("destroy mail service %q", s.Service)
	case plan.UpdateMailService:
		return fmt.Sprintf("update mail service %q", s.Service)
	case plan.CreateAuthDirectory:
		return fmt.Sprintf("create auth directory %q", s.Service)
	case plan.DestroyAuthDirectory:
		return fmt.Sprintf("destroy auth directory %q", s.Service)
	case plan.UpdateAuthDirectory:
		return fmt.Sprintf("update auth directory %q", s.Service)
	case plan.CreateAuthFrontend:
		return fmt.Sprintf("create auth frontend %q", s.Service)
	case plan.DestroyAuthFrontend:
		return fmt.Sprintf("destroy auth frontend %q", s.Service)
	case plan.UpdateAuthFrontend:
		return fmt.Sprintf("update auth frontend %q", s.Service)
	case plan.InstallPlugin:
		return fmt.Sprintf("install plugin %q", s.Service)
	case plan.UninstallPlugin:
		return fmt.Sprintf("uninstall plugin %q", s.Service)
	case plan.UpdatePlugin:
		return fmt.Sprintf("update plugin %q", s.Service)
	case plan.UpdateGlobal:
		return fmt.Sprintf("update global %s", s.Field)
	default:
		return fmt.Sprintf("unknown action: %s", s.Action)
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// resourceCommands generates resource:limit and resource:reserve commands.
func resourceCommands(appName string, resources map[string]schema.ResourceConfig) [][]string {
	var cmds [][]string
	procs := make([]string, 0, len(resources))
	for proc := range resources {
		procs = append(procs, proc)
	}
	sort.Strings(procs)

	for _, proc := range procs {
		rc := resources[proc]
		// Limits
		for _, rv := range []struct {
			flag string
			val  string
		}{
			{"--cpu", rc.Limits.CPU},
			{"--memory", rc.Limits.Memory},
			{"--memory-swap", rc.Limits.MemorySwap},
			{"--network", rc.Limits.Network},
			{"--network-ingress", rc.Limits.NetworkIngress},
			{"--network-egress", rc.Limits.NetworkEgress},
			{"--nvidia-gpu", rc.Limits.NvidiaGPU},
		} {
			if rv.val != "" {
				cmds = append(cmds, []string{"resource:limit", appName, "--process-type", proc, rv.flag, rv.val})
			}
		}
		// Reservations
		for _, rv := range []struct {
			flag string
			val  string
		}{
			{"--cpu", rc.Reservations.CPU},
			{"--memory", rc.Reservations.Memory},
			{"--memory-swap", rc.Reservations.MemorySwap},
			{"--network", rc.Reservations.Network},
			{"--network-ingress", rc.Reservations.NetworkIngress},
			{"--network-egress", rc.Reservations.NetworkEgress},
			{"--nvidia-gpu", rc.Reservations.NvidiaGPU},
		} {
			if rv.val != "" {
				cmds = append(cmds, []string{"resource:reserve", appName, "--process-type", proc, rv.flag, rv.val})
			}
		}
	}
	return cmds
}

// checksCommands generates checks:disable, checks:skip, and checks:set commands.
func checksCommands(appName string, checks *schema.ChecksConfig) [][]string {
	if checks == nil {
		return nil
	}
	var cmds [][]string
	if len(checks.Disabled) > 0 {
		cmds = append(cmds, append([]string{"checks:disable", appName}, checks.Disabled...))
	}
	if len(checks.Skipped) > 0 {
		cmds = append(cmds, append([]string{"checks:skip", appName}, checks.Skipped...))
	}
	if checks.WaitToRetire > 0 {
		cmds = append(cmds, []string{"checks:set", appName, "wait-to-retire", strconv.Itoa(checks.WaitToRetire)})
	}
	return cmds
}

// builderCommands generates builder:set and builder sub-plugin commands.
func builderCommands(appName string, builder *schema.BuilderConfig) [][]string {
	if builder == nil {
		return nil
	}
	var cmds [][]string
	if builder.Selected != "" {
		cmds = append(cmds, []string{"builder:set", appName, "selected", builder.Selected})
	}
	if builder.BuildDir != "" {
		cmds = append(cmds, []string{"builder:set", appName, "build-dir", builder.BuildDir})
	}
	if builder.DockerfilePath != "" {
		cmds = append(cmds, []string{"builder-dockerfile:set", appName, "dockerfile-path", builder.DockerfilePath})
	}
	if builder.PackProjecttomlPath != "" {
		cmds = append(cmds, []string{"builder-pack:set", appName, "projecttoml-path", builder.PackProjecttomlPath})
	}
	if builder.NixpacksTomlPath != "" {
		cmds = append(cmds, []string{"builder-nixpacks:set", appName, "nixpackstoml-path", builder.NixpacksTomlPath})
	}
	if builder.HerokuishAllowed != "" {
		cmds = append(cmds, []string{"builder-herokuish:set", appName, "allowed", builder.HerokuishAllowed})
	}
	return cmds
}

// registryCommands generates registry:set commands.
func registryCommands(appName string, registry *schema.RegistryConfig) [][]string {
	if registry == nil {
		return nil
	}
	var cmds [][]string
	if registry.Server != "" {
		cmds = append(cmds, []string{"registry:set", appName, "server", registry.Server})
	}
	if registry.ImageRepo != "" {
		cmds = append(cmds, []string{"registry:set", appName, "image-repo", registry.ImageRepo})
	}
	if registry.PushOnRelease {
		cmds = append(cmds, []string{"registry:set", appName, "push-on-release", "true"})
	}
	if registry.PushExtraTags != "" {
		cmds = append(cmds, []string{"registry:set", appName, "push-extra-tags", registry.PushExtraTags})
	}
	return cmds
}

// processCommands generates ps:set commands.
func processCommands(appName string, proc *schema.ProcessConfig) [][]string {
	if proc == nil {
		return nil
	}
	var cmds [][]string
	if proc.RestartPolicy != "" {
		cmds = append(cmds, []string{"ps:set", appName, "restart-policy", proc.RestartPolicy})
	}
	if proc.ProcfilePath != "" {
		cmds = append(cmds, []string{"ps:set", appName, "procfile-path", proc.ProcfilePath})
	}
	return cmds
}

// logCommands generates logs:set commands.
func logCommands(appName string, logs *schema.LogConfig) [][]string {
	if logs == nil {
		return nil
	}
	var cmds [][]string
	if logs.MaxSize != "" {
		cmds = append(cmds, []string{"logs:set", appName, "max-size", logs.MaxSize})
	}
	if logs.VectorImage != "" {
		cmds = append(cmds, []string{"logs:set", appName, "vector-image", logs.VectorImage})
	}
	if logs.VectorSink != "" {
		cmds = append(cmds, []string{"logs:set", appName, "vector-sink", logs.VectorSink})
	}
	if logs.AppLabelAlias != "" {
		cmds = append(cmds, []string{"logs:set", appName, "app-label-alias", logs.AppLabelAlias})
	}
	return cmds
}

// schedulerCommands generates scheduler:set and scheduler-docker-local:set commands.
func schedulerCommands(appName string, sched *schema.SchedulerConfig) [][]string {
	if sched == nil {
		return nil
	}
	var cmds [][]string
	if sched.Selected != "" {
		cmds = append(cmds, []string{"scheduler:set", appName, "selected", sched.Selected})
	}
	if sched.DockerLocalInitProcess != "" {
		cmds = append(cmds, []string{"scheduler-docker-local:set", appName, "init-process", sched.DockerLocalInitProcess})
	}
	if sched.DockerLocalParallelScheduleCount != "" {
		cmds = append(cmds, []string{"scheduler-docker-local:set", appName, "parallel-schedule-count", sched.DockerLocalParallelScheduleCount})
	}
	return cmds
}

// buildpacksCommands generates buildpacks:clear + buildpacks:add commands.
func buildpacksCommands(appName string, buildpacks []string) [][]string {
	var cmds [][]string
	cmds = append(cmds, []string{"buildpacks:clear", appName})
	for _, bp := range buildpacks {
		cmds = append(cmds, []string{"buildpacks:add", appName, bp})
	}
	return cmds
}

// createServiceCommands generates commands to create a backing service, with optional image version.
func (e *Executor) createServiceCommands(s plan.Step, desired *schema.Dokkufile) ([][]string, error) {
	cmd := []string{s.ServiceType + ":create", s.Service}
	if svc, ok := desired.Services[s.Service]; ok && svc.ImageVersion != "" {
		cmd = append(cmd, "--image-version", svc.ImageVersion)
	}
	return [][]string{cmd}, nil
}

// updateServiceCommands generates commands to upgrade a backing service's image version.
func (e *Executor) updateServiceCommands(s plan.Step, desired *schema.Dokkufile) ([][]string, error) {
	svc, ok := desired.Services[s.Service]
	if !ok {
		return nil, fmt.Errorf("service %q not found in desired state", s.Service)
	}
	var cmds [][]string
	if svc.ImageVersion != "" {
		cmds = append(cmds, []string{s.ServiceType + ":upgrade", s.Service, "--image-version", svc.ImageVersion})
	}
	return cmds, nil
}

// installPluginCommands generates plugin:install commands.
func (e *Executor) installPluginCommands(name string, desired *schema.Dokkufile) ([][]string, error) {
	plugin, ok := desired.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found in desired state", name)
	}
	cmd := []string{"plugin:install", plugin.URL, "--name", name}
	if plugin.Committish != "" {
		cmd = append(cmd, "--committish", plugin.Committish)
	}
	return [][]string{cmd}, nil
}
