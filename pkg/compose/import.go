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
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports"`
	Environment map[string]string `yaml:"environment"`
	Volumes     []string          `yaml:"volumes"`
	DependsOn   []string          `yaml:"depends_on"`
}

// knownServices maps image prefixes to dokku service types.
var knownServices = map[string]string{
	"postgres":  "postgres",
	"redis":     "redis",
	"mysql":     "mysql",
	"mariadb":   "mariadb",
	"mongo":     "mongo",
	"memcached": "memcached",
	"rabbitmq":  "rabbitmq",
	"elasticsearch": "elasticsearch",
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

		if len(svc.Ports) > 0 {
			app.Ports = map[string]string{
				"http": svc.Ports[0],
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

		df.Apps[name] = app
	}

	return df, nil
}
