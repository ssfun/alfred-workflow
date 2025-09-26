// calculate-anything-go/pkg/calculators/pxemrem.go
package calculators

import (
	"calculate-anything-go/pkg/alfred"
	"calculate-anything-go/pkg/config"
	"calculate-anything-go/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"strconv"
	"strings"
)

const ptToPxFactor = 4.0 / 3.0 // 1pt = 4/3 px

// HandlePxEmRem 处理 px, em, rem, pt 之间的转换
func HandlePxEmRem(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	basePxString := strings.TrimSuffix(strings.ToLower(cfg.PixelsBase), "px")
	basePx, err := strconv.ParseFloat(strings.TrimSpace(basePxString), 64)
	if err != nil || basePx == 0 {
		alfred.ShowError(wf, fmt.Errorf("无效的基础像素配置: %s", cfg.PixelsBase))
		return
	}

	// 首先，将所有输入值转换为 px
	var valueInPx float64
	fromUnit := strings.ToLower(p.From)
	switch fromUnit {
	case "px":
		valueInPx = p.Amount
	case "em", "rem":
		valueInPx = p.Amount * basePx
	case "pt":
		valueInPx = p.Amount * ptToPxFactor
	default:
		alfred.ShowError(wf, fmt.Errorf("未知的源单位: %s", p.From))
		return
	}

	// 如果指定了目标单位，则只显示一个结果
	if p.To != "" {
		toUnit := strings.ToLower(p.To)
		var resultValue float64
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

	// 如果未指定目标单位，则显示所有可能的转换
	pxValue := valueInPx
	emValue := valueInPx / basePx
	ptValue := valueInPx / ptToPxFactor

	results := []alfred.Result{
		{
			Title:    fmt.Sprintf("%g px", pxValue),
			Subtitle: fmt.Sprintf("Base: %gpx | 复制 'px' 值", basePx),
			Arg:      fmt.Sprintf("%g", pxValue),
		},
		{
			Title:    fmt.Sprintf("%g em/rem", emValue),
			Subtitle: fmt.Sprintf("Base: %gpx | 复制 'em/rem' 值", basePx),
			Arg:      fmt.Sprintf("%g", emValue),
		},
		{
			Title:    fmt.Sprintf("%g pt", ptValue),
			Subtitle: fmt.Sprintf("Base: %gpx | 复制 'pt' 值", basePx),
			Arg:      fmt.Sprintf("%g", ptValue),
		},
	}
	alfred.AddToWorkflow(wf, results)
}
