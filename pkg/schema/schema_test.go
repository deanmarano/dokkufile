package schema

import (
	"strings"
	"testing"
)

func TestValidateOK(t *testing.T) {
	df := &Dokkufile{
		Version:  "1",
		Services: map[string]Service{"db": {Type: "postgres"}},
		Apps: map[string]App{
			"web": {
				Image: "myapp:latest",
				Links: map[string]string{"postgres": "db"},
			},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateImageAndGitMutuallyExclusive(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps: map[string]App{
			"web": {
				Image: "myapp:latest",
				Git:   &GitConfig{Repo: "https://github.com/org/app"},
			},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for image+git.repo")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateImageWithoutGitOK(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps:    map[string]App{"web": {Image: "myapp:latest"}},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGitWithoutImageOK(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps: map[string]App{
			"web": {Git: &GitConfig{Repo: "https://github.com/org/app"}},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLetsEncryptAndSSLMutuallyExclusive(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps: map[string]App{
			"web": {
				LetsEncrypt: true,
				SSL:         &SSLConfig{CertFile: "/cert.pem", KeyFile: "/key.pem"},
			},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for letsencrypt+ssl")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSSLRequiresBothFiles(t *testing.T) {
	tests := []struct {
		name string
		ssl  SSLConfig
	}{
		{"cert only", SSLConfig{CertFile: "/cert.pem"}},
		{"key only", SSLConfig{KeyFile: "/key.pem"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			df := &Dokkufile{
				Version: "1",
				Apps:    map[string]App{"web": {SSL: &tc.ssl}},
			}
			err := df.Validate()
			if err == nil {
				t.Fatal("expected error for partial ssl config")
			}
			if !strings.Contains(err.Error(), "both cert_file and key_file") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSSLBothFilesOK(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps: map[string]App{
			"web": {SSL: &SSLConfig{CertFile: "/cert.pem", KeyFile: "/key.pem"}},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateLinkTypeMismatch(t *testing.T) {
	df := &Dokkufile{
		Version:  "1",
		Services: map[string]Service{"db": {Type: "postgres"}},
		Apps: map[string]App{
			"web": {
				Links: map[string]string{"redis": "db"},
			},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for link type mismatch")
	}
	if !strings.Contains(err.Error(), "has type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLinkTypeMatchOK(t *testing.T) {
	df := &Dokkufile{
		Version:  "1",
		Services: map[string]Service{"db": {Type: "postgres"}},
		Apps: map[string]App{
			"web": {Links: map[string]string{"postgres": "db"}},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMailNotFound(t *testing.T) {
	df := &Dokkufile{
		Version:      "1",
		MailServices: map[string]MailService{},
		Apps: map[string]App{
			"web": {Mail: "missing"},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for missing mail service")
	}
	if !strings.Contains(err.Error(), "not found in mail_services") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMailFoundOK(t *testing.T) {
	df := &Dokkufile{
		Version:      "1",
		MailServices: map[string]MailService{"default": {Provider: ""}},
		Apps: map[string]App{
			"web": {Mail: "default"},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAuthDirectoryNotFound(t *testing.T) {
	df := &Dokkufile{
		Version:         "1",
		AuthDirectories: map[string]AuthDirectory{},
		Apps: map[string]App{
			"web": {Auth: &AuthConfig{Directory: "missing"}},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for missing auth directory")
	}
	if !strings.Contains(err.Error(), "not found in auth_directories") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthFrontendNotFound(t *testing.T) {
	df := &Dokkufile{
		Version:       "1",
		AuthFrontends: map[string]AuthFrontend{},
		Apps: map[string]App{
			"web": {Auth: &AuthConfig{Protected: "missing"}},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for missing auth frontend")
	}
	if !strings.Contains(err.Error(), "not found in auth_frontends") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthFrontendDirectoryNotFound(t *testing.T) {
	df := &Dokkufile{
		Version:         "1",
		AuthDirectories: map[string]AuthDirectory{},
		AuthFrontends: map[string]AuthFrontend{
			"authelia": {Directory: "missing"},
		},
	}
	err := df.Validate()
	if err == nil {
		t.Fatal("expected error for auth frontend referencing missing directory")
	}
	if !strings.Contains(err.Error(), "not found in auth_directories") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthFrontendDirectoryFoundOK(t *testing.T) {
	df := &Dokkufile{
		Version:         "1",
		AuthDirectories: map[string]AuthDirectory{"ldap": {Provider: "LDAP"}},
		AuthFrontends: map[string]AuthFrontend{
			"authelia": {Directory: "ldap"},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMailWithoutMailServicesMapOK(t *testing.T) {
	// When MailServices is nil, the check is skipped (mail may be managed externally).
	df := &Dokkufile{
		Version: "1",
		Apps: map[string]App{
			"web": {Mail: "default"},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- FilterByApp tests ---

func TestFilterByAppIncludesPlugins(t *testing.T) {
	df := &Dokkufile{
		Version:  "1",
		Services: map[string]Service{"db": {Type: "postgres"}, "cache": {Type: "redis"}},
		Plugins: map[string]Plugin{
			"postgres": {URL: "https://github.com/dokku/dokku-postgres.git"},
			"redis":    {URL: "https://github.com/dokku/dokku-redis.git"},
		},
		Apps: map[string]App{
			"web": {Links: map[string]string{"postgres": "db"}},
			"api": {Links: map[string]string{"redis": "cache"}},
		},
	}

	filtered := df.FilterByApp("web")
	if len(filtered.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(filtered.Apps))
	}
	if len(filtered.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(filtered.Services))
	}
	if _, ok := filtered.Services["db"]; !ok {
		t.Error("expected db service")
	}
	if len(filtered.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(filtered.Plugins))
	}
	if _, ok := filtered.Plugins["postgres"]; !ok {
		t.Error("expected postgres plugin")
	}
}

func TestFilterByAppPreservesGlobal(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Global:  &GlobalConfig{Domains: []string{"dokku.example.com"}},
		Apps:    map[string]App{"web": {Image: "myapp"}},
	}

	filtered := df.FilterByApp("web")
	if filtered.Global == nil {
		t.Fatal("expected global config to be preserved")
	}
	if len(filtered.Global.Domains) != 1 || filtered.Global.Domains[0] != "dokku.example.com" {
		t.Errorf("expected global domains preserved, got %v", filtered.Global.Domains)
	}
}

func TestFilterByAppNotFound(t *testing.T) {
	df := &Dokkufile{
		Version: "1",
		Apps:    map[string]App{"web": {}},
	}

	filtered := df.FilterByApp("nonexistent")
	if len(filtered.Apps) != 0 {
		t.Fatalf("expected 0 apps, got %d", len(filtered.Apps))
	}
}

func TestValidateLinkToUndeclaredServiceOK(t *testing.T) {
	// Linking to a service not in the Dokkufile is allowed (it may already exist on the server).
	df := &Dokkufile{
		Version:  "1",
		Services: map[string]Service{},
		Apps: map[string]App{
			"web": {Links: map[string]string{"postgres": "external-db"}},
		},
	}
	if err := df.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
