package apply

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
)

// RecordingRunner records all commands run for assertion.
type RecordingRunner struct {
	Commands [][]string
}

func (r *RecordingRunner) Run(args ...string) (string, error) {
	cmd := make([]string, len(args))
	copy(cmd, args)
	r.Commands = append(r.Commands, cmd)
	return "", nil
}

func (r *RecordingRunner) hasCommand(args ...string) bool {
	for _, cmd := range r.Commands {
		if len(cmd) == len(args) {
			match := true
			for i := range cmd {
				if cmd[i] != args[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func (r *RecordingRunner) commandStrings() []string {
	var strs []string
	for _, cmd := range r.Commands {
		strs = append(strs, strings.Join(cmd, " "))
	}
	return strs
}

func TestCreateService(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("postgres:create", "mydb") {
		t.Errorf("expected postgres:create mydb, got: %v", runner.commandStrings())
	}
}

func TestDestroyService(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.DestroyService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("postgres:destroy", "mydb", "--force") {
		t.Errorf("expected postgres:destroy mydb --force, got: %v", runner.commandStrings())
	}
}

func TestDestroyApp(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.DestroyApp, App: "oldapp"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("apps:destroy", "oldapp", "--force") {
		t.Errorf("expected apps:destroy oldapp --force, got: %v", runner.commandStrings())
	}
}

func TestCreateApp(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx:latest"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"example.com"},
				Env:     map[string]string{"FOO": "bar"},
				Ports:   map[string]string{"http:80": "5000"},
				Storage: []string{"/data:/app/data"},
				Scale:   map[string]int{"web": 2},
				Links:   map[string]string{"postgres": "mydb"},
				DockerOptions: schema.DockerOptions{
					Deploy: []string{"--restart=always"},
				},
				LetsEncrypt: true,
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	cmds := runner.commandStrings()

	// Check ordering: apps:create must be first
	if len(cmds) == 0 || cmds[0] != "apps:create myapp" {
		t.Errorf("first command should be apps:create myapp, got: %v", cmds)
	}

	checks := [][]string{
		{"apps:create", "myapp"},
		{"git:from-image", "myapp", "nginx:latest"},
		{"domains:set", "myapp", "example.com"},
		{"config:set", "--no-restart", "myapp", "FOO=bar"},
		{"ports:set", "myapp", "http:80:5000"},
		{"storage:mount", "myapp", "/data:/app/data"},
		{"docker-options:add", "myapp", "deploy", "--restart=always"},
		{"ps:scale", "myapp", "web=2"},
		{"postgres:link", "mydb", "myapp"},
		{"letsencrypt:enable", "myapp"},
	}

	for _, check := range checks {
		if !runner.hasCommand(check...) {
			t.Errorf("missing command: %v\ngot: %v", check, cmds)
		}
	}
}

func TestUpdateImage(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "image", OldValue: "nginx:1.24", NewValue: "nginx:latest"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:1.24"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("git:from-image", "myapp", "nginx:latest") {
		t.Errorf("expected git:from-image, got: %v", runner.commandStrings())
	}
}

func TestUpdateDomains(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "domains"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Domains: []string{"new.example.com", "www.example.com"}}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Domains: []string{"old.example.com"}}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("domains:set", "myapp", "new.example.com", "www.example.com") {
		t.Errorf("expected domains:set, got: %v", runner.commandStrings())
	}
}

func TestUpdateEnv(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "env"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Env: map[string]string{"FOO": "new", "BAZ": "added"},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Env: map[string]string{"FOO": "old", "REMOVED": "val"},
		}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Should set changed + new vars
	if !runner.hasCommand("config:set", "--no-restart", "myapp", "BAZ=added", "FOO=new") {
		t.Errorf("expected config:set for changed/new vars, got: %v", runner.commandStrings())
	}
	// Should unset removed vars
	if !runner.hasCommand("config:unset", "--no-restart", "myapp", "REMOVED") {
		t.Errorf("expected config:unset for removed vars, got: %v", runner.commandStrings())
	}
}

func TestUpdateLinks(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "links"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Links: map[string]string{"postgres": "newdb"},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Links: map[string]string{"postgres": "olddb", "redis": "cache"},
		}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Unlink old
	if !runner.hasCommand("postgres:unlink", "olddb", "myapp") {
		t.Errorf("expected postgres:unlink olddb, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("redis:unlink", "cache", "myapp") {
		t.Errorf("expected redis:unlink cache, got: %v", runner.commandStrings())
	}
	// Link new
	if !runner.hasCommand("postgres:link", "newdb", "myapp") {
		t.Errorf("expected postgres:link newdb, got: %v", runner.commandStrings())
	}
}

func TestUpdateStorage(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "storage"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Storage: []string{"/data:/app/data", "/new:/app/new"},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Storage: []string{"/data:/app/data", "/old:/app/old"},
		}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("storage:unmount", "myapp", "/old:/app/old") {
		t.Errorf("expected storage:unmount for removed mount, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("storage:mount", "myapp", "/new:/app/new") {
		t.Errorf("expected storage:mount for new mount, got: %v", runner.commandStrings())
	}
	// Should NOT touch the unchanged mount
	if runner.hasCommand("storage:mount", "myapp", "/data:/app/data") {
		t.Error("should not re-mount unchanged storage")
	}
}

func TestUpdateDockerOptions(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "docker_options"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			DockerOptions: schema.DockerOptions{
				Deploy: []string{"--restart=always"},
				Run:    []string{"--cap-add=NET_ADMIN"},
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			DockerOptions: schema.DockerOptions{
				Deploy: []string{"--restart=on-failure"},
			},
		}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Remove old deploy option
	if !runner.hasCommand("docker-options:remove", "myapp", "deploy", "--restart=on-failure") {
		t.Errorf("expected remove old deploy option, got: %v", runner.commandStrings())
	}
	// Add new deploy option
	if !runner.hasCommand("docker-options:add", "myapp", "deploy", "--restart=always") {
		t.Errorf("expected add new deploy option, got: %v", runner.commandStrings())
	}
	// Add new run option
	if !runner.hasCommand("docker-options:add", "myapp", "run", "--cap-add=NET_ADMIN") {
		t.Errorf("expected add new run option, got: %v", runner.commandStrings())
	}
}

func TestUpdateScale(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "scale"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Scale: map[string]int{"web": 3, "worker": 1}}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Scale: map[string]int{"web": 1}}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("ps:scale", "myapp", "web=3", "worker=1") {
		t.Errorf("expected ps:scale with all procs, got: %v", runner.commandStrings())
	}
}

