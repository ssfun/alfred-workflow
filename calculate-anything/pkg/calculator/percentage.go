// calculate-anything-go/pkg/calculators/percentage.go
package calculators

import (
	"calculate-anything-go/pkg/parser"
	"fmt"
)

func HandlePercentage(wf *aw.Workflow, p *parser.ParsedQuery) {
	var result float64
    var title, arg string

	switch p.Action {
	case "+":
		result = p.BaseValue * (1 + p.Percent/100)
        title = fmt.Sprintf("%.2f + %.2f%% = %.2f", p.BaseValue, p.Percent, result)
        arg = fmt.Sprintf("%.2f", result)
	case "-":
		result = p.BaseValue * (1 - p.Percent/100)
        title = fmt.Sprintf("%.2f - %.2f%% = %.2f", p.BaseValue, p.Percent, result)
        arg = fmt.Sprintf("%.2f", result)
    case "of":
        result = (p.Percent / 100) * p.BaseValue
        title = fmt.Sprintf("%.2f%% of %.2f = %.2f", p.Percent, p.BaseValue, result)
        arg = fmt.Sprintf("%.2f", result)
    case "as % of":
        if p.BaseValue == 0 {
            alfred.ShowError(wf, fmt.Errorf("不能计算 0 的百分比"))
            return
        }
        result = (p.Amount / p.BaseValue) * 100
        title = fmt.Sprintf("%g 是 %g 的 %.2f%%", p.Amount, p.BaseValue, result)
        arg = fmt.Sprintf("%.2f", result)
	default:
		alfred.ShowError(wf, fmt.Errorf("未知的百分比操作: %s", p.Action))
        return
	}
	
    alfred.AddToWorkflow(wf, []alfred.Result{{
        Title: title,
        Subtitle: fmt.Sprintf("复制 '%s'", arg),
        Arg: arg,
    }})
}
