// calculate-anything/pkg/alfred/feedback.go
package alfred

import (
	// 修正：根据官方文档，统一使用 aw 别名导入
	aw "github.com/deanishe/awgo"
)

// Modifier 定义了 Alfred 结果的修饰键（如 Cmd, Opt, Ctrl）。
type Modifier struct {
	// 修正：兼容 awgo 版本，使用 string 保存修饰键
	Key      string
	Subtitle string
	Arg      string
}

// Result 是用于生成单个 Alfred 结果项的标准结构。
type Result struct {
	Title     string
	Subtitle  string
	Arg       string
	IconPath  string
	Modifiers []Modifier
}

// AddToWorkflow 将一组标准化的 Result 对象添加到 Alfred 的反馈列表中。
func AddToWorkflow(wf *aw.Workflow, results []Result) {
	for _, r := range results {
		// 修正：wf 的类型是 *aw.Workflow
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true)

		if r.IconPath != "" {
			// 修正：使用正确的类型 aw.Icon
			item.Icon(&aw.Icon{Value: r.IconPath})
		}

		for _, mod := range r.Modifiers {
			item.NewModifier(mod.Key).
				Subtitle(mod.Subtitle).
				Arg(mod.Arg)
		}
	}
}

// ShowError 在 Alfred 中显示一个用户友好的错误信息。
func ShowError(wf *aw.Workflow, err error) {
	// 修正：wf 的类型是 *aw.Workflow
	wf.Warn(err.Error(), "计算出错")
}
