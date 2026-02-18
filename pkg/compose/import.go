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
	Image       string                `yaml:"image"`
	Ports       []string              `yaml:"ports"`
	Environment map[string]string     `yaml:"environment"`
	Volumes     []string              `yaml:"volumes"`
	DependsOn   ComposeDependsOn      `yaml:"depends_on"`
	Healthcheck *ComposeHealthcheck   `yaml:"healthcheck"`
	Deploy      *ComposeDeploy        `yaml:"deploy"`
	Restart     string                `yaml:"restart"`
	Build       *ComposeBuild         `yaml:"build"`
	CapAdd      []string              `yaml:"cap_add"`
	CapDrop     []string              `yaml:"cap_drop"`
	Networks    ComposeNetworks       `yaml:"networks"`
	Logging     *ComposeLogging       `yaml:"logging"`
	Entrypoint  interface{}           `yaml:"entrypoint"`
}

// ComposeBuild represents docker-compose build configuration.
type ComposeBuild struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// ComposeLogging represents docker-compose logging configuration.
type ComposeLogging struct {
	Options map[string]string `yaml:"options"`
}

// ComposeNetworks is a custom type that handles both list and map forms of networks.
type ComposeNetworks []string

// UnmarshalYAML handles both list form (networks: [net1]) and map form (networks: {net1: {}}).
func (n *ComposeNetworks) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*n = list
	case yaml.MappingNode:
		var m map[string]interface{}
		if err := value.Decode(&m); err != nil {
			return err
		}
		for k := range m {
			*n = append(*n, k)
		}
	}
	return nil
}

// ComposeDependsOn is a custom type that handles both list and map forms of depends_on.
type ComposeDependsOn []string

// UnmarshalYAML handles both list form (depends_on: [db]) and map form (depends_on: {db: {condition: ...}}).
func (d *ComposeDependsOn) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*d = list
	case yaml.MappingNode:
		var m map[string]interface{}
		if err := value.Decode(&m); err != nil {
			return err
		}
		for k := range m {
			*d = append(*d, k)
		}
	}
	return nil
}

// ComposeHealthcheck represents a docker-compose healthcheck configuration.
type ComposeHealthcheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period"`
}

// ComposeDeploy represents docker-compose deploy configuration.
type ComposeDeploy struct {
	Replicas  int               `yaml:"replicas"`
	Resources *ComposeResources `yaml:"resources"`
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

// servicePluginURLs maps dokku service types to their plugin install URLs.
var servicePluginURLs = map[string]string{
	"postgres":      "https://github.com/dokku/dokku-postgres.git",
	"redis":         "https://github.com/dokku/dokku-redis.git",
	"mysql":         "https://github.com/dokku/dokku-mysql.git",
	"mariadb":       "https://github.com/dokku/dokku-mariadb.git",
	"mongo":         "https://github.com/dokku/dokku-mongo.git",
	"memcached":     "https://github.com/dokku/dokku-memcached.git",
	"rabbitmq":      "https://github.com/dokku/dokku-rabbitmq.git",
	"elasticsearch": "https://github.com/dokku/dokku-elasticsearch.git",
	"clickhouse":    "https://github.com/dokku/dokku-clickhouse.git",
	"couchdb":       "https://github.com/dokku/dokku-couchdb.git",
	"meilisearch":   "https://github.com/dokku/dokku-meilisearch.git",
	"nats":          "https://github.com/dokku/dokku-nats.git",
	"rethinkdb":     "https://github.com/dokku/dokku-rethinkdb.git",
	"solr":          "https://github.com/dokku/dokku-solr.git",
	"typesense":     "https://github.com/dokku/dokku-typesense.git",
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

// extractImageVersion extracts the version tag from a Docker image string.
// Returns empty string if no tag or if tag is "latest".
func extractImageVersion(image string) string {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	tag := parts[1]
	if tag == "latest" {
		return ""
	}
	// Strip common suffixes like "-alpine", "-slim", etc. to get the version core
	// e.g. "15-alpine" -> "15", "7-alpine" -> "7"
	for _, suffix := range []string{"-alpine", "-slim", "-bullseye", "-bookworm", "-buster"} {
		tag = strings.TrimSuffix(tag, suffix)
	}
	return tag
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
			service := schema.Service{Type: svcType}
			if ver := extractImageVersion(svc.Image); ver != "" {
				service.ImageVersion = ver
			}
			df.Services[name] = service
			serviceTypes[name] = svcType

			// Auto-add the corresponding dokku plugin
			if pluginURL, hasPlugin := servicePluginURLs[svcType]; hasPlugin {
				if df.Plugins == nil {
					df.Plugins = map[string]schema.Plugin{}
				}
				if _, exists := df.Plugins[svcType]; !exists {
					df.Plugins[svcType] = schema.Plugin{URL: pluginURL}
				}
			}
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

		// Convert restart policy
		if svc.Restart != "" && svc.Restart != "no" {
			policy := svc.Restart
			if policy == "on-failure" {
				policy = "on-failure:10"
			}
			app.Process = &schema.ProcessConfig{
				RestartPolicy: policy,
			}
		}

		// Convert build dockerfile (only if no image is set)
		if svc.Build != nil && svc.Build.Dockerfile != "" && svc.Image == "" {
			app.Builder = &schema.BuilderConfig{
				Selected:       "dockerfile",
				DockerfilePath: svc.Build.Dockerfile,
			}
		}

		// Convert cap_add and cap_drop to docker options
		var dockerOpts []string
		for _, cap := range svc.CapAdd {
			dockerOpts = append(dockerOpts, "--cap-add="+cap)
		}
		for _, cap := range svc.CapDrop {
			dockerOpts = append(dockerOpts, "--cap-drop="+cap)
		}

		// Convert entrypoint to docker option
		if svc.Entrypoint != nil {
			switch v := svc.Entrypoint.(type) {
			case string:
				if v != "" {
					dockerOpts = append(dockerOpts, "--entrypoint="+v)
				}
			case []interface{}:
				if len(v) > 0 {
					parts := make([]string, len(v))
					for i, p := range v {
						parts[i] = fmt.Sprintf("%v", p)
					}
					dockerOpts = append(dockerOpts, "--entrypoint="+strings.Join(parts, " "))
				}
			}
		}

		if len(dockerOpts) > 0 {
			app.DockerOptions.Deploy = append(app.DockerOptions.Deploy, dockerOpts...)
		}

		// Convert first network to initial_network
		if len(svc.Networks) > 0 {
			app.Network = &schema.NetworkConfig{
				InitialNetwork: svc.Networks[0],
			}
		}

		// Convert logging max-size
		if svc.Logging != nil && svc.Logging.Options != nil {
			if maxSize, ok := svc.Logging.Options["max-size"]; ok {
				app.Logs = &schema.LogConfig{
					MaxSize: maxSize,
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

	if hc.StartPeriod != "" {
		cfg.InitialDelay = parseDurationSeconds(hc.StartPeriod)
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
