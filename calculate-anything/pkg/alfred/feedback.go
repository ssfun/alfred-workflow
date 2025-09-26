// calculate-anything/pkg/alfred/feedback.go
package alfred

import (
	"github.com/deanishe/awgo"
)

// Modifier holds data for an Alfred modifier key (e.g., Cmd, Opt).
type Modifier struct {
	Key      aw.ModKey // e.g., aw.ModCmd, aw.ModOpt
	Subtitle string
	Arg      string
}

// Result is a standard structure for a single calculation result.
type Result struct {
	Title     string
	Subtitle  string
	Arg       string // Value copied to clipboard
	IconPath  string
	Modifiers []Modifier
}

// AddToWorkflow adds a slice of Results to the Alfred workflow feedback.
func AddToWorkflow(wf *aw.Workflow, results []Result) {
	for _, r := range results {
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true)

		if r.IconPath != "" {
			item.Icon(&aw.Icon{Value: r.IconPath})
		}

		// Add modifier keys
		for _, mod := range r.Modifiers {
			item.NewModifier(mod.Key).
				Subtitle(mod.Subtitle).
				Arg(mod.Arg)
		}
	}
}

// ShowError displays a user-friendly error in Alfred's results.
func ShowError(wf *aw.Workflow, err error) {
	// Corrected: Use wf.Warn() instead of wf.NewWarning()
	wf.Warn("Calculation Error", err.Error())
}
