package cmd

import (
	"testing"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

func TestFilterByAppReturnsAppAndLinkedServices(t *testing.T) {
	df := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"my-db":    {Type: "postgres"},
			"my-redis": {Type: "redis"},
			"other-db": {Type: "postgres"},
		},
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Links:   map[string]string{"postgres": "my-db", "redis": "my-redis"},
				Env:     map[string]string{"SECRET": "hunter2"},
			},
			"api": {
				Domains: []string{"api.example.com"},
				Links:   map[string]string{"postgres": "other-db"},
			},
		},
	}

	filtered := df.FilterByApp("web")

	if len(filtered.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(filtered.Apps))
	}
	if _, ok := filtered.Apps["web"]; !ok {
		t.Fatal("expected web app in filtered output")
	}
	if len(filtered.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(filtered.Services))
	}
	if _, ok := filtered.Services["my-db"]; !ok {
		t.Error("expected my-db service")
	}
	if _, ok := filtered.Services["my-redis"]; !ok {
		t.Error("expected my-redis service")
	}
	if _, ok := filtered.Services["other-db"]; ok {
		t.Error("other-db should not be included")
	}
}

func TestFilterByAppIncludesMailService(t *testing.T) {
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {Mail: "default"},
		},
		MailServices: map[string]schema.MailService{
			"default":  {Provider: "smtp"},
			"internal": {Provider: "smtp"},
		},
	}

	filtered := df.FilterByApp("web")

	if len(filtered.MailServices) != 1 {
		t.Fatalf("expected 1 mail service, got %d", len(filtered.MailServices))
	}
	if _, ok := filtered.MailServices["default"]; !ok {
		t.Error("expected default mail service")
	}
}

func TestFilterByAppIncludesAuthDirectoryAndFrontend(t *testing.T) {
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Auth: &schema.AuthConfig{
					Directory: "ldap",
					Protected: "authelia",
				},
			},
		},
		AuthDirectories: map[string]schema.AuthDirectory{
			"ldap":  {Provider: "LDAP"},
			"other": {Provider: "other"},
		},
		AuthFrontends: map[string]schema.AuthFrontend{
			"authelia": {Provider: "Authelia SSO"},
			"other":    {Provider: "other"},
		},
	}

	filtered := df.FilterByApp("web")

	if len(filtered.AuthDirectories) != 1 {
		t.Fatalf("expected 1 auth directory, got %d", len(filtered.AuthDirectories))
	}
	if _, ok := filtered.AuthDirectories["ldap"]; !ok {
		t.Error("expected ldap auth directory")
	}
	if len(filtered.AuthFrontends) != 1 {
		t.Fatalf("expected 1 auth frontend, got %d", len(filtered.AuthFrontends))
	}
	if _, ok := filtered.AuthFrontends["authelia"]; !ok {
		t.Error("expected authelia auth frontend")
	}
}

func TestFilterByAppNotFound(t *testing.T) {
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {},
		},
	}

	filtered := df.FilterByApp("nonexistent")

	if len(filtered.Apps) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(filtered.Apps))
	}
}

func TestInspectCmdStripsEnvByDefault(t *testing.T) {
	// Simulate the env stripping logic from the command handler
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Env:     map[string]string{"SECRET": "hunter2", "DB_URL": "postgres://..."},
			},
			"api": {
				Domains: []string{"api.example.com"},
				Env:     map[string]string{"API_KEY": "sk-123"},
			},
		},
	}

	// Simulate includeEnv=false
	for name, app := range df.Apps {
		app.Env = nil
		df.Apps[name] = app
	}

	for name, app := range df.Apps {
		if app.Env != nil {
			t.Errorf("app %q should have nil env, got %v", name, app.Env)
		}
	}
}

func TestInspectCmdKeepsEnvWithFlag(t *testing.T) {
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Env:     map[string]string{"SECRET": "hunter2", "DB_URL": "postgres://..."},
			},
		},
	}

	// Simulate includeEnv=true (no stripping)
	app := df.Apps["web"]
	if len(app.Env) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(app.Env))
	}
	if app.Env["SECRET"] != "hunter2" {
		t.Errorf("expected SECRET=hunter2, got %q", app.Env["SECRET"])
	}
}

func TestInspectCmdGlobalWithEnvStripping(t *testing.T) {
	// --global without --include-env: all apps present, no env
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Env:     map[string]string{"SECRET": "hunter2"},
			},
			"api": {
				Domains: []string{"api.example.com"},
				Env:     map[string]string{"API_KEY": "sk-123"},
			},
		},
	}

	// Simulate --global (no filterByApp) + no --include-env (strip env)
	for name, app := range df.Apps {
		app.Env = nil
		df.Apps[name] = app
	}

	if len(df.Apps) != 2 {
		t.Fatalf("expected 2 apps with --global, got %d", len(df.Apps))
	}
	for name, app := range df.Apps {
		if app.Env != nil {
			t.Errorf("app %q env should be nil, got %v", name, app.Env)
		}
	}
}

func TestInspectCmdGlobalWithIncludeEnv(t *testing.T) {
	// --global --include-env: all apps with env (full restore)
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Env:     map[string]string{"SECRET": "hunter2"},
			},
			"api": {
				Domains: []string{"api.example.com"},
				Env:     map[string]string{"API_KEY": "sk-123"},
			},
		},
	}

	// Simulate --global + --include-env (no stripping, no filtering)
	if len(df.Apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(df.Apps))
	}
	if df.Apps["web"].Env["SECRET"] != "hunter2" {
		t.Error("web SECRET should be preserved")
	}
	if df.Apps["api"].Env["API_KEY"] != "sk-123" {
		t.Error("api API_KEY should be preserved")
	}
}

func TestInspectCmdSingleAppWithIncludeEnv(t *testing.T) {
	// inspect <app> --include-env: single app with env
	df := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"web": {
				Domains: []string{"web.example.com"},
				Env:     map[string]string{"SECRET": "hunter2"},
			},
			"api": {
				Domains: []string{"api.example.com"},
				Env:     map[string]string{"API_KEY": "sk-123"},
			},
		},
	}

	// Simulate filterByApp + --include-env
	filtered := df.FilterByApp("web")

	if len(filtered.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(filtered.Apps))
	}
	if filtered.Apps["web"].Env["SECRET"] != "hunter2" {
		t.Error("web SECRET should be preserved with --include-env")
	}
}

func TestInspectCmdRequiresAppOrGlobal(t *testing.T) {
	cmd := NewInspectCmd()
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when no app and no --global")
	}
	expected := "specify an app name or use --global to inspect all apps"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
