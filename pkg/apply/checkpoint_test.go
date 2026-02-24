package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deanmarano/dokkufile/pkg/plan"
	"github.com/deanmarano/dokkufile/pkg/schema"
)

func TestStepKey(t *testing.T) {
	tests := []struct {
		name string
		step plan.Step
		want string
	}{
		{
			name: "create app",
			step: plan.Step{Action: plan.CreateApp, App: "myapp"},
			want: "create_app:myapp",
		},
		{
			name: "update app field",
			step: plan.Step{Action: plan.UpdateApp, App: "myapp", Field: "image"},
			want: "update_app:myapp:image",
		},
		{
			name: "create service with type",
			step: plan.Step{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
			want: "create_service:postgres:mydb",
		},
		{
			name: "install plugin",
			step: plan.Step{Action: plan.InstallPlugin, Service: "letsencrypt"},
			want: "install_plugin:letsencrypt",
		},
		{
			name: "update global field",
			step: plan.Step{Action: plan.UpdateGlobal, Field: "domains"},
			want: "update_global:domains",
		},
		{
			name: "create mail service",
			step: plan.Step{Action: plan.CreateMailService, Service: "mail1"},
			want: "create_mail_service:mail1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepKey(tt.step)
			if got != tt.want {
				t.Errorf("StepKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	dir := t.TempDir()

	cp := &Checkpoint{
		DokkufileHash: "abc123",
		CompletedSteps: []CompletedStep{
			{StepKey: "create_service:postgres:mydb", Commands: []string{"postgres:create mydb"}},
			{StepKey: "create_app:myapp", Commands: []string{"apps:create myapp", "domains:set myapp example.com"}},
		},
		FailedStep: &StepRecord{StepKey: "update_app:myapp:image", Error: "exit status 1"},
		CreatedAt:  "2026-01-01T00:00:00Z",
	}

	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	loaded, err := loadCheckpoint(dir)
	if err != nil {
		t.Fatalf("loadCheckpoint: %v", err)
	}

	if loaded.DokkufileHash != cp.DokkufileHash {
		t.Errorf("hash = %q, want %q", loaded.DokkufileHash, cp.DokkufileHash)
	}
	if len(loaded.CompletedSteps) != 2 {
		t.Fatalf("completed steps = %d, want 2", len(loaded.CompletedSteps))
	}
	if loaded.CompletedSteps[0].StepKey != "create_service:postgres:mydb" {
		t.Errorf("step 0 key = %q", loaded.CompletedSteps[0].StepKey)
	}
	if loaded.FailedStep == nil || loaded.FailedStep.StepKey != "update_app:myapp:image" {
		t.Errorf("failed step = %v", loaded.FailedStep)
	}
}

func TestCheckpointLoadNonexistent(t *testing.T) {
	dir := t.TempDir()

	cp, err := loadCheckpoint(dir)
	if err != nil {
		t.Fatalf("loadCheckpoint: %v", err)
	}
	if cp != nil {
		t.Errorf("expected nil checkpoint, got %+v", cp)
	}
}

func TestClearCheckpoint(t *testing.T) {
	dir := t.TempDir()

	cp := &Checkpoint{DokkufileHash: "abc", CreatedAt: "2026-01-01T00:00:00Z"}
	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	if err := clearCheckpoint(dir); err != nil {
		t.Fatalf("clearCheckpoint: %v", err)
	}

	loaded, err := loadCheckpoint(dir)
	if err != nil {
		t.Fatalf("loadCheckpoint: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil after clear, got %+v", loaded)
	}
}

func TestClearCheckpointNonexistent(t *testing.T) {
	dir := t.TempDir()
	if err := clearCheckpoint(dir); err != nil {
		t.Errorf("clearCheckpoint on nonexistent should not error, got: %v", err)
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yml")

	content := []byte("version: '1'\napps:\n  myapp:\n    image: nginx\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	h1, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if h1 == "" {
		t.Error("expected non-empty hash")
	}

	// Same content should produce same hash
	h2, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if h1 != h2 {
		t.Errorf("same file produced different hashes: %q vs %q", h1, h2)
	}

	// Different content should produce different hash
	if err := os.WriteFile(path, []byte("different"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h3, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	if h1 == h3 {
		t.Error("different files produced same hash")
	}
}

// checkpointTestRunner records commands and optionally fails on a specific command prefix.
// It also supports returning "already exists" errors for create commands.
type checkpointTestRunner struct {
	commands       [][]string
	failOn         string            // command prefix that should fail
	failOutput     string            // error output for the failing command
	alreadyExists  map[string]bool   // command prefixes that return "already exists" errors
}

func (r *checkpointTestRunner) Run(args ...string) (string, error) {
	cmd := make([]string, len(args))
	copy(cmd, args)
	r.commands = append(r.commands, cmd)
	cmdStr := strings.Join(cmd, " ")
	if r.failOn != "" && strings.HasPrefix(cmdStr, r.failOn) {
		return r.failOutput, fmt.Errorf("exit status 1")
	}
	for prefix := range r.alreadyExists {
		if strings.HasPrefix(cmdStr, prefix) {
			return fmt.Sprintf("%s already exists", args[len(args)-1]), fmt.Errorf("exit status 1")
		}
	}
	return "", nil
}

func (r *checkpointTestRunner) commandStrings() []string {
	var strs []string
	for _, cmd := range r.commands {
		strs = append(strs, strings.Join(cmd, " "))
	}
	return strs
}

func TestExecuteResumeFromCheckpoint(t *testing.T) {
	dir := t.TempDir()

	// Write a dokkufile so we can hash it
	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	dokkufileContent := []byte("version: '1'\n")
	if err := os.WriteFile(dokkufilePath, dokkufileContent, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	dokkufileHash, _ := hashFile(dokkufilePath)

	// Simulate a previous run that completed 2 steps
	cp := &Checkpoint{
		DokkufileHash: dokkufileHash,
		CompletedSteps: []CompletedStep{
			{StepKey: "create_service:postgres:mydb", Commands: []string{"postgres:create mydb"}},
			{StepKey: "create_app:myapp", Commands: []string{"apps:create myapp"}},
		},
		FailedStep: &StepRecord{StepKey: "update_app:myapp:image", Error: "exit status 1"},
		CreatedAt:  "2026-01-01T00:00:00Z",
	}
	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	// Set up a plan with 3 steps (the same ones)
	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
			{Action: plan.CreateApp, App: "myapp"},
			{Action: plan.UpdateApp, App: "myapp", Field: "image"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
		Apps:     map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
		Apps:     map[string]schema.App{"myapp": {Image: "nginx:old"}},
	}

	runner := &checkpointTestRunner{} // no failures this time
	executor := &Executor{
		Runner:        runner,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	// Suppress output during test
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Only the third step should have run (first two were checkpointed)
	if len(runner.commands) != 1 {
		t.Errorf("expected 1 command, got %d: %v", len(runner.commands), runner.commandStrings())
	}
	if len(runner.commands) > 0 {
		cmdStr := strings.Join(runner.commands[0], " ")
		if cmdStr != "git:from-image myapp nginx:latest" {
			t.Errorf("expected git:from-image command, got: %s", cmdStr)
		}
	}

	// Checkpoint should be cleared on success
	loadedCP, _ := loadCheckpoint(dir)
	if loadedCP != nil {
		t.Error("expected checkpoint to be cleared after successful apply")
	}
}

func TestExecuteSavesCheckpointOnFailure(t *testing.T) {
	dir := t.TempDir()

	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	if err := os.WriteFile(dokkufilePath, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// CreateApp generates git:from-image internally as a sub-command,
	// so the failure happens within the CreateApp step. Only the
	// CreateService step completes successfully.
	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
			{Action: plan.CreateApp, App: "myapp"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
		Apps:     map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	runner := &checkpointTestRunner{failOn: "git:from-image", failOutput: "deploy failed"}
	executor := &Executor{
		Runner:        runner,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor.Execute(p, desired, actual)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Checkpoint should be saved with 1 completed step (CreateService)
	// CreateApp failed partway through, so it is NOT marked complete
	loadedCP, loadErr := loadCheckpoint(dir)
	if loadErr != nil {
		t.Fatalf("loadCheckpoint: %v", loadErr)
	}
	if loadedCP == nil {
		t.Fatal("expected checkpoint to be saved")
	}
	if len(loadedCP.CompletedSteps) != 1 {
		t.Errorf("expected 1 completed step, got %d", len(loadedCP.CompletedSteps))
	}
	if loadedCP.CompletedSteps[0].StepKey != "create_service:postgres:mydb" {
		t.Errorf("completed step key = %q, want create_service:postgres:mydb", loadedCP.CompletedSteps[0].StepKey)
	}
	if loadedCP.FailedStep == nil {
		t.Fatal("expected failed step to be recorded")
	}
	if loadedCP.FailedStep.StepKey != "create_app:myapp" {
		t.Errorf("failed step key = %q, want create_app:myapp", loadedCP.FailedStep.StepKey)
	}
}

func TestExecuteStaleCheckpointIgnored(t *testing.T) {
	dir := t.TempDir()

	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	if err := os.WriteFile(dokkufilePath, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Save checkpoint with a different hash
	cp := &Checkpoint{
		DokkufileHash: "stale-hash-from-old-dokkufile",
		CompletedSteps: []CompletedStep{
			{StepKey: "create_service:postgres:mydb", Commands: []string{"postgres:create mydb"}},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	runner := &checkpointTestRunner{}
	executor := &Executor{
		Runner:        runner,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Despite the checkpoint saying step was done, it should have re-run
	// because the hash didn't match
	if len(runner.commands) != 1 {
		t.Errorf("expected 1 command (stale checkpoint should be ignored), got %d: %v",
			len(runner.commands), runner.commandStrings())
	}

	// Stale checkpoint file should have been deleted
	loadedCP, _ := loadCheckpoint(dir)
	if loadedCP != nil {
		t.Error("stale checkpoint should have been deleted")
	}
}

func TestExecuteNoCheckpointInDryRun(t *testing.T) {
	dir := t.TempDir()

	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	if err := os.WriteFile(dokkufilePath, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	runner := &checkpointTestRunner{failOn: "postgres:create", failOutput: "fail"}
	executor := &Executor{
		Runner:        runner,
		DryRun:        true,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	// Dry run should not fail (commands are not actually run) and should not save checkpoint
	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	loadedCP, _ := loadCheckpoint(dir)
	if loadedCP != nil {
		t.Error("checkpoint should not be saved in dry-run mode")
	}
}

func TestPartialCreateAppResume(t *testing.T) {
	dir := t.TempDir()
	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	if err := os.WriteFile(dokkufilePath, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateApp, App: "myapp"},
		},
	}
	desired := &schema.Dokkufile{
		Version: "1",
		Apps: map[string]schema.App{
			"myapp": {
				Image:   "nginx:latest",
				Domains: []string{"myapp.example.com"},
				Env:     map[string]string{"NODE_ENV": "production"},
			},
		},
	}
	actual := &schema.Dokkufile{Version: "1"}

	// First run: apps:create succeeds, git:from-image fails
	runner1 := &checkpointTestRunner{failOn: "git:from-image", failOutput: "deploy failed"}
	executor1 := &Executor{
		Runner:        runner1,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor1.Execute(p, desired, actual)
	if err == nil {
		t.Fatal("expected error on first run")
	}

	// Verify apps:create ran but step is NOT marked complete (it failed partway)
	if runner1.commands[0][0] != "apps:create" {
		t.Errorf("expected apps:create as first command, got: %s", runner1.commands[0][0])
	}
	cp, _ := loadCheckpoint(dir)
	if cp == nil {
		t.Fatal("expected checkpoint after failure")
	}
	if len(cp.CompletedSteps) != 0 {
		t.Errorf("CreateApp should NOT be marked complete, got %d completed steps", len(cp.CompletedSteps))
	}

	// Second run: apps:create returns "already exists" (tolerated), deploy succeeds
	runner2 := &checkpointTestRunner{
		alreadyExists: map[string]bool{"apps:create": true},
	}
	executor2 := &Executor{
		Runner:        runner2,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	err = executor2.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("expected success on resume, got: %v", err)
	}

	// Verify apps:create was attempted (and tolerated), then the rest ran
	cmds := runner2.commandStrings()
	foundCreate := false
	foundDeploy := false
	for _, c := range cmds {
		if strings.HasPrefix(c, "apps:create") {
			foundCreate = true
		}
		if strings.HasPrefix(c, "git:from-image") {
			foundDeploy = true
		}
	}
	if !foundCreate {
		t.Errorf("expected apps:create to be re-attempted, got: %v", cmds)
	}
	if !foundDeploy {
		t.Errorf("expected git:from-image to run on resume, got: %v", cmds)
	}

	// Checkpoint should be cleared
	cpAfter, _ := loadCheckpoint(dir)
	if cpAfter != nil {
		t.Error("checkpoint should be cleared after successful resume")
	}
}

func TestResumedApplyWithNewFailure(t *testing.T) {
	dir := t.TempDir()
	dokkufilePath := filepath.Join(dir, "Dokkufile.yml")
	if err := os.WriteFile(dokkufilePath, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	dfHash, _ := hashFile(dokkufilePath)

	// Simulate a previous run where step 1 completed
	cp := &Checkpoint{
		DokkufileHash: dfHash,
		CompletedSteps: []CompletedStep{
			{StepKey: "create_service:postgres:mydb", Commands: []string{"postgres:create mydb"}},
		},
		FailedStep: &StepRecord{StepKey: "create_service:redis:mycache", Error: "exit status 1"},
		CreatedAt:  "2026-01-01T00:00:00Z",
	}
	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	// Plan has 3 steps: step 1 already done, step 2 should succeed now, step 3 fails
	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
			{Action: plan.CreateService, Service: "mycache", ServiceType: "redis"},
			{Action: plan.CreateApp, App: "myapp"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{
			"mydb":    {Type: "postgres"},
			"mycache": {Type: "redis"},
		},
		Apps: map[string]schema.App{"myapp": {Image: "nginx:latest"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	// Step 2 (redis:create) succeeds, step 3 (CreateApp -> git:from-image) fails
	runner := &checkpointTestRunner{failOn: "git:from-image", failOutput: "deploy failed"}
	executor := &Executor{
		Runner:        runner,
		CheckpointDir: dir,
		DokkufilePath: dokkufilePath,
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor.Execute(p, desired, actual)
	if err == nil {
		t.Fatal("expected error")
	}

	// New checkpoint should have BOTH step 1 (from old checkpoint) AND step 2 (newly completed)
	newCP, loadErr := loadCheckpoint(dir)
	if loadErr != nil {
		t.Fatalf("loadCheckpoint: %v", loadErr)
	}
	if newCP == nil {
		t.Fatal("expected new checkpoint")
	}
	if len(newCP.CompletedSteps) != 2 {
		t.Errorf("expected 2 completed steps (old + new), got %d", len(newCP.CompletedSteps))
	}

	// Verify the completed steps are correct
	keys := map[string]bool{}
	for _, cs := range newCP.CompletedSteps {
		keys[cs.StepKey] = true
	}
	if !keys["create_service:postgres:mydb"] {
		t.Error("missing original completed step create_service:postgres:mydb")
	}
	if !keys["create_service:redis:mycache"] {
		t.Error("missing newly completed step create_service:redis:mycache")
	}

	// Failed step should be the CreateApp
	if newCP.FailedStep == nil || newCP.FailedStep.StepKey != "create_app:myapp" {
		t.Errorf("expected failed step create_app:myapp, got: %v", newCP.FailedStep)
	}
}

func TestCheckpointDisabledWithoutDokkufilePath(t *testing.T) {
	dir := t.TempDir()

	// Save a checkpoint with empty hash (simulating a previous run without DokkufilePath)
	cp := &Checkpoint{
		DokkufileHash: "",
		CompletedSteps: []CompletedStep{
			{StepKey: "create_service:postgres:mydb", Commands: []string{"postgres:create mydb"}},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if err := saveCheckpoint(dir, cp); err != nil {
		t.Fatalf("saveCheckpoint: %v", err)
	}

	p := &plan.Plan{
		Steps: []plan.Step{
			{Action: plan.CreateService, Service: "mydb", ServiceType: "postgres"},
		},
	}
	desired := &schema.Dokkufile{
		Version:  "1",
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
	}
	actual := &schema.Dokkufile{Version: "1"}

	runner := &checkpointTestRunner{}
	executor := &Executor{
		Runner:        runner,
		CheckpointDir: dir,
		// DokkufilePath intentionally empty — simulates plugin context
	}

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	err := executor.Execute(p, desired, actual)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Without DokkufilePath, checkpoint should be ignored and step should re-run
	if len(runner.commands) != 1 {
		t.Errorf("expected 1 command (checkpoint should be ignored without DokkufilePath), got %d: %v",
			len(runner.commands), runner.commandStrings())
	}
}

func TestIsAlreadyExistsError(t *testing.T) {
	tests := []struct {
		cmd    []string
		output string
		want   bool
	}{
		{[]string{"apps:create", "myapp"}, "App myapp already exists", true},
		{[]string{"postgres:create", "mydb"}, "Service mydb already exists", true},
		{[]string{"mail:create", "mail1"}, "mail1 already exists", true},
		{[]string{"sso:create", "dir1"}, "dir1 already exists", true},
		{[]string{"sso:frontend:create", "fe1"}, "fe1 already exists", true},
		{[]string{"apps:create", "myapp"}, "some other error", false},
		{[]string{"config:set", "myapp", "K=V"}, "already exists", false},
		{[]string{"apps:destroy", "myapp"}, "already exists", false},
	}

	for _, tt := range tests {
		got := isAlreadyExistsError(tt.cmd, tt.output)
		if got != tt.want {
			t.Errorf("isAlreadyExistsError(%v, %q) = %v, want %v", tt.cmd, tt.output, got, tt.want)
		}
	}
}
