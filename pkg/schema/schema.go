package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Dokkufile is the top-level configuration structure.
type Dokkufile struct {
	Version  string             `yaml:"version" json:"version"`
	Services map[string]Service `yaml:"services,omitempty" json:"services,omitempty"`
	Apps     map[string]App     `yaml:"apps,omitempty" json:"apps,omitempty"`
}

// Service represents a backing service (database, cache, etc).
type Service struct {
	Type string `yaml:"type" json:"type"`
}

// App represents an application deployment.
type App struct {
	Image         string            `yaml:"image" json:"image"`
	Library       string            `yaml:"library,omitempty" json:"library,omitempty"`
	Domains       []string          `yaml:"domains,omitempty" json:"domains,omitempty"`
	Ports         map[string]string `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env           map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Secrets       []string          `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Links         map[string]string `yaml:"links,omitempty" json:"links,omitempty"`
	Storage       []string          `yaml:"storage,omitempty" json:"storage,omitempty"`
	DockerOptions DockerOptions     `yaml:"docker_options,omitempty" json:"docker_options,omitempty"`
	LetsEncrypt   bool              `yaml:"letsencrypt,omitempty" json:"letsencrypt,omitempty"`
	Auth          *AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	DNS           *DNSConfig        `yaml:"dns,omitempty" json:"dns,omitempty"`
	Mail          string            `yaml:"mail,omitempty" json:"mail,omitempty"`
	Healthcheck   *Healthcheck      `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Scale         map[string]int    `yaml:"scale,omitempty" json:"scale,omitempty"`
}

// DockerOptions holds docker options grouped by phase.
type DockerOptions struct {
	Deploy []string `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Run    []string `yaml:"run,omitempty" json:"run,omitempty"`
	Build  []string `yaml:"build,omitempty" json:"build,omitempty"`
}

// AuthConfig holds authentication settings for an app.
type AuthConfig struct {
	Integration string `yaml:"integration" json:"integration"`
	Group       string `yaml:"group,omitempty" json:"group,omitempty"`
}

// DNSConfig holds DNS settings for an app.
type DNSConfig struct {
	Zone string `yaml:"zone" json:"zone"`
}

// Healthcheck holds health check configuration.
type Healthcheck struct {
	Path    string `yaml:"path" json:"path"`
	Timeout int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// Load reads a Dokkufile from disk, auto-detecting YAML or JSON by extension.
func Load(path string) (*Dokkufile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading dokkufile: %w", err)
	}
	return Parse(data, path)
}

// Parse parses a Dokkufile from bytes, using the filename to detect format.
func Parse(data []byte, filename string) (*Dokkufile, error) {
	var df Dokkufile

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &df); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
	case ".yml", ".yaml", "":
		if err := yaml.Unmarshal(data, &df); err != nil {
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	return &df, nil
}
