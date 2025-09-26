// calculate-anything/cmd/root.go
package cmd

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/calculators"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/i18n"
	"calculate-anything/pkg/parser"
	"fmt"
	"strings"

	aw "github.com/deanishe/awgo"
)

// Run 是 Alfred 执行的入口函数，也是整个应用的大脑。
func Run(wf *aw.Workflow) {
	// 步骤 1: 加载用户在 Alfred Workflow 中设置的配置（语言、API密钥等）
	cfg := config.Load(wf)

	// 如果没有任何输入参数，则直接退出
	if len(wf.Args()) == 0 {
		return
	}
	query := wf.Args()[0]

	// 步骤 2: 加载与用户配置语言相对应的语言包（用于关键字和停用词）
	langPack, err := i18n.LoadLanguagePack(cfg.Language)
	if err != nil {
		// 即使语言包加载失败，也应该继续执行，只是关键字功能会受限。
		// 在当前 awgo 版本中不依赖 Logger()，改为在界面上提示一次。
		wf.Warn(fmt.Sprintf("无法加载语言包: %v", err), "提示")
	}

	// 步骤 3: 检查是否是特殊内部命令，如 "_caclear" 用于清除缓存
	if handleSpecialCommands(wf, query) {
		wf.SendFeedback() // 发送反馈并退出
		return
	}

	// 步骤 4: 检查是否是颜色代码，如果是，则直接调用颜色计算器
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if strings.HasPrefix(trimmedQuery, "#") || strings.HasPrefix(trimmedQuery, "rgb(") {
		calculators.HandleColor(wf, query)
		wf.SendFeedback()
		return
	}

	var p *parser.ParsedQuery
	// 步骤 5: 检查是否由特定关键字触发，如 'time' 或 'vat'
	if strings.HasPrefix(trimmedQuery, "time ") {
		p = &parser.ParsedQuery{Type: parser.TimeQuery, Input: strings.TrimPrefix(query, "time ")}
	} else if strings.HasPrefix(trimmedQuery, "vat ") {
		p = &parser.ParsedQuery{Type: parser.VATQuery, Input: strings.TrimPrefix(query, "vat ")}
	} else {
		// 如果没有特定关键字，则使用通用的智能解析器进行解析
		p = parser.Parse(query, langPack)
	}

	// 步骤 6: 智能分发逻辑。解析器返回了一个通用的 UnitQuery，
	// 这里需要根据单位的具体内容，将其细化为 Currency, Crypto, DataStorage 或保持为 Unit。
	if p.Type == parser.UnitQuery {
		from := strings.ToUpper(p.From)
		to := strings.ToUpper(p.To)

		// 检查单位是否属于特定类别，并更新查询类型
		if calculators.IsCrypto(from) || calculators.IsCrypto(to) {
			p.Type = parser.CryptoQuery
		} else if calculators.IsCurrency(from) || calculators.IsCurrency(to) {
			p.Type = parser.CurrencyQuery
		} else if calculators.IsDataStorageUnit(p.From) || calculators.IsDataStorageUnit(p.To) {
			p.Type = parser.DataStorageQuery
		}
		// 如果都不是，则它就是一个普通的物理单位查询 (UnitQuery)，无需改变
	}

	// 步骤 7: 根据最终确定的查询类型，调用相应的计算器处理模块
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
		// 如果所有解析都失败，向用户显示有用的提示信息
		wf.NewItem("无法解析查询 '"+query+"'").
			Subtitle("请尝试: '100 usd to eur', '10km in mi', '120 + 15%', 'time +3 days'").
			Valid(false)
	default:
		// 为尚未实现的查询类型提供一个占位符
		wf.NewItem(fmt.Sprintf("查询类型 '%v' 暂未实现", p.Type)).Valid(false)
	}

	// 步骤 8: 将所有生成的反馈项发送给 Alfred 进行显示
	wf.SendFeedback()
}

// handleSpecialCommands 处理内部命令，目前只支持清除缓存。
func handleSpecialCommands(wf *aw.Workflow, query string) bool {
	if query == "_caclear" {
		if err := wf.ClearCache(); err != nil {
			alfred.ShowError(wf, fmt.Errorf("清除缓存失败: %w", err))
		} else {
			alfred.AddToWorkflow(wf, []alfred.Result{{Title: "缓存已成功清除"}})
		}
		return true // 表示已处理
	}
	return false // 表示未处理
}
