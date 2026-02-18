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
	Global          *GlobalConfig                `yaml:"global,omitempty" json:"global,omitempty"`
	Services        map[string]Service           `yaml:"services,omitempty" json:"services,omitempty"`
	Apps            map[string]App               `yaml:"apps,omitempty" json:"apps,omitempty"`
	MailServices    map[string]MailService        `yaml:"mail_services,omitempty" json:"mail_services,omitempty"`
	AuthDirectories map[string]AuthDirectory     `yaml:"auth_directories,omitempty" json:"auth_directories,omitempty"`
	AuthFrontends   map[string]AuthFrontend      `yaml:"auth_frontends,omitempty" json:"auth_frontends,omitempty"`
	Plugins         map[string]Plugin            `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}

// GlobalConfig holds server-wide default settings.
// These are applied via --global flag and serve as defaults for all apps.
type GlobalConfig struct {
	Domains   []string         `yaml:"domains,omitempty" json:"domains,omitempty"`
	Nginx     *NginxConfig     `yaml:"nginx,omitempty" json:"nginx,omitempty"`
	Proxy     *ProxyConfig     `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Network   *NetworkConfig   `yaml:"network,omitempty" json:"network,omitempty"`
	Builder   *BuilderConfig   `yaml:"builder,omitempty" json:"builder,omitempty"`
	Registry  *RegistryConfig  `yaml:"registry,omitempty" json:"registry,omitempty"`
	Logs      *LogConfig       `yaml:"logs,omitempty" json:"logs,omitempty"`
	Scheduler *SchedulerConfig `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
}

// Plugin represents a dokku plugin to install.
type Plugin struct {
	URL         string `yaml:"url" json:"url"`
	Committish  string `yaml:"committish,omitempty" json:"committish,omitempty"`
}

// Service represents a backing service (database, cache, etc).
type Service struct {
	Type         string `yaml:"type" json:"type"`
	ImageVersion string `yaml:"image_version,omitempty" json:"image_version,omitempty"`
}

// App represents an application deployment.
type App struct {
	Image         string            `yaml:"image,omitempty" json:"image,omitempty"`
	Git           *GitConfig        `yaml:"git,omitempty" json:"git,omitempty"`
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
	NginxTemplate string                           `yaml:"nginx_template,omitempty" json:"nginx_template,omitempty"`
	Scale         map[string]int                   `yaml:"scale,omitempty" json:"scale,omitempty"`
	Resources     map[string]ResourceConfig        `yaml:"resources,omitempty" json:"resources,omitempty"`
	Checks        *ChecksConfig                    `yaml:"checks,omitempty" json:"checks,omitempty"`
	Builder       *BuilderConfig                   `yaml:"builder,omitempty" json:"builder,omitempty"`
	Registry      *RegistryConfig                  `yaml:"registry,omitempty" json:"registry,omitempty"`
	Maintenance   bool                             `yaml:"maintenance,omitempty" json:"maintenance,omitempty"`
	Scripts       *ScriptsConfig                   `yaml:"scripts,omitempty" json:"scripts,omitempty"`
	Locked        bool                             `yaml:"locked,omitempty" json:"locked,omitempty"`
	Process       *ProcessConfig                   `yaml:"process,omitempty" json:"process,omitempty"`
	Logs          *LogConfig                       `yaml:"logs,omitempty" json:"logs,omitempty"`
	Scheduler     *SchedulerConfig                 `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
	Buildpacks    []string                         `yaml:"buildpacks,omitempty" json:"buildpacks,omitempty"`
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
	HSTS                  bool              `yaml:"hsts,omitempty" json:"hsts,omitempty"`
	HSTSIncludeSubdomains bool              `yaml:"hsts_include_subdomains,omitempty" json:"hsts_include_subdomains,omitempty"`
	HSTSMaxAge            int               `yaml:"hsts_max_age,omitempty" json:"hsts_max_age,omitempty"`
	HSTSPreload           bool              `yaml:"hsts_preload,omitempty" json:"hsts_preload,omitempty"`
	Properties            map[string]string `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// ProxyConfig holds proxy settings for an app.
type ProxyConfig struct {
	Enabled bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Type    string            `yaml:"type,omitempty" json:"type,omitempty"`
	Caddy   map[string]string `yaml:"caddy,omitempty" json:"caddy,omitempty"`
	HAProxy map[string]string `yaml:"haproxy,omitempty" json:"haproxy,omitempty"`
	Traefik map[string]string `yaml:"traefik,omitempty" json:"traefik,omitempty"`
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

// ResourceConfig holds resource limits and reservations for a process type.
type ResourceConfig struct {
	Limits       ResourceValues `yaml:"limits,omitempty" json:"limits,omitempty"`
	Reservations ResourceValues `yaml:"reservations,omitempty" json:"reservations,omitempty"`
}

// ResourceValues holds individual resource values.
type ResourceValues struct {
	CPU            string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory         string `yaml:"memory,omitempty" json:"memory,omitempty"`
	MemorySwap     string `yaml:"memory_swap,omitempty" json:"memory_swap,omitempty"`
	Network        string `yaml:"network,omitempty" json:"network,omitempty"`
	NetworkIngress string `yaml:"network_ingress,omitempty" json:"network_ingress,omitempty"`
	NetworkEgress  string `yaml:"network_egress,omitempty" json:"network_egress,omitempty"`
	NvidiaGPU      string `yaml:"nvidia_gpu,omitempty" json:"nvidia_gpu,omitempty"`
}

// ChecksConfig holds zero-downtime deploy check settings.
type ChecksConfig struct {
	Disabled      []string `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Skipped       []string `yaml:"skipped,omitempty" json:"skipped,omitempty"`
	WaitToRetire  int      `yaml:"wait_to_retire,omitempty" json:"wait_to_retire,omitempty"`
}

