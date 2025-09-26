// calculate-anything-go/pkg/alfred/feedback.go
package alfred

import (
	"fmt"
	"github.com/deanishe/awgo"
)

// Result is a standard structure for a single calculation result
type Result struct {
	Title    string
	Subtitle string
	Arg      string // Value copied to clipboard
	IconPath string
}

// AddToWorkflow adds a slice of Results to the Alfred workflow feedback
func AddToWorkflow(wf *aw.Workflow, results []Result) {
	for _, r := range results {
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true)

		if r.IconPath != "" {
			item.Icon(&aw.Icon{Value: r.IconPath})
		}
	}
}

// ShowError displays a user-friendly error in Alfred
func ShowError(wf *aw.Workflow, err error) {
	wf.NewWarning("计算出错", err.Error())
}
