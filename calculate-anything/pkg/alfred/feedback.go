// calculate-anything/pkg/alfred/feedback.go
package alfred

import (
	"github.com/deanishe/awgo"
)

// Modifier 定义了 Alfred 结果的修饰键（如 Cmd, Opt, Ctrl）。
// 当用户按住这些键时，可以看到并执行不同的操作。
type Modifier struct {
	Key      aw.ModKey // 修饰键的类型，例如 aw.ModCmd, aw.ModOpt
	Subtitle string    // 按下修饰键后显示的副标题
	Arg      string    // 按下修饰键后，回车所执行的参数（通常是复制到剪贴板的内容）
}

// Result 是用于生成单个 Alfred 结果项的标准结构。
// 它统一了所有计算器返回结果的格式。
type Result struct {
	Title     string     // Alfred 结果项的主标题
	Subtitle  string     // 结果项的副标题
	Arg       string     // 默认操作（回车）的参数，通常是复制到剪贴板的内容
	IconPath  string     // 结果项的图标路径
	Modifiers []Modifier // 附加的修饰键操作列表
}

// AddToWorkflow 将一组标准化的 Result 对象添加到 Alfred 的反馈列表中。
func AddToWorkflow(wf *aw.Workflow, results []Result) {
	for _, r := range results {
		// 为每个 Result 创建一个新的 Alfred 结果项
		item := wf.NewItem(r.Title).
			Subtitle(r.Subtitle).
			Arg(r.Arg).
			Valid(true) // 表示这是一个可执行的结果

		// 如果指定了图标路径，则设置图标
		if r.IconPath != "" {
			item.Icon(&aw.Icon{Value: r.IconPath})
		}

		// 遍历并添加所有的修饰键操作
		for _, mod := range r.Modifiers {
			item.NewModifier(mod.Key).
				Subtitle(mod.Subtitle).
				Arg(mod.Arg)
		}
	}
}

// ShowError 在 Alfred 中显示一个用户友好的错误信息。
func ShowError(wf *aw.Workflow, err error) {
	// awgo 库推荐使用 Warn() 方法来显示非致命的错误信息。
	// 第一个参数是错误详情，第二个参数是错误的标题。
	wf.Warn(err.Error(), "计算出错")
}
