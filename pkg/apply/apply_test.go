package apply

import (
	"fmt"
	"io"
	"os"
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
		{"postgres:link", "mydb", "myapp", "--no-restart"},
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
	fileRunner := &FakeFileRunner{Files: map[string]string{}}
	executor := &Executor{Runner: runner, FileRunner: fileRunner}

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

	// Should write app.json file containing healthchecks
	content, ok := fileRunner.Files["/home/dokku/myapp/app.json"]
	if !ok {
		t.Errorf("expected app.json to be written, got files: %v", fileRunner.Files)
	}
	if !strings.Contains(content, "healthchecks") {
		t.Errorf("app.json should contain healthchecks, got: %s", content)
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
	fileRunner := &FakeFileRunner{Files: map[string]string{}}
	executor := &Executor{Runner: runner, FileRunner: fileRunner}

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

	// Should write app.json file containing scripts
	content, ok := fileRunner.Files["/home/dokku/myapp/app.json"]
	if !ok {
		t.Errorf("expected app.json to be written, got files: %v", fileRunner.Files)
	}
	if !strings.Contains(content, "predeploy") || !strings.Contains(content, "postdeploy") {
		t.Errorf("app.json should contain scripts, got: %s", content)
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

func TestUpdateLogs(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "logs"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Logs: &schema.LogConfig{
					MaxSize:       "50m",
					VectorSink:    "console://",
					VectorImage:   "timberio/vector:latest",
					AppLabelAlias: "myapp-alias",
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("logs:set", "myapp", "max-size", "50m") {
		t.Errorf("expected logs:set max-size, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("logs:set", "myapp", "vector-sink", "console://") {
		t.Errorf("expected logs:set vector-sink, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("logs:set", "myapp", "vector-image", "timberio/vector:latest") {
		t.Errorf("expected logs:set vector-image, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("logs:set", "myapp", "app-label-alias", "myapp-alias") {
		t.Errorf("expected logs:set app-label-alias, got: %v", runner.commandStrings())
	}
}

func TestUpdateScheduler(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "scheduler"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Scheduler: &schema.SchedulerConfig{
					Selected:                         "docker-local",
					DockerLocalInitProcess:            "false",
					DockerLocalParallelScheduleCount: "2",
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("scheduler:set", "myapp", "selected", "docker-local") {
		t.Errorf("expected scheduler:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("scheduler-docker-local:set", "myapp", "init-process", "false") {
		t.Errorf("expected scheduler-docker-local:set init-process, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("scheduler-docker-local:set", "myapp", "parallel-schedule-count", "2") {
		t.Errorf("expected scheduler-docker-local:set parallel-schedule-count, got: %v", runner.commandStrings())
	}
}

func TestUpdateBuildpacks(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "buildpacks"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Buildpacks: []string{
					"https://github.com/heroku/heroku-buildpack-nodejs",
					"https://github.com/heroku/heroku-buildpack-ruby",
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("buildpacks:clear", "myapp") {
		t.Errorf("expected buildpacks:clear, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("buildpacks:add", "myapp", "https://github.com/heroku/heroku-buildpack-nodejs") {
		t.Errorf("expected buildpacks:add nodejs, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("buildpacks:add", "myapp", "https://github.com/heroku/heroku-buildpack-ruby") {
		t.Errorf("expected buildpacks:add ruby, got: %v", runner.commandStrings())
	}
}

func TestUpdateBuilderSubPlugins(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "builder"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Builder: &schema.BuilderConfig{
					Selected:            "dockerfile",
					DockerfilePath:      "docker/Dockerfile.prod",
					PackProjecttomlPath: "custom/project.toml",
					NixpacksTomlPath:    "custom/nixpacks.toml",
					HerokuishAllowed:    "true",
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("builder:set", "myapp", "selected", "dockerfile") {
		t.Errorf("expected builder:set selected, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("builder-dockerfile:set", "myapp", "dockerfile-path", "docker/Dockerfile.prod") {
		t.Errorf("expected builder-dockerfile:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("builder-pack:set", "myapp", "projecttoml-path", "custom/project.toml") {
		t.Errorf("expected builder-pack:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("builder-nixpacks:set", "myapp", "nixpackstoml-path", "custom/nixpacks.toml") {
		t.Errorf("expected builder-nixpacks:set, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("builder-herokuish:set", "myapp", "allowed", "true") {
		t.Errorf("expected builder-herokuish:set, got: %v", runner.commandStrings())
	}
}

func TestInstallPlugin(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.InstallPlugin, Service: "letsencrypt"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Plugins: map[string]schema.Plugin{
			"letsencrypt": {
				URL:        "https://github.com/dokku/dokku-letsencrypt.git",
				Committish: "v1.0.0",
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("plugin:install", "https://github.com/dokku/dokku-letsencrypt.git", "--name", "letsencrypt", "--committish", "v1.0.0") {
		t.Errorf("expected plugin:install, got: %v", runner.commandStrings())
	}
}

func TestUninstallPlugin(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UninstallPlugin, Service: "letsencrypt"},
		},
	}
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("plugin:uninstall", "letsencrypt") {
		t.Errorf("expected plugin:uninstall, got: %v", runner.commandStrings())
	}
}

func TestUpdateProxyWithCaddy(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "proxy"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Proxy: &schema.ProxyConfig{
					Enabled: true,
					Type:    "caddy",
					Caddy: map[string]string{
						"tls-internal":     "true",
						"letsencrypt-email": "admin@example.com",
					},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("proxy:enable", "myapp") {
		t.Errorf("expected proxy:enable, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("proxy:set", "myapp", "caddy") {
		t.Errorf("expected proxy:set caddy, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("caddy:set", "myapp", "tls-internal", "true") {
		t.Errorf("expected caddy:set tls-internal, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("caddy:set", "myapp", "letsencrypt-email", "admin@example.com") {
		t.Errorf("expected caddy:set letsencrypt-email, got: %v", runner.commandStrings())
	}
}

func TestUpdateProxyWithTraefik(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "proxy"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Proxy: &schema.ProxyConfig{
					Enabled: true,
					Type:    "traefik",
					Traefik: map[string]string{
						"api-enabled": "true",
						"log-level":   "DEBUG",
					},
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("proxy:set", "myapp", "traefik") {
		t.Errorf("expected proxy:set traefik, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("traefik:set", "myapp", "api-enabled", "true") {
		t.Errorf("expected traefik:set api-enabled, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("traefik:set", "myapp", "log-level", "DEBUG") {
		t.Errorf("expected traefik:set log-level, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalDomains(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "domains"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Domains: []string{"example.com", "example.org"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("domains:set", "--global", "example.com", "example.org") {
		t.Errorf("expected domains:set --global, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalNginx(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Nginx: &schema.NginxConfig{
				HSTS:       true,
				HSTSMaxAge: 31536000,
				Properties: map[string]string{"client-max-body-size": "50m"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("nginx:set", "--global", "hsts", "true") {
		t.Errorf("expected nginx:set --global hsts true, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("nginx:set", "--global", "hsts-max-age", "31536000") {
		t.Errorf("expected nginx:set --global hsts-max-age, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("nginx:set", "--global", "client-max-body-size", "50m") {
		t.Errorf("expected nginx:set --global client-max-body-size, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalLogs(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "logs"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Logs: &schema.LogConfig{MaxSize: "10m", VectorSink: "console://"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("logs:set", "--global", "max-size", "10m") {
		t.Errorf("expected logs:set --global max-size, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("logs:set", "--global", "vector-sink", "console://") {
		t.Errorf("expected logs:set --global vector-sink, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalNetwork(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "network"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Network: &schema.NetworkConfig{InitialNetwork: "bridge", TLD: "dokku.me"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("network:set", "--global", "initial-network", "bridge") {
		t.Errorf("expected network:set --global initial-network, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("network:set", "--global", "tld", "dokku.me") {
		t.Errorf("expected network:set --global tld, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalBuilder(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "builder"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Builder: &schema.BuilderConfig{Selected: "pack"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("builder:set", "--global", "selected", "pack") {
		t.Errorf("expected builder:set --global selected, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalScheduler(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "scheduler"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Scheduler: &schema.SchedulerConfig{Selected: "docker-local"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("scheduler:set", "--global", "selected", "docker-local") {
		t.Errorf("expected scheduler:set --global selected, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalRegistry(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "registry"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Registry: &schema.RegistryConfig{Server: "docker.io", PushOnRelease: true},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("registry:set", "--global", "server", "docker.io") {
		t.Errorf("expected registry:set --global server, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("registry:set", "--global", "push-on-release", "true") {
		t.Errorf("expected registry:set --global push-on-release, got: %v", runner.commandStrings())
	}
}

func TestSecretsApply(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{
		Runner: runner,
		EnvGetter: func(key string) string {
			switch key {
			case "DATABASE_URL":
				return "postgres://localhost/mydb"
			case "API_KEY":
				return "secret123"
			default:
				return ""
			}
		},
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "secrets"},
		},
	}
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
		Apps:    map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Should have a config:set with both secrets
	found := false
	for _, cmd := range runner.Commands {
		if len(cmd) >= 4 && cmd[0] == "config:set" && cmd[1] == "--no-restart" && cmd[2] == "myapp" {
			found = true
			cmdStr := strings.Join(cmd, " ")
			if !strings.Contains(cmdStr, "DATABASE_URL=postgres://localhost/mydb") {
				t.Errorf("expected DATABASE_URL in config:set, got: %s", cmdStr)
			}
			if !strings.Contains(cmdStr, "API_KEY=secret123") {
				t.Errorf("expected API_KEY in config:set, got: %s", cmdStr)
			}
		}
	}
	if !found {
		t.Errorf("expected config:set command for secrets, got: %v", runner.commandStrings())
	}
}

func TestCreateAppWithSecrets(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{
		Runner: runner,
		EnvGetter: func(key string) string {
			if key == "SECRET_KEY" {
				return "mysecretvalue"
			}
			return ""
		},
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx",
				Env:     map[string]string{"APP_NAME": "myapp"},
				Secrets: []string{"SECRET_KEY"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Should have two config:set commands: one for env, one for secrets
	configSetCount := 0
	hasSecretSet := false
	for _, cmd := range runner.Commands {
		if len(cmd) >= 3 && cmd[0] == "config:set" {
			configSetCount++
			cmdStr := strings.Join(cmd, " ")
			if strings.Contains(cmdStr, "SECRET_KEY=mysecretvalue") {
				hasSecretSet = true
			}
		}
	}
	if configSetCount < 2 {
		t.Errorf("expected at least 2 config:set commands (env + secrets), got %d: %v", configSetCount, runner.commandStrings())
	}
	if !hasSecretSet {
		t.Errorf("expected SECRET_KEY=mysecretvalue in config:set, got: %v", runner.commandStrings())
	}
}

func TestUpdateGlobalDomainsClear(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateGlobal, Field: "domains"},
		},
	}
	// Desired has nil Global (or empty domains) - should clear
	desired := &schema.Dokkufile{Version: "1"}
	actual := &schema.Dokkufile{
		Version: "1",
		Global: &schema.GlobalConfig{
			Domains: []string{"old.example.com"},
		},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("domains:clear", "--global") {
		t.Errorf("expected domains:clear --global, got: %v", runner.commandStrings())
	}
}

func TestMailLinkUpdate(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "mail"},
		},
	}
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

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("mail:unlink", "oldmail", "myapp") {
		t.Errorf("expected mail:unlink oldmail, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("mail:link", "newmail", "myapp") {
		t.Errorf("expected mail:link newmail, got: %v", runner.commandStrings())
	}
}

func TestAuthLinkUpdate(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "auth"},
		},
	}
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

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:unlink", "olddir", "myapp") {
		t.Errorf("expected auth:unlink olddir, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:link", "newdir", "myapp") {
		t.Errorf("expected auth:link newdir, got: %v", runner.commandStrings())
	}
}

func TestCreateAppWithMailAndAuth(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Mail:  "mymail",
				Auth:  &schema.AuthConfig{Directory: "mydir"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("mail:link", "mymail", "myapp") {
		t.Errorf("expected mail:link mymail, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:link", "mydir", "myapp") {
		t.Errorf("expected auth:link mydir, got: %v", runner.commandStrings())
	}
}

// FakeFileRunner for SSL cert tests.
type FakeFileRunner struct {
	Files map[string]string
}

func (f *FakeFileRunner) ReadFile(path string) (string, error) {
	if content, ok := f.Files[path]; ok {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s", path)
}

func (f *FakeFileRunner) WriteFile(path string, content []byte, perm os.FileMode) error {
	f.Files[path] = string(content)
	return nil
}

// RecordingStdinRunner records commands including stdin calls.
type RecordingStdinRunner struct {
	RecordingRunner
	StdinCalls []struct {
		Args []string
		Data []byte
	}
}

func (r *RecordingStdinRunner) RunWithStdin(stdin io.Reader, args ...string) (string, error) {
	data, _ := io.ReadAll(stdin)
	r.StdinCalls = append(r.StdinCalls, struct {
		Args []string
		Data []byte
	}{Args: args, Data: data})
	return "", nil
}

func TestSSLCertUploadViaTar(t *testing.T) {
	runner := &RecordingStdinRunner{}
	fileRunner := &FakeFileRunner{
		Files: map[string]string{
			"/certs/server.crt": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n",
			"/certs/server.key": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----\n",
		},
	}
	executor := &Executor{Runner: runner, FileRunner: fileRunner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "ssl"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				SSL: &schema.SSLConfig{
					CertFile: "/certs/server.crt",
					KeyFile:  "/certs/server.key",
				},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{"myapp": {Image: "nginx"}},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(runner.StdinCalls) != 1 {
		t.Fatalf("expected 1 stdin call, got %d", len(runner.StdinCalls))
	}
	call := runner.StdinCalls[0]
	if len(call.Args) != 2 || call.Args[0] != "certs:add" || call.Args[1] != "myapp" {
		t.Errorf("expected certs:add myapp, got: %v", call.Args)
	}
	if len(call.Data) == 0 {
		t.Error("expected non-empty tar data in stdin")
	}
}

func TestAuthProtectedUpdate(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateApp, App: "myapp", Field: "auth"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Protected: "newfe"},
			},
		},
	}
	actual := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Protected: "oldfe"},
			},
		},
	}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:frontend:unprotect", "oldfe", "myapp") {
		t.Errorf("expected auth:frontend:unprotect oldfe, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:frontend:protect", "newfe", "myapp") {
		t.Errorf("expected auth:frontend:protect newfe, got: %v", runner.commandStrings())
	}
}

func TestCreateAppWithAuthProtected(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp", NewValue: "nginx"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image: "nginx",
				Auth:  &schema.AuthConfig{Directory: "mydir", Protected: "myfe"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("auth:link", "mydir", "myapp") {
		t.Errorf("expected auth:link mydir, got: %v", runner.commandStrings())
	}
	if !runner.hasCommand("auth:frontend:protect", "myfe", "myapp") {
		t.Errorf("expected auth:frontend:protect myfe, got: %v", runner.commandStrings())
	}
}

func TestCreateServiceWithImageVersion(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres", ImageVersion: "15"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("postgres:create", "mydb", "--image-version", "15") {
		t.Errorf("expected postgres:create mydb --image-version 15, got: %v", runner.commandStrings())
	}
}

func TestCreateServiceWithoutImageVersion(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Services: map[string]schema.Service{
			"mydb": {Type: "postgres"},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("postgres:create", "mydb") {
		t.Errorf("expected postgres:create mydb (no --image-version), got: %v", runner.commandStrings())
	}
}

func TestUpdateServiceVersion(t *testing.T) {
	runner := &RecordingRunner{}
	executor := &Executor{Runner: runner}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.UpdateService, Service: "mydb", ServiceType: "postgres", Field: "image_version"},
		},
	}
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

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if !runner.hasCommand("postgres:upgrade", "mydb", "--image-version", "16") {
		t.Errorf("expected postgres:upgrade mydb --image-version 16, got: %v", runner.commandStrings())
	}
}
