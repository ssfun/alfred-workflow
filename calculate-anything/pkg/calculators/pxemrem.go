// calculate-anything/pkg/calculators/pxemrem.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"strconv"
	"strings"

	"github.com/deanishe/awgo"
)

// 1pt (point) 等于 4/3 px (pixel) 是一个标准的 Web 和印刷转换因子
const ptToPxFactor = 4.0 / 3.0

// HandlePxEmRem 处理 Web 开发单位 px, em, rem, pt 之间的转换。
func HandlePxEmRem(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	// 从配置中获取用户设置的基础像素值 (例如 "16px")
	basePxString := strings.TrimSuffix(strings.ToLower(cfg.PixelsBase), "px")
	basePx, err := strconv.ParseFloat(strings.TrimSpace(basePxString), 64)
	if err != nil || basePx == 0 {
		alfred.ShowError(wf, fmt.Errorf("无效的基础像素配置: %s", cfg.PixelsBase))
		return
	}

	// 步骤 1: 将所有输入值统一转换为 px，作为计算的基准
	var valueInPx float64
	fromUnit := strings.ToLower(p.From)
	switch fromUnit {
	case "px":
		valueInPx = p.Amount
	case "em", "rem": // em 和 rem 在此上下文中等价，都相对于基础像素值
		valueInPx = p.Amount * basePx
	case "pt":
		valueInPx = p.Amount * ptToPxFactor
	default:
		alfred.ShowError(wf, fmt.Errorf("未知的源单位: %s", p.From))
		return
	}

	// 场景 1: 如果用户明确指定了目标单位 (e.g., "2rem to pt")
	if p.To != "" {
		toUnit := strings.ToLower(p.To)
		var resultValue float64
		// 从基准 px 值转换到目标单位
		switch toUnit {
		case "px":
			resultValue = valueInPx
		case "em", "rem":
			resultValue = valueInPx / basePx
		case "pt":
			resultValue = valueInPx / ptToPxFactor
		default:
			alfred.ShowError(wf, fmt.Errorf("未知的目标单位: %s", p.To))
			return
		}
		resultString := fmt.Sprintf("%g", resultValue)
		title := fmt.Sprintf("%g%s = %s%s", p.Amount, fromUnit, resultString, toUnit)
		alfred.AddToWorkflow(wf, []alfred.Result{{
			Title:    title,
			Subtitle: fmt.Sprintf("复制 '%s'", resultString),
			Arg:      resultString,
		}})
		return
	}

	// 场景 2: 如果用户只输入了一个值 (e.g., "12px" or "2rem")，则显示所有可能的转换
	pxValue := valueInPx
	emValue := valueInPx / basePx
	ptValue := valueInPx / ptToPxFactor

	results := []alfred.Result{
		{
			Title:    fmt.Sprintf("%g px", pxValue),
			Subtitle: fmt.Sprintf("基础字号: %gpx | 复制 'px' 值", basePx),
			Arg:      fmt.Sprintf("%g", pxValue),
		},
		{
			Title:    fmt.Sprintf("%g em/rem", emValue),
			Subtitle: fmt.Sprintf("基础字号: %gpx | 复制 'em/rem' 值", basePx),
			Arg:      fmt.Sprintf("%g", emValue),
		},
		{
			Title:    fmt.Sprintf("%g pt", ptValue),
			Subtitle: fmt.Sprintf("基础字号: %gpx | 复制 'pt' 值", basePx),
			Arg:      fmt.Sprintf("%g", ptValue),
		},
	}
	alfred.AddToWorkflow(wf, results)
}
