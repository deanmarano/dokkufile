package compose

import (
	"fmt"
	"sort"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
	"gopkg.in/yaml.v3"
)

// ImportResult holds the converted Dokkufile and any warnings from the import.
type ImportResult struct {
	Dokkufile *schema.Dokkufile
	Warnings  []string
}

// ComposeFile represents the subset of docker-compose.yml we parse.
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a single service in a docker-compose file.
type ComposeService struct {
	Image           string               `yaml:"image"`
	Ports           []string             `yaml:"ports"`
	Environment     map[string]string    `yaml:"environment"`
	Volumes         []string             `yaml:"volumes"`
	DependsOn       ComposeDependsOn     `yaml:"depends_on"`
	Healthcheck     *ComposeHealthcheck  `yaml:"healthcheck"`
	Deploy          *ComposeDeploy       `yaml:"deploy"`
	Restart         string               `yaml:"restart"`
	Build           *ComposeBuild        `yaml:"build"`
	CapAdd          []string             `yaml:"cap_add"`
	CapDrop         []string             `yaml:"cap_drop"`
	Networks        ComposeNetworks      `yaml:"networks"`
	Logging         *ComposeLogging      `yaml:"logging"`
	Entrypoint      interface{}          `yaml:"entrypoint"`
	ExtraHosts      []string             `yaml:"extra_hosts"`
	Tmpfs           ComposeTmpfs         `yaml:"tmpfs"`
	Sysctls         ComposeSysctls       `yaml:"sysctls"`
	ShmSize         string               `yaml:"shm_size"`
	User            string               `yaml:"user"`
	StopGracePeriod string               `yaml:"stop_grace_period"`
	Labels          ComposeLabels        `yaml:"labels"`
	Privileged      bool                 `yaml:"privileged"`
	Dns             []string             `yaml:"dns"`
	DnsSearch       []string             `yaml:"dns_search"`
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

// ComposeTmpfs is a custom type that handles both string and list forms of tmpfs.
type ComposeTmpfs []string

// UnmarshalYAML handles both string form (tmpfs: /run) and list form (tmpfs: [/run, /tmp]).
func (t *ComposeTmpfs) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*t = []string{s}
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*t = list
	}
	return nil
}

// ComposeSysctls is a custom type that handles both map and list forms of sysctls.
type ComposeSysctls map[string]string

// UnmarshalYAML handles both map form (sysctls: {k: v}) and list form (sysctls: ["k=v"]).
func (s *ComposeSysctls) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var m map[string]string
		if err := value.Decode(&m); err != nil {
			return err
		}
		*s = m
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		result := make(map[string]string)
		for _, item := range list {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
		*s = result
	}
	return nil
}

// ComposeLabels is a custom type that handles both map and list forms of labels.
type ComposeLabels map[string]string

