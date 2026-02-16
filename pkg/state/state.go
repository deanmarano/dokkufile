package state

import (
	"os/exec"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

// CommandRunner abstracts command execution so tests can use fakes.
type CommandRunner interface {
	Run(args ...string) (string, error)
}

// ExecRunner shells out to the real dokku binary.
type ExecRunner struct{}

func (r *ExecRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("dokku", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Reader reads live state from a dokku server.
type Reader interface {
	Read() (*schema.Dokkufile, error)
}

// DokkuReader reads state by shelling out to dokku commands.
type DokkuReader struct {
	Runner CommandRunner
}

// serviceTypes lists the backing service plugins to scan.
var serviceTypes = []string{"postgres", "redis", "mysql", "mariadb", "mongo"}

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
