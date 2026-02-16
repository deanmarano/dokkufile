package apply

import (
	"fmt"
	"sort"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
	"github.com/deanmarano/dokkufile/pkg/state"
)

// Executor applies a plan by shelling out to dokku commands.
type Executor struct {
	Runner state.CommandRunner
	DryRun bool
}

// Execute runs each step in the plan, using the desired and actual state for context.
func (e *Executor) Execute(p *plan.Plan, desired, actual *schema.Dokkufile) error {
	for _, step := range p.Steps {
		cmds, err := e.commandsForStep(step, desired, actual)
		if err != nil {
			return fmt.Errorf("planning commands for %s: %w", describeStep(step), err)
		}

		for _, cmd := range cmds {
			if e.DryRun {
				fmt.Printf("[dry-run] dokku %s\n", strings.Join(cmd, " "))
				continue
			}
			fmt.Printf("Running: dokku %s\n", strings.Join(cmd, " "))
			out, err := e.Runner.Run(cmd...)
			if err != nil {
				return fmt.Errorf("dokku %s: %s: %w", strings.Join(cmd, " "), out, err)
			}
		}
	}
	return nil
}

// commandsForStep translates a plan step into one or more dokku command arg slices.
func (e *Executor) commandsForStep(s plan.Step, desired, actual *schema.Dokkufile) ([][]string, error) {
	switch s.Action {
	case plan.CreateService:
		return [][]string{{s.ServiceType + ":create", s.Service}}, nil

	case plan.DestroyService:
		return [][]string{{s.ServiceType + ":destroy", s.Service, "--force"}}, nil

	case plan.CreateApp:
		return e.createAppCommands(s.App, desired)

	case plan.DestroyApp:
		return [][]string{{"apps:destroy", s.App, "--force"}}, nil

	case plan.UpdateApp:
		return e.updateAppCommands(s, desired, actual)

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

	// Set image
	if app.Image != "" {
		cmds = append(cmds, []string{"git:from-image", appName, app.Image})
	}

	// Set domains
	if len(app.Domains) > 0 {
		cmds = append(cmds, append([]string{"domains:set", appName}, app.Domains...))
	}

	// Set env
	if len(app.Env) > 0 {
		cmds = append(cmds, configSetArgs(appName, app.Env))
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

	// Link services
	for svcType, svcName := range app.Links {
		cmds = append(cmds, []string{svcType + ":link", svcName, appName})
	}

	// Enable letsencrypt
	if app.LetsEncrypt {
		cmds = append(cmds, []string{"letsencrypt:enable", appName})
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
		// Clear and reset ports
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
		if dApp.LetsEncrypt {
			return [][]string{{"letsencrypt:enable", s.App}}, nil
		}
		return [][]string{{"letsencrypt:disable", s.App}}, nil

	default:
		return nil, fmt.Errorf("unknown field: %s", s.Field)
	}
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
	toSet := map[string]string{}
	for k, v := range desired {
		if actual[k] != v {
			toSet[k] = v
		}
	}
	if len(toSet) > 0 {
		cmds = append(cmds, configSetArgs(appName, toSet))
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
		name          string
		desiredOpts   []string
		actualOpts    []string
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