// BuilderConfig holds builder settings for an app.
type BuilderConfig struct {
	Selected          string `yaml:"selected,omitempty" json:"selected,omitempty"`
	BuildDir          string `yaml:"build_dir,omitempty" json:"build_dir,omitempty"`
	DockerfilePath    string `yaml:"dockerfile_path,omitempty" json:"dockerfile_path,omitempty"`
	PackProjecttomlPath string `yaml:"pack_projecttoml_path,omitempty" json:"pack_projecttoml_path,omitempty"`
	NixpacksTomlPath  string `yaml:"nixpacks_toml_path,omitempty" json:"nixpacks_toml_path,omitempty"`
	HerokuishAllowed  string `yaml:"herokuish_allowed,omitempty" json:"herokuish_allowed,omitempty"`
}

// LogConfig holds log management settings for an app.
type LogConfig struct {
	MaxSize       string `yaml:"max_size,omitempty" json:"max_size,omitempty"`
	VectorImage   string `yaml:"vector_image,omitempty" json:"vector_image,omitempty"`
	VectorSink    string `yaml:"vector_sink,omitempty" json:"vector_sink,omitempty"`
	AppLabelAlias string `yaml:"app_label_alias,omitempty" json:"app_label_alias,omitempty"`
}

// SchedulerConfig holds scheduler settings for an app.
type SchedulerConfig struct {
	Selected              string `yaml:"selected,omitempty" json:"selected,omitempty"`
	DockerLocalInitProcess       string `yaml:"docker_local_init_process,omitempty" json:"docker_local_init_process,omitempty"`
	DockerLocalParallelScheduleCount string `yaml:"docker_local_parallel_schedule_count,omitempty" json:"docker_local_parallel_schedule_count,omitempty"`
}

// RegistryConfig holds Docker registry settings for an app.
type RegistryConfig struct {
	Server        string `yaml:"server,omitempty" json:"server,omitempty"`
	ImageRepo     string `yaml:"image_repo,omitempty" json:"image_repo,omitempty"`
	PushOnRelease bool   `yaml:"push_on_release,omitempty" json:"push_on_release,omitempty"`
	PushExtraTags string `yaml:"push_extra_tags,omitempty" json:"push_extra_tags,omitempty"`
}

// ScriptsConfig holds deployment task scripts for app.json.
type ScriptsConfig struct {
	Predeploy  string `yaml:"predeploy,omitempty" json:"predeploy,omitempty"`
	Postdeploy string `yaml:"postdeploy,omitempty" json:"postdeploy,omitempty"`
}