func TestUpdateLetsEncrypt(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	// Enable
	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "letsencrypt"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {LetsEncrypt: true}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {LetsEncrypt: false}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("letsencrypt:enable", "myapp") {
		t.Errorf("expected letsencrypt:enable, got: %v", runner.commandStrings())
	}

	// Disable
	runner2 := &RecordingRunner{}
	executor2 := &Executor{Runner: runner2}
	desired2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {LetsEncrypt: false}},
	}
	actual2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {LetsEncrypt: true}},
	}

	err = executor2.Execute(p, desired2, actual2)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner2.hasCommand("letsencrypt:disable", "myapp") {
		t.Errorf("expected letsencrypt:disable, got: %v", runner2.commandStrings())
	}
}

func TestDryRun(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner, DryRun: true}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx:latest"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(runner.Commands) != 0 {
		t.Errorf("dry run should not execute commands, but ran: %v", runner.commandStrings())
	}
}

func TestExecuteStopsOnError(t *testing.T) {
	runner := &FailingRunner{failOn: "git:from-image"}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx:latest"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "git:from-image") {
		t.Errorf("error should mention failing command, got: %v", err)
	}
}

func TestUpdateGitConfig(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "git"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Git: &schema.GitConfig{
				Repo:       "https://github.com/example/repo.git",
				Branch:     "main",
				KeepGitDir: true,
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("git:set", "myapp", "deploy-branch", "main") {
		t.Errorf("expected git:set deploy-branch, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("git:set", "myapp", "keep-git-dir", "true") {
		t.Errorf("expected git:set keep-git-dir, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("git:sync", "--build", "myapp", "https://github.com/example/repo.git", "main") {
		t.Errorf("expected git:sync, got: %v", runner.commandStrings())
	}
}

func TestUpdateNetworkConfig(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "network"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Network: &schema.NetworkConfig{
				AttachPostCreate:  "mynet",
				BindAllInterfaces: true,
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("network:set", "myapp", "attach-post-create", "mynet") {
		t.Errorf("expected network:set attach-post-create, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("network:set", "myapp", "bind-all-interfaces", "true") {
		t.Errorf("expected network:set bind-all-interfaces, got: %v", runner.commandStrings())
	}
}

func TestUpdateNginxConfig(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Nginx: &schema.NginxConfig{HSTS: true, HSTSMaxAge: 31536000},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("nginx:set", "myapp", "hsts", "true") {
		t.Errorf("expected nginx:set hsts, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("nginx:set", "myapp", "hsts-max-age", "31536000") {
		t.Errorf("expected nginx:set hsts-max-age, got: %v", runner.commandStrings())
	}
}

func TestUpdateProxyConfig(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "proxy"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Proxy: &schema.ProxyConfig{Enabled: true, Type: "nginx"},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("proxy:enable", "myapp") {
		t.Errorf("expected proxy:enable, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("proxy:set", "myapp", "nginx") {
		t.Errorf("expected proxy:set, got: %v", runner.commandStrings())
	}
}

func TestUpdateSSLRemove(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "ssl"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			SSL: &schema.SSLConfig{},
		}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("certs:remove", "myapp") {
		t.Errorf("expected certs:remove, got: %v", runner.commandStrings())
	}
}

func TestUpdateHealthchecks(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "healthchecks"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Healthchecks: map[string][]schema.HealthcheckConfig{
				"web": {{Path: "/health", Timeout: 10}},
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Should generate an app-json:set command
	found := false
	for _, cmd := range runner.Commands {
		if len(cmd) >= 2 && cmd[0] == "app-json:set" && cmd[1] == "myapp" {
			found = true
			// The third arg should be valid JSON containing healthchecks
			if len(cmd) >= 3 && !strings.Contains(cmd[2], "healthchecks") {
				t.Errorf("app-json:set should contain healthchecks, got: %s", cmd[2])
			}
		}
	}
	if !found {
		t.Errorf("expected app-json:set command, got: %v", runner.commandStrings())
	}
}

func TestUpdateNginxTemplate(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "nginx_template"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			NginxTemplate: "server { listen 80; }",
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("nginx:build-config", "myapp") {
		t.Errorf("expected nginx:build-config, got: %v", runner.commandStrings())
	}
}

func TestCreateMailService(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateMailService, Service: "mymail"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		MailServices: map[string]schema.MailService{
			"mymail": {
				Provider: "smtp",
				Config:   map[string]string{"host": "smtp.example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("mail:create", "mymail") {
		t.Errorf("expected mail:create, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("mail:provider:set", "mymail", "smtp") {
		t.Errorf("expected mail:provider:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("mail:provider:config", "mymail", "host=smtp.example.com") {
		t.Errorf("expected mail:provider:config, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("mail:provider:apply", "mymail") {
		t.Errorf("expected mail:provider:apply, got: %v", runner.commandStrings())
	}
}

func TestDestroyMailService(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.DestroyMailService, Service: "mymail"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("mail:destroy", "mymail", "--force") {
		t.Errorf("expected mail:destroy, got: %v", runner.commandStrings())
	}
}

func TestCreateAuthDirectory(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateAuthDirectory, Service: "mydir"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		AuthDirectories: map[string]schema.AuthDirectory{
			"mydir": {
				Provider: "ldap",
				Config:   map[string]string{"url": "ldap://example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:create", "mydir") {
		t.Errorf("expected auth:create, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:provider:set", "mydir", "ldap") {
		t.Errorf("expected auth:provider:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:provider:config", "mydir", "url=ldap://example.com") {
		t.Errorf("expected auth:provider:config, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:provider:apply", "mydir") {
		t.Errorf("expected auth:provider:apply, got: %v", runner.commandStrings())
	}
}

func TestCreateAuthFrontend(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateAuthFrontend, Service: "myfe"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		AuthFrontends: map[string]schema.AuthFrontend{
			"myfe": {
				Provider:      "oauth2",
				Directory:     "mydir",
				ProtectedApps: []string{"webapp"},
				OIDCEnabled:   true,
				OIDCClients: []schema.OIDCClient{
					{ID: "client1", Secret: "secret1", RedirectURI: "https://example.com/callback"},
				},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:frontend:create", "myfe") {
		t.Errorf("expected auth:frontend:create, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:frontend:provider:set", "myfe", "oauth2") {
		t.Errorf("expected auth:frontend:provider:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:frontend:use-directory", "myfe", "mydir") {
		t.Errorf("expected auth:frontend:use-directory, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:frontend:protect", "myfe", "webapp") {
		t.Errorf("expected auth:frontend:protect, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:oidc:enable", "myfe") {
		t.Errorf("expected auth:oidc:enable, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:oidc:add-client", "myfe", "client1", "secret1", "https://example.com/callback") {
		t.Errorf("expected auth:oidc:add-client, got: %v", runner.commandStrings())
	}
}

func TestDestroyAuthFrontend(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.DestroyAuthFrontend, Service: "myfe"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:frontend:destroy", "myfe", "--force") {
		t.Errorf("expected auth:frontend:destroy, got: %v", runner.commandStrings())
	}
}

func TestCreateAppWithGitConfig(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Git: &schema.GitConfig{
					Repo:   "https://github.com/example/repo.git",
					Branch: "main",
				},
				Domains: []string{"example.com"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("apps:create", "myapp") {
		t.Errorf("expected apps:create, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("git:set", "myapp", "deploy-branch", "main") {
		t.Errorf("expected git:set deploy-branch, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("git:sync", "--build", "myapp", "https://github.com/example/repo.git", "main") {
		t.Errorf("expected git:sync, got: %v", runner.commandStrings())
	}
}

func TestUpdateResources(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "resources"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Resources: map[string]schema.ResourceConfig{
				"web": {
					Limits:       schema.ResourceValues{CPU: "2", Memory: "1024m"},
					Reservations: schema.ResourceValues{Memory: "512m"},
				},
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("resource:limit", "myapp", "--process-type", "web", "--cpu", "2") {
		t.Errorf("expected resource:limit cpu, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("resource:limit", "myapp", "--process-type", "web", "--memory", "1024m") {
		t.Errorf("expected resource:limit memory, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("resource:reserve", "myapp", "--process-type", "web", "--memory", "512m") {
		t.Errorf("expected resource:reserve memory, got: %v", runner.commandStrings())
	}
}

func TestUpdateChecks(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "checks"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Checks: &schema.ChecksConfig{
				Disabled:     []string{"worker"},
				WaitToRetire: 30,
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("checks:disable", "myapp", "worker") {
		t.Errorf("expected checks:disable, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("checks:set", "myapp", "wait-to-retire", "30") {
		t.Errorf("expected checks:set wait-to-retire, got: %v", runner.commandStrings())
	}
}

func TestUpdateBuilder(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "builder"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Builder: &schema.BuilderConfig{
				Selected: "herokuish",
				BuildDir: "src",
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("builder:set", "myapp", "selected", "herokuish") {
		t.Errorf("expected builder:set selected, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("builder:set", "myapp", "build-dir", "src") {
		t.Errorf("expected builder:set build-dir, got: %v", runner.commandStrings())
	}
}

func TestUpdateRegistry(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "registry"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Registry: &schema.RegistryConfig{
				Server:        "registry.example.com",
				ImageRepo:     "myorg/myapp",
				PushOnRelease: true,
				PushExtraTags: "latest",
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("registry:set", "myapp", "server", "registry.example.com") {
		t.Errorf("expected registry:set server, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("registry:set", "myapp", "image-repo", "myorg/myapp") {
		t.Errorf("expected registry:set image-repo, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("registry:set", "myapp", "push-on-release", "true") {
		t.Errorf("expected registry:set push-on-release, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("registry:set", "myapp", "push-extra-tags", "latest") {
		t.Errorf("expected registry:set push-extra-tags, got: %v", runner.commandStrings())
	}
}

func TestUpdateMaintenance(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	// Enable
	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "maintenance"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Maintenance: true}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("maintenance:enable", "myapp") {
		t.Errorf("expected maintenance:enable, got: %v", runner.commandStrings())
	}

	// Disable
	runner2 := &RecordingRunner{}
	executor2 := &Executor{Runner: runner2}
	desired2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Maintenance: false}},
	}
	actual2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Maintenance: true}},
	}

	err = executor2.Execute(p, desired2, actual2)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner2.hasCommand("maintenance:disable", "myapp") {
		t.Errorf("expected maintenance:disable, got: %v", runner2.commandStrings())
	}
}

func TestCreateAppWithNewFeatures(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx:latest",
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
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	cmds := runner.commandStrings()
	if !runner.hasCommand("apps:create", "myapp") {
		t.Errorf("expected apps:create, got: %v", cmds)
	}
	if !runner.hasCommand("resource:limit", "myapp", "--process-type", "web", "--cpu", "1") {
		t.Errorf("expected resource:limit, got: %v", cmds)
	}
	if !runner.hasCommand("checks:disable", "myapp", "worker") {
		t.Errorf("expected checks:disable, got: %v", cmds)
	}
	if !runner.hasCommand("builder:set", "myapp", "selected", "herokuish") {
		t.Errorf("expected builder:set, got: %v", cmds)
	}
	if !runner.hasCommand("registry:set", "myapp", "server", "registry.example.com") {
		t.Errorf("expected registry:set, got: %v", cmds)
	}
	if !runner.hasCommand("maintenance:enable", "myapp") {
		t.Errorf("expected maintenance:enable, got: %v", cmds)
	}
}

func TestUpdateNginxProperties(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Nginx: &schema.NginxConfig{
				HSTS: true,
				Properties: map[string]string{
					"client-max-body-size": "50m",
					"proxy-read-timeout":   "120s",
				},
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("nginx:set", "myapp", "hsts", "true") {
		t.Errorf("expected nginx:set hsts, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("nginx:set", "myapp", "client-max-body-size", "50m") {
		t.Errorf("expected nginx:set client-max-body-size, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("nginx:set", "myapp", "proxy-read-timeout", "120s") {
		t.Errorf("expected nginx:set proxy-read-timeout, got: %v", runner.commandStrings())
	}
}

func TestUpdateScripts(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "scripts"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Scripts: &schema.ScriptsConfig{
				Predeploy:  "rake db:migrate",
				Postdeploy: "rake cache:clear",
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Should generate an app-json:set command containing scripts
	found := false
	for _, cmd := range runner.Commands {
		if len(cmd) >= 2 && cmd[0] == "app-json:set" && cmd[1] == "myapp" {
			found = true
			if len(cmd) >= 3 {
				if !strings.Contains(cmd[2], "predeploy") || !strings.Contains(cmd[2], "postdeploy") {
					t.Errorf("app-json:set should contain scripts, got: %s", cmd[2])
				}
			}
		}
	}
	if !found {
		t.Errorf("expected app-json:set command, got: %v", runner.commandStrings())
	}
}

func TestUpdateLocked(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "locked"},
		},
	}

	// Lock
	desired := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Locked: true}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("apps:lock", "myapp") {
		t.Errorf("expected apps:lock, got: %v", runner.commandStrings())
	}

	// Unlock
	runner2 := &RecordingRunner{}
	executor2 := &Executor{Runner: runner2}
	desired2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Locked: false}},
	}
	actual2 := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Locked: true}},
	}

	err = executor2.Execute(p, desired2, actual2)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner2.hasCommand("apps:unlock", "myapp") {
		t.Errorf("expected apps:unlock, got: %v", runner2.commandStrings())
	}
}

func TestUpdateProcess(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "process"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {
			Process: &schema.ProcessConfig{
				RestartPolicy: "on-failure:3",
				ProcfilePath:  "Procfile.web",
			},
		}},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("ps:set", "myapp", "restart-policy", "on-failure:3") {
		t.Errorf("expected ps:set restart-policy, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("ps:set", "myapp", "procfile-path", "Procfile.web") {
		t.Errorf("expected ps:set procfile-path, got: %v", runner.commandStrings())
	}
}

// FailingRunner fails on a specific command prefix.
type FailingRunner struct {
	failOn string
}

func (r *FailingRunner) Run(args ...string) (string, error) {
	if len(args) > 0 && args[0] == r.failOn {
		return "command failed", fmt.Errorf("exit status 1")
	}
	return "", nil
}
