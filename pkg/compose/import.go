package compose

import (
	"fmt"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"gopkg.in/yaml.v3"
)

// ComposeFile represents the subset of docker-compose.yml we parse.
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a single service in a docker-compose file.
type ComposeService struct {
	Image       string                 `yaml:"image"`
	Ports       []string               `yaml:"ports"`
	Environment map[string]string      `yaml:"environment"`
	Volumes     []string               `yaml:"volumes"`
	DependsOn   []string               `yaml:"depends_on"`
	Healthcheck *ComposeHealthcheck    `yaml:"healthcheck"`
	Deploy      *ComposeDeploy         `yaml:"deploy"`
}

// ComposeHealthcheck represents a docker-compose healthcheck configuration.
type ComposeHealthcheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval"`
	Timeout  string   `yaml:"timeout"`
	Retries  int      `yaml:"retries"`
}

// ComposeDeploy represents docker-compose deploy configuration.
type ComposeDeploy struct {
	Replicas int                    `yaml:"replicas"`
	Resources *ComposeResources     `yaml:"resources"`
}

// ComposeResources represents docker-compose resource constraints.
type ComposeResources struct {
	Limits       *ComposeResourceValues `yaml:"limits"`
	Reservations *ComposeResourceValues `yaml:"reservations"`
}

// ComposeResourceValues holds CPU/memory constraints.
type ComposeResourceValues struct {
	CPUs   string `yaml:"cpus"`
	Memory string `yaml:"memory"`
}

// knownServices maps image prefixes to dokku service types.
var knownServices = map[string]string{
	"postgres":      "postgres",
	"redis":         "redis",
	"mysql":         "mysql",
	"mariadb":       "mariadb",
	"mongo":         "mongo",
	"memcached":     "memcached",
	"rabbitmq":      "rabbitmq",
	"elasticsearch": "elasticsearch",
	"clickhouse":    "clickhouse",
	"couchdb":       "couchdb",
	"meilisearch":   "meilisearch",
	"nats":          "nats",
	"rethinkdb":     "rethinkdb",
	"solr":          "solr",
	"typesense":     "typesense",
}

// isKnownService checks if an image name matches a known backing service.
func isKnownService(image string) (string, bool) {
	// Extract image name without tag
	name := strings.Split(image, ":")[0]
	// Remove registry prefix if present
	parts := strings.Split(name, "/")
	name = parts[len(parts)-1]

	for prefix, svcType := range knownServices {
		if strings.HasPrefix(name, prefix) {
			return svcType, true
		}
	}
	return "", false
}

