// calculate-anything/main.go
package main

import (
	"calculate-anything/cmd"
	"github.com/deanishe/awgo"
)

// wf 是一个全局的 Workflow 实例，负责与 Alfred 的所有交互。
var wf *aw.Workflow

// init 函数在 main 函数执行前被调用，用于初始化 workflow 对象。
func init() {
	// New() 方法创建并配置 workflow 实例。
	// aw.HelpURL() 设置 workflow 的帮助文档链接。
	// aw.Update() 配置自动更新功能，指向项目的 GitHub 仓库。
	wf = aw.New(aw.HelpURL("https://github.com/ssfun/alfred-workflow/calculate-anything"), aw.Update(aw.GitHub("ssfun/alfred-workflow/calculate-anything")))
}

// main 是程序的入口函数。
func main() {
	// wf.Run() 包装了主逻辑的执行。
	// 它会处理命令行参数的解析、panic 的恢复，并在最后自动向 Alfred 发送反馈结果。
	// 我们的核心业务逻辑被封装在 cmd.Run 函数中。
	wf.Run(func() {
		cmd.Run(wf)
	})
}
