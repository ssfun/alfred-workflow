// calculate-anything/pkg/alfred/feedback.go
package alfred

import (
	"github.com/deanishe/awgo"
)

// Modifier 定义了 Alfred 结果的修饰键（如 Cmd, Opt）
type Modifier struct {
	Key      aw.ModKey // 例如 aw.ModCmd, aw.ModOpt
	Subtitle string
	Arg      string
}

// Result 是用于生成单个 Alfred 结果项的标准结构
type Result struct {
	Title     string
	Subtitle  string
	Arg       string     // 回车后复制到剪贴板的内容
	IconPath  string
	Modifiers []Modifier // 附加的修饰键操作
}

// AddToWorkflow 将一组 Result 添加到 Alfred 的反馈列表中
func AddToWorkflow(wf *aw.Workflow, results []Result) {
	for _, r := range results {
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true)

		if r.IconPath != "" {
			item.Icon(&aw.Icon{Value: r.IconPath})
		}

		// 添加修饰键
		for _, mod := range r.Modifiers {
			item.NewModifier(mod.Key).
				Subtitle(mod.Subtitle).
				Arg(mod.Arg)
		}
	}
}

// ShowError 在 Alfred 中显示一个用户友好的错误信息
func ShowError(wf *aw.Workflow, err error) {
	// 修正: awgo库中使用的是 Warn() 方法
	wf.Warn(err.Error(), "计算出错")
}
