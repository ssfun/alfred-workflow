// alfred.go
package main

import (
	"encoding/json"
	"os"
)

// Workflow 是 Alfred Script Filter 的顶层结构
type Workflow struct {
	items []*Item `json:"items"`
}

// Item 是 Script Filter 中的一个条目
type Item struct {
	UID      string `json:"uid,omitempty"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Arg      string `json:"arg,omitempty"`
	Match    string `json:"match,omitempty"`
	Valid    bool   `json:"valid"`
	Mods     Mods   `json:"mods,omitempty"`
	Icon     Icon   `json:"icon,omitempty"`
}

// Mods 定义了修饰键 (Cmd, Alt, etc.) 的行为
type Mods struct {
	Cmd Mod `json:"cmd,omitempty"`
	Alt Mod `json:"alt,omitempty"`
}

// Mod 是修饰键的具体定义
type Mod struct {
	Arg      string `json:"arg"`
	Subtitle string `json:"subtitle"`
}

// Icon 定义了 item 的图标
type Icon struct {
	Path string `json:"path"`
}

// NewWorkflow 创建一个新的 Workflow 实例
func NewWorkflow() *Workflow {
	return &Workflow{
		items: []*Item{},
	}
}

// NewItem 创建一个 item 并添加到 workflow 中
func (w *Workflow) NewItem(title string) *Item {
	item := &Item{
		Title: title,
		Valid: true, // 默认设为 true
	}
	w.items = append(w.items, item)
	return item
}

// SendFeedback 将 workflow 的内容以 JSON 格式输出到标准输出
func (w *Workflow) SendFeedback() {
	output := map[string]interface{}{
		"items": w.items,
	}
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(output); err != nil {
		// 在 Alfred 中显示错误
		w.NewItem("Failed to encode JSON").
			SetSubtitle(err.Error()).
			SetValid(false)
		errorOutput := map[string]interface{}{"items": w.items}
		_ = json.NewEncoder(os.Stdout).Encode(errorOutput)
	}
}

// --------------- Item 的链式调用方法 ---------------

// SetUID 设置 item 的 UID
func (i *Item) SetUID(uid string) *Item {
	i.UID = uid
	return i
}

// SetSubtitle 设置 item 的副标题
func (i *Item) SetSubtitle(subtitle string) *Item {
	i.Subtitle = subtitle
	return i
}

// SetArg 设置 item 的 arg (回传给 Alfred 的参数)
func (i *Item) SetArg(arg string) *Item {
	i.Arg = arg
	return i
}

// SetMatch 设置用于 Alfred 筛选的匹配字符串
func (i *Item) SetMatch(match string) *Item {
	i.Match = match
	return i
}

// SetValid 设置 item 是否有效 (能否被执行)
func (i *Item) SetValid(valid bool) *Item {
	i.Valid = valid
	return i
}

// SetCmdModifier 添加 Cmd 键的修饰符行为
func (i *Item) SetCmdModifier(arg, subtitle string) *Item {
	i.Mods.Cmd = Mod{
		Arg:      arg,
		Subtitle: subtitle,
	}
	return i
}

// SetAltModifier 添加 Alt 键的修饰符行为
func (i *Item) SetAltModifier(arg, subtitle string) *Item {
	i.Mods.Alt = Mod{
		Arg:      arg,
		Subtitle: subtitle,
	}
	return i
}

// SetIcon 设置 item 的图标
func (i *Item) SetIcon(path string) *Item {
	i.Icon.Path = path
	return i
}
