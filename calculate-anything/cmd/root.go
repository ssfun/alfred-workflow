// calculate-anything-go/cmd/root.go
package cmd

import (
	"calculate-anything-go/pkg/config"
	"calculate-anything-go/pkg/parser"
	"calculate-anything-go/pkg/calculators"
	"fmt"
	"github.com/deanishe/awgo"
	"strings"
)

// Run 是 Alfred 执行的入口
func Run(wf *aw.Workflow) {
	cfg := config.Load(wf)
	
	// 从命令行参数获取查询
	query := wf.Args()[0]

	// 特殊命令处理
	if handleSpecialCommands(wf, query) {
		wf.SendFeedback()
		return
	}

	// 关键字触发
	var p *parser.ParsedQuery
	if strings.HasPrefix(query, "time ") {
		p = &parser.ParsedQuery{Type: parser.TimeQuery, Input: strings.TrimPrefix(query, "time ")}
	} else if strings.HasPrefix(query, "vat ") {
		p = &parser.ParsedQuery{Type: parser.VATQuery, Input: strings.TrimPrefix(query, "vat ")}
	} else {
		// 通用查询解析
		p = parser.Parse(query)
	}

	// 根据解析结果调用不同的处理函数
	var result string
	var err error

	switch p.Type {
	case parser.CurrencyQuery:
		result, err = calculators.HandleCurrency(p)
	case parser.PercentageQuery:
		result, err = calculators.HandlePercentage(p)
	// ... 其他 case
	case parser.UnknownQuery:
		// 如果无法解析，可以不显示任何结果或显示帮助信息
		wf.NewItem("无法解析查询").Subtitle("请尝试 '100 usd to eur' 或查看帮助文档").Valid(false)
	default:
		result = fmt.Sprintf("暂未实现: %v", p.Type)
	}

	if err != nil {
		wf.NewWarning("计算出错", err.Error())
	} else if result != "" {
		// 将结果添加到 Alfred
		wf.NewItem(result).
			Subtitle(fmt.Sprintf("复制 '%s' 到剪贴板", result)).
			Arg(result). // 这是回车后复制到剪贴板的内容
			Valid(true)
	}

	// 向 Alfred 发送最终的反馈
	wf.SendFeedback()
}

func handleSpecialCommands(wf *aw.Workflow, query string) bool {
	if query == "_caclear" {
		if err := wf.Cache.Clear(); err != nil {
			wf.NewWarning("清除缓存失败", err.Error())
		} else {
			wf.NewItem("缓存已清除")
		}
		return true
	}
	// ... 其他特殊命令
	return false
}
