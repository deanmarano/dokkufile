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