// ProcessConfig holds process management settings.
type ProcessConfig struct {
	RestartPolicy string `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty"`
	ProcfilePath  string `yaml:"procfile_path,omitempty" json:"procfile_path,omitempty"`
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
		if app.SSL != nil && (app.SSL.CertFile == "") != (app.SSL.KeyFile == "") {
			return fmt.Errorf("app %q: ssl requires both cert_file and key_file", name)
		}
		// Validate service links reference declared services
		for svcType, svcName := range app.Links {
			if df.Services != nil {
				if svc, ok := df.Services[svcName]; ok && svc.Type != svcType {
					return fmt.Errorf("app %q: link %s=%s has type %q but service %q has type %q", name, svcType, svcName, svcType, svcName, svc.Type)
				}
			}
		}
		// Validate mail link references a declared mail service
		if app.Mail != "" && df.MailServices != nil {
			if _, ok := df.MailServices[app.Mail]; !ok {
				return fmt.Errorf("app %q: mail %q not found in mail_services", name, app.Mail)
			}
		}
		// Validate auth directory reference
		if app.Auth != nil && app.Auth.Directory != "" && df.AuthDirectories != nil {
			if _, ok := df.AuthDirectories[app.Auth.Directory]; !ok {
				return fmt.Errorf("app %q: auth directory %q not found in auth_directories", name, app.Auth.Directory)
			}
		}
		// Validate auth frontend reference
		if app.Auth != nil && app.Auth.Protected != "" && df.AuthFrontends != nil {
			if _, ok := df.AuthFrontends[app.Auth.Protected]; !ok {
				return fmt.Errorf("app %q: auth frontend %q not found in auth_frontends", name, app.Auth.Protected)
			}
		}
	}
	// Validate auth frontend directory references
	for name, fe := range df.AuthFrontends {
		if fe.Directory != "" && df.AuthDirectories != nil {
			if _, ok := df.AuthDirectories[fe.Directory]; !ok {
				return fmt.Errorf("auth frontend %q: directory %q not found in auth_directories", name, fe.Directory)
			}
		}
	}
	return nil
}

// FilterByApp returns a new Dokkufile containing only the specified app
// and its linked services, plugins, mail services, and auth config.
func (df *Dokkufile) FilterByApp(appName string) *Dokkufile {
	app, ok := df.Apps[appName]
	if !ok {
		return &Dokkufile{Version: df.Version}
	}

	filtered := &Dokkufile{
		Version: df.Version,
		Global:  df.Global,
		Apps:    map[string]App{appName: app},
	}

	// Include linked services and their plugins
	if len(app.Links) > 0 {
		filtered.Services = map[string]Service{}
		for _, svcName := range app.Links {
			if svc, ok := df.Services[svcName]; ok {
				filtered.Services[svcName] = svc
				// Include the plugin for this service type
				if df.Plugins != nil {
					if plugin, ok := df.Plugins[svc.Type]; ok {
						if filtered.Plugins == nil {
							filtered.Plugins = map[string]Plugin{}
						}
						filtered.Plugins[svc.Type] = plugin
					}
				}
			}
		}
	}

	// Include linked mail service
	if app.Mail != "" && df.MailServices != nil {
		if svc, ok := df.MailServices[app.Mail]; ok {
			filtered.MailServices = map[string]MailService{app.Mail: svc}
		}
	}

	// Include linked auth directory
	if app.Auth != nil && app.Auth.Directory != "" && df.AuthDirectories != nil {
		if dir, ok := df.AuthDirectories[app.Auth.Directory]; ok {
			filtered.AuthDirectories = map[string]AuthDirectory{app.Auth.Directory: dir}
		}
	}

	// Include protecting auth frontend
	if app.Auth != nil && app.Auth.Protected != "" && df.AuthFrontends != nil {
		if fe, ok := df.AuthFrontends[app.Auth.Protected]; ok {
			filtered.AuthFrontends = map[string]AuthFrontend{app.Auth.Protected: fe}
		}
	}

	return filtered
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
