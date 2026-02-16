package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

// Action describes the type of change to make.
type Action string

const (
	CreateApp      Action = "create_app"
	DestroyApp     Action = "destroy_app"
	UpdateApp      Action = "update_app"
	CreateService  Action = "create_service"
	DestroyService Action = "destroy_service"
)

// Step is a single planned change.
type Step struct {
	Action   Action `json:"action"`
	App      string `json:"app,omitempty"`
	Service  string `json:"service,omitempty"`
	Field    string `json:"field,omitempty"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// Plan holds the list of steps needed to converge actual state to desired state.
type Plan struct {
	Steps []Step `json:"steps"`
}

// String returns a human-readable summary of the plan.
func (p *Plan) String() string {
	if len(p.Steps) == 0 {
		return "No changes needed."
	}
	var b strings.Builder
	for _, s := range p.Steps {
		switch s.Action {
		case CreateApp:
			fmt.Fprintf(&b, "+ app %q\n", s.App)
		case DestroyApp:
			fmt.Fprintf(&b, "- app %q\n", s.App)
		case UpdateApp:
			fmt.Fprintf(&b, "~ app %q: %s %q → %q\n", s.App, s.Field, s.OldValue, s.NewValue)
		case CreateService:
			fmt.Fprintf(&b, "+ service %q\n", s.Service)
		case DestroyService:
			fmt.Fprintf(&b, "- service %q\n", s.Service)
		}
	}
	return b.String()
}

// Diff computes the plan to go from actual to desired state.
func Diff(desired, actual *schema.Dokkufile) *Plan {
	var steps []Step

	steps = append(steps, diffServices(desired, actual)...)
	steps = append(steps, diffApps(desired, actual)...)

	return &Plan{Steps: steps}
}

func diffServices(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredSvc := desired.Services
	actualSvc := actual.Services

	if desiredSvc == nil {
		desiredSvc = map[string]schema.Service{}
	}
	if actualSvc == nil {
		actualSvc = map[string]schema.Service{}
	}

	for name := range desiredSvc {
		if _, exists := actualSvc[name]; !exists {
			steps = append(steps, Step{
				Action:  CreateService,
				Service: name,
			})
		}
	}

	for name := range actualSvc {
		if _, exists := desiredSvc[name]; !exists {
			steps = append(steps, Step{
				Action:  DestroyService,
				Service: name,
			})
		}
	}

	return steps
}

func diffApps(desired, actual *schema.Dokkufile) []Step {
	var steps []Step
	desiredApps := desired.Apps
	actualApps := actual.Apps

	if desiredApps == nil {
		desiredApps = map[string]schema.App{}
	}
	if actualApps == nil {
		actualApps = map[string]schema.App{}
	}

	// New apps
	for name, dApp := range desiredApps {
		if _, exists := actualApps[name]; !exists {
			steps = append(steps, Step{
				Action:   CreateApp,
				App:      name,
				NewValue: dApp.Image,
			})
			continue
		}
		// Existing app — diff fields
		aApp := actualApps[name]
		steps = append(steps, diffApp(name, dApp, aApp)...)
	}

	// Removed apps
	for name := range actualApps {
		if _, exists := desiredApps[name]; !exists {
			steps = append(steps, Step{
				Action: DestroyApp,
				App:    name,
			})
		}
	}

	return steps
}

func diffApp(name string, desired, actual schema.App) []Step {
	var steps []Step

	if desired.Image != actual.Image {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "image",
			OldValue: actual.Image,
			NewValue: desired.Image,
		})
	}

	if !sliceEqual(desired.Domains, actual.Domains) {
		steps = append(steps, Step{
			Action:   UpdateApp,
			App:      name,
			Field:    "domains",
			OldValue: strings.Join(actual.Domains, ","),
			NewValue: strings.Join(desired.Domains, ","),
		})
	}

	if !mapEqual(desired.Env, actual.Env) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "env",
		})
	}

	if !mapEqual(desired.Links, actual.Links) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "links",
		})
	}

	if !mapEqual(desired.Ports, actual.Ports) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "ports",
		})
	}

	if !sliceEqual(desired.Storage, actual.Storage) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "storage",
		})
	}

	if !mapIntEqual(desired.Scale, actual.Scale) {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "scale",
		})
	}

	if desired.LetsEncrypt != actual.LetsEncrypt {
		steps = append(steps, Step{
			Action: UpdateApp,
			App:    name,
			Field:  "letsencrypt",
		})
	}

	return steps
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sorted := func(s []string) []string {
		c := make([]string, len(s))
		copy(c, s)
		sort.Strings(c)
		return c
	}
	sa, sb := sorted(a), sorted(b)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

func mapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func mapIntEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
