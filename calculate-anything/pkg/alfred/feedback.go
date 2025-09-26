// calculate-anything/pkg/alfred/feedback.go
package alfred

import (
	// 根据官方文档和您的指示，统一使用 aw 别名导入
	aw "github.com/deanishe/awgo"
)

// Modifier 定义了 Alfred 结果的修饰键（如 Cmd, Opt, Ctrl）。
type Modifier struct {
	// 修正：您是对的，这里不应该使用 aw.ModKey。
	// aw.ModCmd, aw.ModOpt 等常量本身是字符串类型，
	// 因此这里直接使用 string 类型来接收它们。
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
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true)

		if r.IconPath != "" {
			item.Icon(&aw.Icon{Value: r.IconPath})
		}

		// 修正：移除了错误的 aw.ModKey() 类型转换。
		// NewModifier 方法的参数是 aw.ModKey，而 aw.ModKey 的底层类型是 string，
		// 因此可以直接传递 string 类型的 mod.Key。
		for _, mod := range r.Modifiers {
			item.NewModifier(aw.ModKey(mod.Key)).
				Subtitle(mod.Subtitle).
				Arg(mod.Arg)
		}
	}
}

// ShowError 在 Alfred 中显示一个用户友好的错误信息。
func ShowError(wf *aw.Workflow, err error) {
	wf.Warn(err.Error(), "计算出错")
}
