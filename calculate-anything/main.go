// calculate-anything-go/main.go
package main

import (
    "calculate-anything/cmd"
    "github.com/deanishe/awgo"
)

// wf 是一个全局的 Workflow 实例
var wf *aw.Workflow

func init() {
    // 初始化一个新的 Workflow 对象
    wf = aw.New(aw.HelpURL("https://github.com/biati-digital/alfred-calculate-anything"), aw.Update(aw.GitHub("biati-digital/alfred-calculate-anything")))
}

func main() {
    // 将实际的业务逻辑传递给 wf.Run
    // awgo 会处理查询参数、panic 恢复和向 Alfred 发送反馈
    wf.Run(func() {
        cmd.Run(wf)
    })
}
