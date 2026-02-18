package plan

import (
	"testing"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

func TestDetectImageChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:1.24"},
		},
	}

	p := Diff(desired, actual)

	if len(p.Steps) == 0 {
		t.Fatal("expected at least one step for image change")
	}

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "image" {
			found = true
			if s.OldValue != "nginx:1.24" || s.NewValue != "nginx:latest" {
				t.Errorf("unexpected values: old=%q new=%q", s.OldValue, s.NewValue)
			}
		}
	}
	if !found {
		t.Error("did not find image update step")
	}
}

func TestDetectNewApp(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp":  {Image: "nginx:latest"},
			"newapp": {Image: "redis:7"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "newapp" && s.Action == CreateApp {
			found = true
		}
	}
	if !found {
		t.Error("did not find create step for new app")
	}
}

func TestUnmanagedAppIgnored(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp":  {Image: "nginx:latest"},
			"oldapp": {Image: "redis:6"},
		},
	}

	p := Diff(desired, actual)

	for _, s := range p.Steps {
		if s.App == "oldapp" {
			t.Errorf("should not generate steps for unmanaged app, got %s", s.Action)
		}
	}
}

func TestNoChanges(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(state, state)

	if len(p.Steps) != 0 {
		t.Errorf("expected no steps, got %d", len(p.Steps))
	}
}

func TestDetectNewService(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "mydb" && s.Action == CreateService {
			found = true
			if s.ServiceType != "postgres" {
				t.Errorf("expected ServiceType %q, got %q", "postgres", s.ServiceType)
			}
		}
	}
	if !found {
		t.Error("did not find create step for new service")
	}
}

func TestUnmanagedServiceIgnored(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres"},
		},
	}

	p := Diff(desired, actual)

	for _, s := range p.Steps {
		if s.Service == "mydb" {
			t.Errorf("should not generate steps for unmanaged service, got %s", s.Action)
		}
	}
}

func TestDetectDomainChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"new.example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"old.example.com"},
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "domains" {
			found = true
		}
	}
	if !found {
		t.Error("did not find domain update step")
	}
}

func TestDetectEnvChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Env:   map[string]string{"FOO": "bar"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Env:   map[string]string{"FOO": "baz"},
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "env" {
			found = true
		}
	}
	if !found {
		t.Error("did not find env update step")
	}
}

func TestDetectLinkChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Links: map[string]string{"postgres": "mydb"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "links" {
			found = true
		}
	}
	if !found {
		t.Error("did not find link update step")
	}
}

func TestDetectDockerOptionsChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				DockerOptions: schema.DockerOptions{
					Deploy: []string{"--restart=always"},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "docker_options" {
			found = true
		}
	}
	if !found {
		t.Error("did not find docker_options update step")
	}
}

func TestDetectScaleChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Scale: map[string]int{"web": 2},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Scale: map[string]int{"web": 1},
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "scale" {
			found = true
		}
	}
	if !found {
		t.Error("did not find scale update step")
	}
}

func TestDetectLetsEncryptChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:       "nginx:latest",
				LetsEncrypt: true,
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:       "nginx:latest",
				LetsEncrypt: false,
			},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "letsencrypt" {
			found = true
		}
	}
	if !found {
		t.Error("did not find letsencrypt update step")
	}
}

func TestDetectGitChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Git:   &schema.GitConfig{Branch: "main", Repo: "https://github.com/example/repo.git"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "git" {
			found = true
		}
	}
	if !found {
		t.Error("did not find git update step")
	}
}

func TestDetectNetworkChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Network: &schema.NetworkConfig{InitialNetwork: "mynet"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "network" {
			found = true
		}
	}
	if !found {
		t.Error("did not find network update step")
	}
}

func TestDetectNginxChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Nginx: &schema.NginxConfig{HSTS: true, HSTSMaxAge: 31536000},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "nginx" {
			found = true
		}
	}
	if !found {
		t.Error("did not find nginx update step")
	}
}

func TestDetectProxyChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Proxy: &schema.ProxyConfig{Enabled: true, Type: "nginx"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "proxy" {
			found = true
		}
	}
	if !found {
		t.Error("did not find proxy update step")
	}
}

func TestDetectSSLChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				SSL:   &schema.SSLConfig{CertFile: "/path/to/cert", KeyFile: "/path/to/key"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "ssl" {
			found = true
		}
	}
	if !found {
		t.Error("did not find ssl update step")
	}
}

func TestDetectHealthcheckChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Healthchecks: map[string][]schema.HealthcheckConfig{
					"web": {{Path: "/health", Timeout: 10}},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "healthchecks" {
			found = true
		}
	}
	if !found {
		t.Error("did not find healthchecks update step")
	}
}

func TestDetectCronChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Cron:  []schema.CronJob{{Command: "rake db:backup", Schedule: "@daily"}},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "cron" {
			found = true
		}
	}
	if !found {
		t.Error("did not find cron update step")
	}
}

func TestDetectNginxTemplateChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:         "nginx:latest",
				NginxTemplate: "server { listen 80; }",
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "nginx_template" {
			found = true
		}
	}
	if !found {
		t.Error("did not find nginx_template update step")
	}
}

func TestDetectMailServiceCreate(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		MailServices: map[string]schema.MailService{
			"mymail": {Provider: "smtp", Config: map[string]string{"host": "smtp.example.com"}},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "mymail" && s.Action == CreateMailService {
			found = true
		}
	}
	if !found {
		t.Error("did not find create mail service step")
	}
}

func TestUnmanagedMailServiceIgnored(t *testing.T) {
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{
		Version: "1",
		MailServices: map[string]schema.MailService{
			"mymail": {Provider: "smtp"},
		},
	}

	p := Diff(desired, actual)

	for _, s := range p.Steps {
		if s.Service == "mymail" {
			t.Errorf("should not generate steps for unmanaged mail service, got %s", s.Action)
		}
	}
}

func TestDetectAuthDirectoryCreate(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		AuthDirectories: map[string]schema.AuthDirectory{
			"mydir": {Provider: "ldap"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "mydir" && s.Action == CreateAuthDirectory {
			found = true
		}
	}
	if !found {
		t.Error("did not find create auth directory step")
	}
}

func TestDetectAuthFrontendCreate(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		AuthFrontends: map[string]schema.AuthFrontend{
			"myfe": {Provider: "oauth2", Directory: "mydir"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "myfe" && s.Action == CreateAuthFrontend {
			found = true
		}
	}
	if !found {
		t.Error("did not find create auth frontend step")
	}
}

func TestDetectResourceChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Resources: map[string]schema.ResourceConfig{
					"web": {Limits: schema.ResourceValues{CPU: "2", Memory: "1024m"}},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "resources" {
			found = true
		}
	}
	if !found {
		t.Error("did not find resources update step")
	}
}

func TestDetectChecksChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:  "nginx:latest",
				Checks: &schema.ChecksConfig{Disabled: []string{"worker"}, WaitToRetire: 30},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "checks" {
			found = true
		}
	}
	if !found {
		t.Error("did not find checks update step")
	}
}

func TestDetectBuilderChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Builder: &schema.BuilderConfig{Selected: "herokuish"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "builder" {
			found = true
		}
	}
	if !found {
		t.Error("did not find builder update step")
	}
}

func TestDetectRegistryChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:    "nginx:latest",
				Registry: &schema.RegistryConfig{Server: "registry.example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "registry" {
			found = true
		}
	}
	if !found {
		t.Error("did not find registry update step")
	}
}

func TestDetectMaintenanceChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:       "nginx:latest",
				Maintenance: true,
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "maintenance" {
			found = true
		}
	}
	if !found {
		t.Error("did not find maintenance update step")
	}
}

func TestDetectScriptsChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Scripts: &schema.ScriptsConfig{Predeploy: "rake db:migrate"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "scripts" {
			found = true
		}
	}
	if !found {
		t.Error("did not find scripts update step")
	}
}

func TestDetectLockedChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest", Locked: true}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "locked" {
			found = true
		}
	}
	if !found {
		t.Error("did not find locked update step")
	}
}

func TestDetectProcessChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Process: &schema.ProcessConfig{RestartPolicy: "always"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "process" {
			found = true
		}
	}
	if !found {
		t.Error("did not find process update step")
	}
}

func TestDetectNginxPropertyChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
				Nginx: &schema.NginxConfig{
					Properties: map[string]string{"client-max-body-size": "50m"},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "myapp" && s.Action == UpdateApp && s.Field == "nginx" {
			found = true
		}
	}
	if !found {
		t.Error("did not find nginx update step for property change")
	}
}

func TestNoChangesNewFields(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Git:     &schema.GitConfig{Branch: "main"},
				Network: &schema.NetworkConfig{InitialNetwork: "mynet"},
				Nginx:   &schema.NginxConfig{HSTS: true},
				Proxy:   &schema.ProxyConfig{Enabled: true, Type: "nginx"},
				Resources: map[string]schema.ResourceConfig{
					"web": {Limits: schema.ResourceValues{CPU: "1"}},
				},
				Checks:      &schema.ChecksConfig{Disabled: []string{"worker"}},
				Builder:     &schema.BuilderConfig{Selected: "herokuish"},
				Registry:    &schema.RegistryConfig{Server: "registry.example.com"},
				Maintenance: true,
				Scripts:     &schema.ScriptsConfig{Predeploy: "rake db:migrate"},
				Locked:      true,
				Process:     &schema.ProcessConfig{RestartPolicy: "always"},
				Logs:        &schema.LogConfig{MaxSize: "20m", VectorSink: "console://"},
				Scheduler:   &schema.SchedulerConfig{Selected: "docker-local", DockerLocalInitProcess: "true"},
				Buildpacks:  []string{"https://github.com/heroku/heroku-buildpack-nodejs"},
			},
		},
	}

	p := Diff(state, state)

	if len(p.Steps) != 0 {
		t.Errorf("expected no steps, got %d: %v", len(p.Steps), p.String())
	}
}

func TestDetectLogChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Logs: &schema.LogConfig{MaxSize: "50m"}},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Logs: &schema.LogConfig{MaxSize: "20m"}},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Field == "logs" {
			found = true
		}
	}
	if !found {
		t.Error("expected logs update step")
	}
}

func TestDetectSchedulerChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Scheduler: &schema.SchedulerConfig{Selected: "docker-local"}},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx"},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Field == "scheduler" {
			found = true
		}
	}
	if !found {
		t.Error("expected scheduler update step")
	}
}

func TestDetectBuildpacksChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Buildpacks: []string{"https://github.com/heroku/heroku-buildpack-nodejs"}},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx"},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Field == "buildpacks" {
			found = true
		}
	}
	if !found {
		t.Error("expected buildpacks update step")
	}
}

func TestDetectPluginChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Plugins: map[string]schema.Plugin{
			"letsencrypt": {URL: "https://github.com/dokku/dokku-letsencrypt.git"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == InstallPlugin && s.Service == "letsencrypt" {
			found = true
		}
	}
	if !found {
		t.Error("expected install plugin step")
	}
}

func TestUnmanagedPluginIgnored(t *testing.T) {
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{
		Version: "1",
		Plugins: map[string]schema.Plugin{
			"letsencrypt": {URL: "https://github.com/dokku/dokku-letsencrypt.git"},
		},
	}

	p := Diff(desired, actual)
	for _, s := range p.Steps {
		if s.Service == "letsencrypt" {
			t.Errorf("should not generate steps for unmanaged plugin, got %s", s.Action)
		}
	}
}

func TestDetectProxyPropertyChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Proxy: &schema.ProxyConfig{
					Enabled: true,
					Type:    "caddy",
					Caddy:   map[string]string{"tls-internal": "true"},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Proxy: &schema.ProxyConfig{
					Enabled: true,
					Type:    "caddy",
				},
			},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Field == "proxy" {
			found = true
		}
	}
	if !found {
		t.Error("expected proxy update step for caddy property change")
	}
}

func TestNoChangeProxySame(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Proxy: &schema.ProxyConfig{
					Enabled: true,
					Type:    "traefik",
					Traefik: map[string]string{"log-level": "DEBUG"},
				},
			},
		},
	}

	p := Diff(state, state)
	if len(p.Steps) != 0 {
		t.Errorf("expected no steps, got %d: %v", len(p.Steps), p.String())
	}
}

func TestDetectGlobalDomainChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Domains: []string{"example.com", "example.org"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Domains: []string{"example.com"},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateGlobal && s.Field == "domains" {
			found = true
		}
	}
	if !found {
		t.Error("expected UpdateGlobal domains step")
	}
}

func TestDetectGlobalNginxChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Nginx: &schema.NginxConfig{HSTS: true, HSTSMaxAge: 31536000},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Nginx: &schema.NginxConfig{HSTS: false},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateGlobal && s.Field == "nginx" {
			found = true
		}
	}
	if !found {
		t.Error("expected UpdateGlobal nginx step")
	}
}

func TestDetectGlobalLogsChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Logs: &schema.LogConfig{MaxSize: "10m"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateGlobal && s.Field == "logs" {
			found = true
		}
	}
	if !found {
		t.Error("expected UpdateGlobal logs step")
	}
}

func TestNoChangeGlobalSame(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Domains: []string{"example.com"},
			Nginx:   &schema.NginxConfig{HSTS: true},
			Logs:    &schema.LogConfig{MaxSize: "10m"},
		},
	}

	p := Diff(state, state)
	for _, s := range p.Steps {
		if s.Action == UpdateGlobal {
			t.Errorf("expected no global steps, got: %s", s.Field)
		}
	}
}

func TestNoChangeGlobalBothNil(t *testing.T) {
	state := &schema.Dokkufile{Version: "1"}
	p := Diff(state, state)
	for _, s := range p.Steps {
		if s.Action == UpdateGlobal {
			t.Errorf("expected no global steps, got: %s", s.Field)
		}
	}
}

func TestDetectSecretsChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx",
				Secrets: []string{"DATABASE_URL", "API_KEY"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx",
				Secrets: []string{"DATABASE_URL"},
			},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "secrets" {
			found = true
		}
	}
	if !found {
		t.Error("expected secrets update step")
	}
}

