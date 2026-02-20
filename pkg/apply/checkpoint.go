package apply

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deanmarano/dokkufile/pkg/plan"
)

const checkpointFileName = ".dokkufile-checkpoint.json"

// Checkpoint records the state of a partially-completed apply so that a
// subsequent apply can resume from where it left off.
type Checkpoint struct {
	DokkufileHash  string          `json:"dokkufile_hash"`
	CompletedSteps []CompletedStep `json:"completed_steps"`
	FailedStep     *StepRecord     `json:"failed_step,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

// CompletedStep records a step that was successfully executed.
type CompletedStep struct {
	StepKey  string   `json:"step_key"`
	Commands []string `json:"commands"`
}

// StepRecord records a step that failed, for diagnostics.
type StepRecord struct {
	StepKey string `json:"step_key"`
	Error   string `json:"error"`
}

// StepKey returns a unique, deterministic identifier for a plan step.
func StepKey(s plan.Step) string {
	switch {
	case s.App != "" && s.Field != "":
		return string(s.Action) + ":" + s.App + ":" + s.Field
	case s.App != "":
		return string(s.Action) + ":" + s.App
	case s.Service != "" && s.ServiceType != "":
		return string(s.Action) + ":" + s.ServiceType + ":" + s.Service
	case s.Service != "":
		return string(s.Action) + ":" + s.Service
	case s.Field != "":
		return string(s.Action) + ":" + s.Field
	default:
		return string(s.Action)
	}
}

// loadCheckpoint reads a checkpoint file from the given directory.
// Returns nil, nil if no checkpoint file exists.
func loadCheckpoint(dir string) (*Checkpoint, error) {
	data, err := os.ReadFile(filepath.Join(dir, checkpointFileName))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parsing checkpoint: %w", err)
	}
	return &cp, nil
}

// saveCheckpoint writes a checkpoint file to the given directory.
func saveCheckpoint(dir string, cp *Checkpoint) error {
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, checkpointFileName), data, 0644)
}

// clearCheckpoint removes the checkpoint file from the given directory.
func clearCheckpoint(dir string) error {
	err := os.Remove(filepath.Join(dir, checkpointFileName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// hashFile returns the hex-encoded SHA-256 hash of a file's contents.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

// ClearCheckpointDir removes the checkpoint file from a directory.
// This is exported for use by the CLI clear-checkpoint flag.
func ClearCheckpointDir(dir string) error {
	return clearCheckpoint(dir)
}

// newCheckpoint creates a new Checkpoint with the given dokkufile hash.
func newCheckpoint(dokkufileHash string) *Checkpoint {
	return &Checkpoint{
		DokkufileHash: dokkufileHash,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}
