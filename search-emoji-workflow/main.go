package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Emoji struct {
	Char     string `json:"char"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Item struct {
	Title    string            `json:"title"`
	Subtitle string            `json:"subtitle"`
	Arg      string            `json:"arg"`
	Icon     map[string]string `json:"icon"`
}

func main() {
	// Workflow 根目录
	baseDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	// 读取 emoji.json
	data, err := ioutil.ReadFile(filepath.Join(baseDir, "emoji.json"))
	if err != nil {
		fmt.Printf(`{"items":[{"title":"错误","subtitle":"无法读取 emoji.json","valid":false}]}`)
		return
	}

	var emojis []Emoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		fmt.Printf(`{"items":[{"title":"错误","subtitle":"emoji.json 解析失败","valid":false}]}`)
		return
	}

	// 获取关键词（来自 Alfred）
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	items := []Item{}
	for _, e := range emojis {
		if query != "" &&
			!strings.Contains(strings.ToLower(e.Name), query) &&
			!strings.Contains(strings.ToLower(e.Category), query) &&
			!strings.Contains(strings.ToLower(e.Char), query) {
			continue
		}

		items = append(items, Item{
			Title:    e.Char,     // Grid View 下，大标题显示 emoji
			Subtitle: e.Name,     // 辅助文字
			Arg:      e.Char,     // 传递给后续节点的参数
			Icon:     map[string]string{"path": "icon.png"}, // 占位图标
		})
	}

	if len(items) == 0 {
		items = append(items, Item{
			Title:    "未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	output, _ := json.Marshal(map[string][]Item{"items": items})
	fmt.Println(string(output))
}
