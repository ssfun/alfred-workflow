package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Emoji struct {
	Codes    string `json:"codes"`
	Char     string `json:"char"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Group    string `json:"group"`
	Subgroup string `json:"subgroup"`
}

type Icon struct {
	Path string `json:"path"`
}

type Item struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Match    string `json:"match,omitempty"`
	Icon     Icon   `json:"icon"`
}

func main() {
	// 找到 workflow 下的资源目录
	baseDir, _ := os.Getwd()
	dataFile := filepath.Join(baseDir, "emoji.json")
	iconDir := filepath.Join(baseDir, "icons")

	// 读取 emoji.json
	data, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading emoji.json: %v\n", err)
		os.Exit(1)
	}

	// JSON 反序列化
	var emojis []Emoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing emoji.json: %v\n", err)
		os.Exit(1)
	}

	// 读取用户输入
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	var results []Item
	for _, e := range emojis {
		emojiChar := e.Char
		// 搜索提示串
		searchText := strings.ToLower(e.Name + " " + e.Category + " " + e.Group + " " + e.Subgroup + " " + e.Char)

		// 分类过滤 :xxx
		if strings.HasPrefix(query, ":") {
			category := strings.TrimPrefix(query, ":")
			if !strings.Contains(strings.ToLower(e.Category), category) {
				continue
			}
		} else {
			// 普通搜索过滤
			if query != "" && !strings.Contains(searchText, query) {
				continue
			}
		}

		// 确定 PNG 文件路径（例：1F600 -> 1f600.png）
		code := strings.ToLower(strings.ReplaceAll(e.Codes, " ", "-"))
		iconPath := filepath.Join(iconDir, code+".png")

		results = append(results, Item{
			Title:    emojiChar,                                  // Grid 展示大 emoji
			Subtitle: fmt.Sprintf("%s | %s", e.Name, e.Category), // 底部说明
			Arg:      emojiChar,                                  // 返回表情
			Match:    searchText,                                 // ✅ 用于搜索
			Icon:     Icon{Path: iconPath},
		})
	}

	// 如果没有结果
	if len(results) == 0 {
		results = append(results, Item{
			Title:    "❌ 未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	// 输出 JSON （符合 Alfred Script Filter 格式）
	output, _ := json.Marshal(map[string]interface{}{"items": results})
	fmt.Println(string(output))
}
