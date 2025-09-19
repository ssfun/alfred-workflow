package main

import (
	"encoding/json"
	"os"
	"strconv"
)

// Workflow 是 Alfred Script Filter 的顶层结构
type Workflow struct {
	Items []*Item `json:"items"`
}

// Item 是 Script Filter 中的一个条目
type Item struct {
	UID          string    `json:"uid,omitempty"`
	Title        string    `json:"title"`
	Subtitle     string    `json:"subtitle,omitempty"`
	Arg          string    `json:"arg,omitempty"`
	Match        string    `json:"match,omitempty"`
	Valid        bool      `json:"valid"`
	Mods         *Mods     `json:"mods,omitempty"`
	Autocomplete string    `json:"autocomplete,omitempty"`
	Icon         *Icon     `json:"icon,omitempty"`
	Text         *Text     `json:"text,omitempty"`
}

// Mods 定义了修饰键 (Cmd, Alt, etc.) 的行为
type Mods struct {
	Cmd *Mod `json:"cmd,omitempty"`
	Alt *Mod `json:"alt,omitempty"`
}

// Mod 是修饰键的具体定义
type Mod struct {
	Arg      string `json:"arg"`
	Subtitle string `json:"subtitle"`
	Valid    bool   `json:"valid"`
}

// Icon 定义了 item 的图标
type Icon struct {
	Type string `json:"type,omitempty"`
	Path string `json:"path,omitempty"`
}

// Text 定义了 copy 和 largetype 的文本
type Text struct {
	Copy      string `json:"copy,omitempty"`
	LargeType string `json:"largetype,omitempty"`
}

// NewWorkflow 创建一个新的 Workflow 实例
func NewWorkflow() *Workflow {
	return &Workflow{
		Items: []*Item{},
	}
}

// NewItem 创建一个新的 Item 并添加到 Workflow 中
func (w *Workflow) NewItem(title string) *Item {
	item := &Item{
		Title: title,
		Valid: true, // 默认是有效的
	}
	w.Items = append(w.Items, item)
	return item
}

// UID sets UID
func (i *Item) UID(uid string) *Item {
	i.UID = uid
	return i
}

// Subtitle sets Subtitle
func (i *Item) Subtitle(subtitle string) *Item {
	i.Subtitle = subtitle
	return i
}

// Arg sets Arg
func (i *Item) Arg(arg string) *Item {
	i.Arg = arg
	return i
}

// Match sets Match
func (i *Item) Match(match string) *Item {
	i.Match = match
	return i
}

// Valid sets Valid
func (i *Item) Valid(valid bool) *Item {
	i.Valid = valid
	return i
}

// Cmd sets Cmd Mod
func (i *Item) Cmd(arg, subtitle string) *Item {
	if i.Mods == nil {
		i.Mods = &Mods{}
	}
	i.Mods.Cmd = &Mod{Arg: arg, Subtitle: subtitle, Valid: true}
	return i
}

// Alt sets Alt Mod
func (i *Item) Alt(arg, subtitle string) *Item {
	if i.Mods == nil {
		i.Mods = &Mods{}
	}
	i.Mods.Alt = &Mod{Arg: arg, Subtitle: subtitle, Valid: true}
	return i
}

// SendFeedback 将 workflow 的内容以 JSON 格式输出到标准输出
func (w *Workflow) SendFeedback() {
	// 如果没有 items，添加一个默认的 "未找到" 消息
	// 但要小心，某些情况下空列表是预期的
	if len(w.Items) == 0 {
		w.NewItem("未找到结果").Valid(false)
	}
	
	output, _ := json.Marshal(map[string][]*Item{"items": w.Items})
	os.Stdout.Write(output)
}

// Helpers
func ptr[T any](v T) *T {
	return &v
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseInt(s string, fallback int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return i
}
