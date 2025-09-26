// calculate-anything/pkg/calculators/percentage.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
)

// HandlePercentage 处理所有类型的百分比计算。
func HandlePercentage(wf *aw.Workflow, p *parser.ParsedQuery) {
	var result float64
	var title, arg string

	// 根据解析器识别出的不同动作，执行相应的计算
	switch p.Action {
	// 场景 1: "120 + 30%"
	case "+":
		result = p.BaseValue * (1 + p.Percent/100)
		title = fmt.Sprintf("%g + %g%% = %g", p.BaseValue, p.Percent, result)
		arg = fmt.Sprintf("%g", result)

	// 场景 2: "120 - 30%"
	case "-":
		result = p.BaseValue * (1 - p.Percent/100)
		title = fmt.Sprintf("%g - %g%% = %g", p.BaseValue, p.Percent, result)
		arg = fmt.Sprintf("%g", result)

	// 场景 3: "15% of 50"
	case "of":
		result = (p.Percent / 100) * p.BaseValue
		title = fmt.Sprintf("%g%% of %g = %g", p.Percent, p.BaseValue, result)
		arg = fmt.Sprintf("%g", result)

	// 场景 4: "40 as a % of 50"
	case "as % of":
		if p.BaseValue == 0 {
			alfred.ShowError(wf, fmt.Errorf("不能计算 0 的百分比"))
			return
		}
		result = (p.Amount / p.BaseValue) * 100
		title = fmt.Sprintf("%g 是 %g 的 %g%%", p.Amount, p.BaseValue, result)
		arg = fmt.Sprintf("%g", result)

	default:
		alfred.ShowError(wf, fmt.Errorf("未知的百分比操作: %s", p.Action))
		return
	}

	// 将计算结果添加到 Alfred 反馈
	alfred.AddToWorkflow(wf, []alfred.Result{{
		Title:    title,
		Subtitle: fmt.Sprintf("复制 '%s'", arg),
		Arg:      arg,
	}})
}
