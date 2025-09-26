package cmd

import (
	"strings"
	"github.com/deanishe/awgo"
	// ... 导入你的 calculator 和 parser 包
)

var wf *aw.Workflow

func init() {
	wf = aw.New()
}

// Run 是 Alfred 执行的入口
func Run() {
	query := wf.Args()[0]

	// 根据关键字分发，类似 process.php 的 switch
	// awgo 提供了更优雅的方式来处理关键字，但这里为了演示基本逻辑
	if strings.HasPrefix(query, "time ") {
		// handleTime(strings.TrimPrefix(query, "time "))
		wf.NewItem("Time Query").Subtitle(query)
	} else if strings.HasPrefix(query, "vat ") {
		// handleVat(strings.TrimPrefix(query, "vat "))
		wf.NewItem("VAT Query").Subtitle(query)
	} else {
		// handleGeneralQuery(query)
		wf.NewItem("General Query").Subtitle(query)
	}

	// 向 Alfred 发送结果
	wf.SendFeedback()
}

// ... 实现 handleTime, handleVat, handleGeneralQuery 等函数