// ImportCompose parses a docker-compose YAML and returns a Dokkufile.
func ImportCompose(data []byte) (*schema.Dokkufile, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}

	df := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{},
		Apps:     map[string]schema.App{},
	}

	// First pass: classify services vs apps
	serviceTypes := map[string]string{} // compose service name -> dokku service type
	for name, svc := range compose.Services {
		if svcType, ok := isKnownService(svc.Image); ok {
			df.Services[name] = schema.Service{Type: svcType}
			serviceTypes[name] = svcType
		}
	}

	// Second pass: build apps
	for name, svc := range compose.Services {
		if _, isService := serviceTypes[name]; isService {
			continue
		}

		app := schema.App{
			Image: svc.Image,
		}

		// Parse ports with scheme detection
		if len(svc.Ports) > 0 {
			app.Ports = map[string]string{}
			for _, portMapping := range svc.Ports {
				scheme, key := parsePortMapping(portMapping)
				app.Ports[scheme+":"+key] = parseContainerPort(portMapping)
			}
		}

		if len(svc.Environment) > 0 {
			app.Env = svc.Environment
		}

		if len(svc.Volumes) > 0 {
			app.Storage = svc.Volumes
		}

		// Convert depends_on to links for known services
		if len(svc.DependsOn) > 0 {
			app.Links = map[string]string{}
			for _, dep := range svc.DependsOn {
				if svcType, ok := serviceTypes[dep]; ok {
					app.Links[svcType] = dep
				}
			}
			if len(app.Links) == 0 {
				app.Links = nil
			}
		}

		// Convert healthcheck
		if svc.Healthcheck != nil {
			hc := convertHealthcheck(svc.Healthcheck)
			if hc != nil {
				app.Healthchecks = map[string][]schema.HealthcheckConfig{
					"web": {*hc},
				}
			}
		}

		// Convert deploy replicas to scale
		if svc.Deploy != nil && svc.Deploy.Replicas > 0 {
			app.Scale = map[string]int{
				"web": svc.Deploy.Replicas,
			}
		}

		// Convert deploy resources
		if svc.Deploy != nil && svc.Deploy.Resources != nil {
			rc := schema.ResourceConfig{}
			if svc.Deploy.Resources.Limits != nil {
				rc.Limits = schema.ResourceValues{
					CPU:    svc.Deploy.Resources.Limits.CPUs,
					Memory: svc.Deploy.Resources.Limits.Memory,
				}
			}
			if svc.Deploy.Resources.Reservations != nil {
				rc.Reservations = schema.ResourceValues{
					CPU:    svc.Deploy.Resources.Reservations.CPUs,
					Memory: svc.Deploy.Resources.Reservations.Memory,
				}
			}
			if rc.Limits.CPU != "" || rc.Limits.Memory != "" || rc.Reservations.CPU != "" || rc.Reservations.Memory != "" {
				app.Resources = map[string]schema.ResourceConfig{
					"web": rc,
				}
			}
		}

		df.Apps[name] = app
	}

	return df, nil
}

// parsePortMapping extracts the scheme and host port from a compose port string.
// Format: "host:container" or "host:container/protocol"
// Returns (scheme, hostPort). Scheme is "https" for 443, "http" otherwise.
func parsePortMapping(port string) (string, string) {
	// Strip protocol suffix if present (e.g., "8080:80/tcp")
	port = strings.Split(port, "/")[0]

	parts := strings.Split(port, ":")
	hostPort := parts[0]
	if len(parts) >= 2 {
		hostPort = parts[0]
	}

	scheme := "http"
	if hostPort == "443" {
		scheme = "https"
	}
	return scheme, hostPort
}

// parseContainerPort extracts the container port from a compose port string.
func parseContainerPort(port string) string {
	port = strings.Split(port, "/")[0]
	parts := strings.Split(port, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return parts[0]
}

// convertHealthcheck converts a compose healthcheck to a dokku healthcheck config.
func convertHealthcheck(hc *ComposeHealthcheck) *schema.HealthcheckConfig {
	if hc == nil || len(hc.Test) == 0 {
		return nil
	}

	cfg := &schema.HealthcheckConfig{}

	// Parse the test command
	// docker-compose test can be: ["CMD", "curl", ...] or ["CMD-SHELL", "cmd"]
	if len(hc.Test) >= 2 {
		switch hc.Test[0] {
		case "CMD", "CMD-SHELL":
			cfg.Command = strings.Join(hc.Test[1:], " ")
		default:
			cfg.Command = strings.Join(hc.Test, " ")
		}
	}

	if hc.Retries > 0 {
		cfg.Attempts = hc.Retries
	}

	if hc.Timeout != "" {
		cfg.Timeout = parseDurationSeconds(hc.Timeout)
	}

	if hc.Interval != "" {
		cfg.Wait = parseDurationSeconds(hc.Interval)
	}

	return cfg
}

// parseDurationSeconds parses a docker-compose duration string (e.g., "30s", "1m") to seconds.
func parseDurationSeconds(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Try simple formats: "30s", "5m", "1h"
	if strings.HasSuffix(s, "s") {
		var n int
		fmt.Sscanf(s, "%ds", &n)
		return n
	}
	if strings.HasSuffix(s, "m") {
		var n int
		fmt.Sscanf(s, "%dm", &n)
		return n * 60
	}
	if strings.HasSuffix(s, "h") {
		var n int
		fmt.Sscanf(s, "%dh", &n)
		return n * 3600
	}
	return 0
}
