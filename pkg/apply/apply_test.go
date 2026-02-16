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
