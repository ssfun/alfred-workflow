// calculate-anything/main.go
package main

import (
	"calculate-anything/cmd"
	// 修正：根据官方文档，导入时使用 aw 别名
	aw "github.com/deanishe/awgo"
)

// wf 是一个全局的 Workflow 实例，负责与 Alfred 的所有交互。
var wf *aw.Workflow

// init 函数在 main 函数执行前被调用，用于初始化 workflow 对象。
func init() {
	wf = aw.New(aw.HelpURL("https://github.com/ssfun/alfred-workflow/calculate-anything"), aw.Update(aw.GitHub("ssfun/alfred-workflow/calculate-anything")))
}

// run 是我们工作流的真正入口点。
func run() {
	// 我们的核心业务逻辑被封装在 cmd.Run 函数中。
	// 将 wf 实例传递给它，以便其他包可以使用。
	cmd.Run(wf)
}

// main 是程序的入口函数。
func main() {
	// wf.Run() 包装了主逻辑的执行。
	// 它会捕获并记录 panic，并在 Alfred 中显示错误，而不是静默失败。
	wf.Run(run)
}
