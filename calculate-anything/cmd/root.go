// calculate-anything-go/cmd/root.go
package cmd

import (
	"calculate-anything-go/pkg/alfred"
	"calculate-anything-go/pkg/calculators"
	"calculate-anything-go/pkg/config"
	"calculate-anything-go/pkg/parser"
	"github.com/deanishe/awgo"
	"strings"
)

// Run 是 Alfred 执行的入口
func Run(wf *aw.Workflow) {
	cfg := config.Load(wf)

	if len(wf.Args()) == 0 {
		return
	}
	query := wf.Args()[0]

	if handleSpecialCommands(wf, query) {
		wf.SendFeedback()
		return
	}

	var p *parser.ParsedQuery
	if strings.HasPrefix(strings.ToLower(query), "time ") {
		p = &parser.ParsedQuery{Type: parser.TimeQuery, Input: strings.TrimPrefix(query, "time ")}
	} else if strings.HasPrefix(strings.ToLower(query), "vat ") {
		p = &parser.ParsedQuery{Type: parser.VATQuery, Input: strings.TrimPrefix(query, "vat ")}
	} else {
		p = parser.Parse(query)
	}

	switch p.Type {
	case parser.CurrencyQuery:
		calculators.HandleCurrency(wf, cfg, p)
	case parser.UnitQuery:
		calculators.HandleUnits(wf, p)
	case parser.PercentageQuery:
		calculators.HandlePercentage(wf, p)
	case parser.TimeQuery:
		calculators.HandleTime(wf, cfg, p)
	// case parser.VATQuery:
	// 	calculators.HandleVAT(wf, cfg, p)
	// case parser.PxEmRemQuery:
	// 	calculators.HandlePxEmRem(wf, cfg, p)
	case parser.UnknownQuery:
		wf.NewItem("无法解析查询 '"+query+"'").
			Subtitle("尝试: '100 usd to eur', '10km in mi', '120 + 15%', 'time +3 days'").
			Valid(false)
	default:
		wf.NewItem(fmt.Sprintf("查询类型 '%v' 暂未实现", p.Type)).Valid(false)
	}

	wf.SendFeedback()
}

func handleSpecialCommands(wf *aw.Workflow, query string) bool {
	if query == "_caclear" {
		if err := wf.Cache.Clear(); err != nil {
			alfred.ShowError(wf, fmt.Errorf("清除缓存失败: %w", err))
		} else {
			alfred.AddToWorkflow(wf, []alfred.Result{{Title: "缓存已成功清除"}})
		}
		return true
	}
	return false
}
