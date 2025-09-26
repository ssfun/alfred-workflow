package main

import (
	"github.com/deanishe/awgo"
	"calculate-anything-go/cmd"
)

var wf *aw.Workflow

func init() {
	wf = aw.New()
}

func main() {
	// 将实际的业务逻辑封装在 cmd.Run 中
	// awgo 会自动处理 panic 并向 Alfred 显示错误
	wf.Run(cmd.Run)
}