func TestEnvEqualExcludesSecrets(t *testing.T) {
	// Env differs only on a secret key — should be considered equal
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx",
				Env:     map[string]string{"APP_NAME": "myapp"},
				Secrets: []string{"DATABASE_URL"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Env:   map[string]string{"APP_NAME": "myapp", "DATABASE_URL": "postgres://..."},
			},
		},
	}

	p := Diff(desired, actual)
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "env" {
			t.Error("expected no env update — secret key should be excluded from comparison")
		}
	}
}

func TestUpdateGlobalString(t *testing.T) {
	p := &Plan{
		Steps: []Step{
			{Action: UpdateGlobal, Field: "domains"},
		},
	}
	s := p.String()
	if s == "" || s == "No changes needed." {
		t.Error("expected non-empty plan string for UpdateGlobal")
	}
}

func TestDetectMailLinkChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Mail: "newmail"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Mail: "oldmail"},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "mail" {
			found = true
			if s.OldValue != "oldmail" || s.NewValue != "newmail" {
				t.Errorf("unexpected mail values: old=%q new=%q", s.OldValue, s.NewValue)
			}
		}
	}
	if !found {
		t.Error("expected mail update step")
	}
}

func TestDetectAuthChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Directory: "newdir"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Directory: "olddir"},
			},
		},
	}

	p := Diff(desired, actual)
	found := false
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "auth" {
			found = true
		}
	}
	if !found {
		t.Error("expected auth update step")
	}
}

func TestNoChangeMailSame(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx", Mail: "mymail"},
		},
	}

	p := Diff(state, state)
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "mail" {
			t.Error("expected no mail update step")
		}
	}
}

func TestNoChangeAuthSame(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Directory: "mydir", Protected: "myfe"},
			},
		},
	}

	p := Diff(state, state)
	for _, s := range p.Steps {
		if s.Action == UpdateApp && s.Field == "auth" {
			t.Error("expected no auth update step")
		}
	}
}

func TestNoFalsePositiveUnmanagedFields(t *testing.T) {
	// When the dokkufile doesn't declare ports, scale, proxy, or healthchecks,
	// we should NOT report drift even if the actual state has values for them.
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"myapp.example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"myapp.example.com"},
				Ports:   map[string]string{"http:80": "3001"},
				Scale:   map[string]int{"web": 1},
				Proxy:   &schema.ProxyConfig{Enabled: true, Type: "nginx"},
				Healthchecks: map[string][]schema.HealthcheckConfig{
					"web": {{Path: "/", Timeout: 10}},
				},
			},
		},
	}

	p := Diff(desired, actual)

	for _, s := range p.Steps {
		switch s.Field {
		case "ports", "scale", "proxy", "healthchecks":
			t.Errorf("should not report drift for unmanaged field %q", s.Field)
		}
	}
}

func TestPortsEqualDifferentFormats(t *testing.T) {
	// YAML format: {"http": "80:3001"}
	// State reader format: {"http:80": "3001"}
	// These should be considered equal.
	yamlPorts := map[string]string{"http": "80:3001"}
	statePorts := map[string]string{"http:80": "3001"}

	if !portsEqual(yamlPorts, statePorts) {
		t.Error("expected YAML ports and state reader ports to be equal")
	}
}

func TestPortsEqualBothNil(t *testing.T) {
	if !portsEqual(nil, nil) {
		t.Error("expected nil ports to be equal")
	}
}

func TestPortsNotEqual(t *testing.T) {
	a := map[string]string{"http": "80:3001"}
	b := map[string]string{"http": "80:8080"}

	if portsEqual(a, b) {
		t.Error("expected different ports to not be equal")
	}
}

func TestDetectServiceVersionChange(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres", ImageVersion: "16"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres", ImageVersion: "15"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "mydb" && s.Action == UpdateService {
			found = true
			if s.Field != "image_version" {
				t.Errorf("expected field 'image_version', got %q", s.Field)
			}
			if s.OldValue != "15" || s.NewValue != "16" {
				t.Errorf("unexpected version values: old=%q new=%q", s.OldValue, s.NewValue)
			}
		}
	}
	if !found {
		t.Error("did not find update service step for version change")
	}
}

func TestNoChangeServiceVersionSame(t *testing.T) {
	state := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres", ImageVersion: "15"},
		},
	}

	p := Diff(state, state)
	for _, s := range p.Steps {
		if s.Service == "mydb" && s.Action == UpdateService {
			t.Error("expected no update service step when versions match")
		}
	}
}

func TestServiceVersionEmptyDesiredSkips(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres", ImageVersion: "15"},
		},
	}

	p := Diff(desired, actual)
	for _, s := range p.Steps {
		if s.Service == "mydb" && s.Action == UpdateService {
			t.Error("expected no update when desired version is empty (unmanaged)")
		}
	}
}