// UnmarshalYAML handles both map form (labels: {k: v}) and list form (labels: ["k=v"]).
func (l *ComposeLabels) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.MappingNode:
		var m map[string]string
		if err := value.Decode(&m); err != nil {
			return err
		}
		*l = m
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		result := make(map[string]string)
		for _, item := range list {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
		*l = result
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

// handledFields are compose service fields we parse and convert.
var handledFields = map[string]bool{
	"image":             true,
	"ports":             true,
	"environment":       true,
	"volumes":           true,
	"depends_on":        true,
	"healthcheck":       true,
	"deploy":            true,
	"restart":           true,
	"build":             true,
	"cap_add":           true,
	"cap_drop":          true,
	"networks":          true,
	"logging":           true,
	"entrypoint":        true,
	"extra_hosts":       true,
	"tmpfs":             true,
	"sysctls":           true,
	"shm_size":          true,
	"user":              true,
	"stop_grace_period": true,
	"labels":            true,
	"privileged":        true,
	"dns":               true,
	"dns_search":        true,
}

// skippedFields are compose service fields we recognize but skip, with reasons.
var skippedFields = map[string]string{
	"command":        "use a Procfile instead",
	"stdin_open":     "not applicable to production deployments",
	"tty":            "not applicable to production deployments",
	"env_file":       "cannot resolve file paths during import; add env vars manually",
	"container_name": "managed by dokku",
	"hostname":       "managed by dokku",
	"working_dir":    "set in Dockerfile instead",
	"expose":         "use ports mapping instead",
	"pid":            "namespace mode not supported",
	"ipc":            "namespace mode not supported",
	"profiles":       "compose-specific, not applicable to dokku",
	"extends":        "compose-specific, not applicable to dokku",
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
	for _, suffix := range []string{"-alpine", "-slim", "-bullseye", "-bookworm", "-buster"} {
		tag = strings.TrimSuffix(tag, suffix)
	}
	return tag
}

// ImportCompose parses a docker-compose YAML and returns an ImportResult.
func ImportCompose(data []byte) (*ImportResult, error) {
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

		// Convert extra_hosts
		for _, host := range svc.ExtraHosts {
			dockerOpts = append(dockerOpts, "--add-host="+host)
		}

		// Convert tmpfs
		for _, path := range svc.Tmpfs {
			dockerOpts = append(dockerOpts, "--tmpfs="+path)
		}

		// Convert sysctls
		if len(svc.Sysctls) > 0 {
			keys := make([]string, 0, len(svc.Sysctls))
			for k := range svc.Sysctls {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				dockerOpts = append(dockerOpts, "--sysctl="+k+"="+svc.Sysctls[k])
			}
		}

		// Convert shm_size
		if svc.ShmSize != "" {
			dockerOpts = append(dockerOpts, "--shm-size="+svc.ShmSize)
		}

		// Convert user
		if svc.User != "" {
			dockerOpts = append(dockerOpts, "--user="+svc.User)
		}

		// Convert stop_grace_period
		if svc.StopGracePeriod != "" {
			seconds := parseDurationSeconds(svc.StopGracePeriod)
			if seconds > 0 {
				dockerOpts = append(dockerOpts, fmt.Sprintf("--stop-timeout=%d", seconds))
			}
		}

		// Convert labels
		if len(svc.Labels) > 0 {
			keys := make([]string, 0, len(svc.Labels))
			for k := range svc.Labels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				dockerOpts = append(dockerOpts, "--label="+k+"="+svc.Labels[k])
			}
		}

		// Convert privileged
		if svc.Privileged {
			dockerOpts = append(dockerOpts, "--privileged")
		}

		// Convert dns
		for _, server := range svc.Dns {
			dockerOpts = append(dockerOpts, "--dns="+server)
		}

		// Convert dns_search
		for _, domain := range svc.DnsSearch {
			dockerOpts = append(dockerOpts, "--dns-search="+domain)
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

	warnings := collectWarnings(data)

	return &ImportResult{
		Dokkufile: df,
		Warnings:  warnings,
	}, nil
}

// collectWarnings walks the raw YAML to detect skipped and unrecognized service fields.
func collectWarnings(data []byte) []string {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}

	// root.Content[0] is the document mapping
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil
	}

	// Find the "services" key
	var servicesNode *yaml.Node
	for i := 0; i < len(doc.Content)-1; i += 2 {
		if doc.Content[i].Value == "services" {
			servicesNode = doc.Content[i+1]
			break
		}
	}
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return nil
	}

	var warnings []string

	// Iterate over each service
	for i := 0; i < len(servicesNode.Content)-1; i += 2 {
		svcName := servicesNode.Content[i].Value
		svcNode := servicesNode.Content[i+1]
		if svcNode.Kind != yaml.MappingNode {
			continue
		}

		// Iterate over each field in the service
		for j := 0; j < len(svcNode.Content)-1; j += 2 {
			fieldName := svcNode.Content[j].Value

			if handledFields[fieldName] {
				continue
			}

			if reason, ok := skippedFields[fieldName]; ok {
				warnings = append(warnings, fmt.Sprintf("service %q: field %q skipped (%s)", svcName, fieldName, reason))
			} else {
				warnings = append(warnings, fmt.Sprintf("service %q: unrecognized field %q ignored", svcName, fieldName))
			}
		}
	}

	return warnings
}

// parsePortMapping extracts the scheme and host port from a compose port string.
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
