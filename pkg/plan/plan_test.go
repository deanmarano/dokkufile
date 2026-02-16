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

func TestDetectRemovedApp(t *testing.T) {
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {Image: "nginx:latest"},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp":    {Image: "nginx:latest"},
			"oldapp":   {Image: "redis:6"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.App == "oldapp" && s.Action == DestroyApp {
			found = true
		}
	}
	if !found {
		t.Error("did not find destroy step for removed app")
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

func TestDetectRemovedService(t *testing.T) {
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

	found := false
	for _, s := range p.Steps {
		if s.Service == "mydb" && s.Action == DestroyService {
			found = true
			if s.ServiceType != "postgres" {
				t.Errorf("expected ServiceType %q, got %q", "postgres", s.ServiceType)
			}
		}
	}
	if !found {
		t.Error("did not find destroy step for removed service")
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

func TestDetectMailServiceDestroy(t *testing.T) {
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{
		Version: "1",
		MailServices: map[string]schema.MailService{
			"mymail": {Provider: "smtp"},
		},
	}

	p := Diff(desired, actual)

	found := false
	for _, s := range p.Steps {
		if s.Service == "mymail" && s.Action == DestroyMailService {
			found = true
		}
	}
	if !found {
		t.Error("did not find destroy mail service step")
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
			},
		},
	}

	p := Diff(state, state)

	if len(p.Steps) != 0 {
		t.Errorf("expected no steps, got %d: %v", len(p.Steps), p.String())
	}
}
