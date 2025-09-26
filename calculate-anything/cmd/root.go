// calculate-anything/cmd/root.go
package cmd

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/calculator"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"strings"
)

// Run 是 Alfred 执行的入口
func Run(wf *aw.Workflow) {
	// 加载用户在 Alfred Workflow 中设置的配置
	cfg := config.Load(wf)

	if len(wf.Args()) == 0 {
		return // 如果没有输入参数则直接退出
	}
	query := wf.Args()[0]

	// 首先处理特殊命令，如清除缓存
	if handleSpecialCommands(wf, query) {
		wf.SendFeedback()
		return
	}

	var p *parser.ParsedQuery
	// 检查是否有关键字触发器，如 'time' 或 'vat'
	if strings.HasPrefix(strings.ToLower(query), "time ") {
		p = &parser.ParsedQuery{Type: parser.TimeQuery, Input: strings.TrimPrefix(query, "time ")}
	} else if strings.HasPrefix(strings.ToLower(query), "vat ") {
		p = &parser.ParsedQuery{Type: parser.VATQuery, Input: strings.TrimPrefix(query, "vat ")}
	} else {
		// 如果没有关键字，则使用通用的智能解析器
		p = parser.Parse(query)
	}

	// ----------------------------------------------------
	// 智能分发逻辑:
	// 解析器返回了一个通用的 UnitQuery，这里需要根据单位的具体内容，
	// 将其细化为 Currency, Crypto, DataStorage 或保持为 Unit。
	// ----------------------------------------------------
	if p.Type == parser.UnitQuery {
		from := strings.ToUpper(p.From)
		to := strings.ToUpper(p.To)
		
		if calculators.IsCrypto(from) || calculators.IsCrypto(to) {
			p.Type = parser.CryptoQuery
		} else if calculators.IsCurrency(from) || calculators.IsCurrency(to) {
			p.Type = parser.CurrencyQuery
		} else if calculators.IsDataStorageUnit(p.From) || calculators.IsDataStorageUnit(p.To) {
			p.Type = parser.DataStorageQuery
		}
		// 如果都不是，则它就是一个普通的物理单位查询 (UnitQuery)
	}

	// 根据最终确定的查询类型，调用相应的计算器
	switch p.Type {
	case parser.CurrencyQuery:
		calculators.HandleCurrency(wf, cfg, p)
	case parser.CryptoQuery:
		calculators.HandleCrypto(wf, cfg, p)
	case parser.UnitQuery:
		calculators.HandleUnits(wf, p)
	case parser.DataStorageQuery:
		calculators.HandleDataStorage(wf, cfg, p)
	case parser.PercentageQuery:
		calculators.HandlePercentage(wf, p)
	case parser.TimeQuery:
		calculators.HandleTime(wf, cfg, p)
	case parser.VATQuery:
		calculators.HandleVAT(wf, cfg, p)
	case parser.PxEmRemQuery:
		calculators.HandlePxEmRem(wf, cfg, p)
	case parser.UnknownQuery:
		// 如果无法解析，向用户显示提示信息
		wf.NewItem("无法解析查询 '"+query+"'").
			Subtitle("请尝试: '100 usd to eur', '10km in mi', '120 + 15%', 'time +3 days'").
			Valid(false)
	default:
		wf.NewItem(fmt.Sprintf("查询类型 '%v' 暂未实现", p.Type)).Valid(false)
	}

	// 将所有生成的反馈项发送给 Alfred 进行显示
	wf.SendFeedback()
}

// handleSpecialCommands 处理内部命令，如清除缓存
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
