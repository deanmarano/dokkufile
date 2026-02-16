package apply

import (
	"fmt"

	"github.com/deanmarano/dokkufile/pkg/plan"
)

// Executor applies a plan by shelling out to dokku commands.
type Executor struct {
	DryRun bool
}

// Execute runs each step in the plan.
// TODO: implement actual dokku command execution.
func (e *Executor) Execute(p *plan.Plan) error {
	for _, step := range p.Steps {
		if e.DryRun {
			fmt.Printf("[dry-run] %s\n", describeStep(step))
			continue
		}
		fmt.Printf("Executing: %s\n", describeStep(step))
		// TODO: shell out to dokku
	}
	return nil
}

func describeStep(s plan.Step) string {
	switch s.Action {
	case plan.CreateApp:
		return fmt.Sprintf("create app %q (image: %s)", s.App, s.NewValue)
	case plan.DestroyApp:
		return fmt.Sprintf("destroy app %q", s.App)
	case plan.UpdateApp:
		return fmt.Sprintf("update app %q field %s: %q → %q", s.App, s.Field, s.OldValue, s.NewValue)
	case plan.CreateService:
		return fmt.Sprintf("create service %q", s.Service)
	case plan.DestroyService:
		return fmt.Sprintf("destroy service %q", s.Service)
	default:
		return fmt.Sprintf("unknown action: %s", s.Action)
	}
}
