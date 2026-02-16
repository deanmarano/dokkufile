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
	Version         string                       `yaml:"version" json:"version"`
	Services        map[string]Service           `yaml:"services,omitempty" json:"services,omitempty"`
	Apps            map[string]App               `yaml:"apps,omitempty" json:"apps,omitempty"`
	MailServices    map[string]MailService        `yaml:"mail_services,omitempty" json:"mail_services,omitempty"`
	AuthDirectories map[string]AuthDirectory     `yaml:"auth_directories,omitempty" json:"auth_directories,omitempty"`
	AuthFrontends   map[string]AuthFrontend      `yaml:"auth_frontends,omitempty" json:"auth_frontends,omitempty"`
}

// Service represents a backing service (database, cache, etc).
type Service struct {
	Type string `yaml:"type" json:"type"`
}

// App represents an application deployment.
type App struct {
	Image         string            `yaml:"image,omitempty" json:"image,omitempty"`
	Git           *GitConfig        `yaml:"git,omitempty" json:"git,omitempty"`
	Library       string            `yaml:"library,omitempty" json:"library,omitempty"`
	Domains       []string          `yaml:"domains,omitempty" json:"domains,omitempty"`
	Ports         map[string]string `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env           map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Secrets       []string          `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Links         map[string]string `yaml:"links,omitempty" json:"links,omitempty"`
	Storage       []string          `yaml:"storage,omitempty" json:"storage,omitempty"`
	DockerOptions DockerOptions     `yaml:"docker_options,omitempty" json:"docker_options,omitempty"`
	LetsEncrypt   bool              `yaml:"letsencrypt,omitempty" json:"letsencrypt,omitempty"`
	SSL           *SSLConfig        `yaml:"ssl,omitempty" json:"ssl,omitempty"`
	Network       *NetworkConfig    `yaml:"network,omitempty" json:"network,omitempty"`
	Nginx         *NginxConfig      `yaml:"nginx,omitempty" json:"nginx,omitempty"`
	Proxy         *ProxyConfig      `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Auth          *AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	Mail          string            `yaml:"mail,omitempty" json:"mail,omitempty"`
	Healthchecks  map[string][]HealthcheckConfig `yaml:"healthchecks,omitempty" json:"healthchecks,omitempty"`
	Cron          []CronJob         `yaml:"cron,omitempty" json:"cron,omitempty"`
	NginxTemplate string            `yaml:"nginx_template,omitempty" json:"nginx_template,omitempty"`
	Scale         map[string]int    `yaml:"scale,omitempty" json:"scale,omitempty"`
}

// DockerOptions holds docker options grouped by phase.
type DockerOptions struct {
	Deploy []string `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Run    []string `yaml:"run,omitempty" json:"run,omitempty"`
	Build  []string `yaml:"build,omitempty" json:"build,omitempty"`
}

// GitConfig holds git deployment settings.
type GitConfig struct {
	Repo       string `yaml:"repo,omitempty" json:"repo,omitempty"`
	Branch     string `yaml:"branch,omitempty" json:"branch,omitempty"`
	KeepGitDir bool   `yaml:"keep_git_dir,omitempty" json:"keep_git_dir,omitempty"`
}

// NetworkConfig holds network settings for an app.
type NetworkConfig struct {
	AttachPostCreate  string `yaml:"attach_post_create,omitempty" json:"attach_post_create,omitempty"`
	AttachPostDeploy  string `yaml:"attach_post_deploy,omitempty" json:"attach_post_deploy,omitempty"`
	BindAllInterfaces bool   `yaml:"bind_all_interfaces,omitempty" json:"bind_all_interfaces,omitempty"`
	InitialNetwork    string `yaml:"initial_network,omitempty" json:"initial_network,omitempty"`
	StaticWebListener string `yaml:"static_web_listener,omitempty" json:"static_web_listener,omitempty"`
	TLD               string `yaml:"tld,omitempty" json:"tld,omitempty"`
}

// NginxConfig holds nginx settings for an app.
type NginxConfig struct {
	HSTS                  bool `yaml:"hsts,omitempty" json:"hsts,omitempty"`
	HSTSIncludeSubdomains bool `yaml:"hsts_include_subdomains,omitempty" json:"hsts_include_subdomains,omitempty"`
	HSTSMaxAge            int  `yaml:"hsts_max_age,omitempty" json:"hsts_max_age,omitempty"`
	HSTSPreload           bool `yaml:"hsts_preload,omitempty" json:"hsts_preload,omitempty"`
}

// ProxyConfig holds proxy settings for an app.
type ProxyConfig struct {
	Enabled bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`
}

// SSLConfig holds custom SSL certificate settings for an app.
type SSLConfig struct {
	CertFile string `yaml:"cert_file,omitempty" json:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
}

// AuthConfig holds authentication settings for an app.
type AuthConfig struct {
	Directory string `yaml:"directory,omitempty" json:"directory,omitempty"`
	Protected string `yaml:"protected,omitempty" json:"protected,omitempty"`
}

// HealthcheckConfig holds health check configuration for a process type.
type HealthcheckConfig struct {
	Path         string `yaml:"path,omitempty" json:"path,omitempty"`
	Port         int    `yaml:"port,omitempty" json:"port,omitempty"`
	Attempts     int    `yaml:"attempts,omitempty" json:"attempts,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Wait         int    `yaml:"wait,omitempty" json:"wait,omitempty"`
	InitialDelay int    `yaml:"initial_delay,omitempty" json:"initial_delay,omitempty"`
	Content      string `yaml:"content,omitempty" json:"content,omitempty"`
	Command      string `yaml:"command,omitempty" json:"command,omitempty"`
}

// CronJob holds a cron job definition.
type CronJob struct {
	Command  string `yaml:"command" json:"command"`
	Schedule string `yaml:"schedule" json:"schedule"`
}

// MailService represents a mail service.
type MailService struct {
	Provider string            `yaml:"provider" json:"provider"`
	Config   map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// AuthDirectory represents an auth directory service.
type AuthDirectory struct {
	Provider string            `yaml:"provider" json:"provider"`
	Config   map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// AuthFrontend represents an auth frontend service.
type AuthFrontend struct {
	Provider      string            `yaml:"provider" json:"provider"`
	Directory     string            `yaml:"directory" json:"directory"`
	Config        map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
	ProtectedApps []string          `yaml:"protected_apps,omitempty" json:"protected_apps,omitempty"`
	OIDCEnabled   bool              `yaml:"oidc_enabled,omitempty" json:"oidc_enabled,omitempty"`
	OIDCClients   []OIDCClient      `yaml:"oidc_clients,omitempty" json:"oidc_clients,omitempty"`
}

// OIDCClient represents an OIDC client configuration.
type OIDCClient struct {
	ID          string `yaml:"id" json:"id"`
	Secret      string `yaml:"secret,omitempty" json:"secret,omitempty"`
	RedirectURI string `yaml:"redirect_uri,omitempty" json:"redirect_uri,omitempty"`
}

// Validate checks the Dokkufile for logical errors.
func (df *Dokkufile) Validate() error {
	for name, app := range df.Apps {
		if app.Image != "" && app.Git != nil && app.Git.Repo != "" {
			return fmt.Errorf("app %q: image and git.repo are mutually exclusive", name)
		}
		if app.LetsEncrypt && app.SSL != nil && app.SSL.CertFile != "" {
			return fmt.Errorf("app %q: letsencrypt and ssl are mutually exclusive", name)
		}
	}
	return nil
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

	if err := df.Validate(); err != nil {
		return nil, err
	}
	return &df, nil
}
