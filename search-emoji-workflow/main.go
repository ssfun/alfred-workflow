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
	Icon     Icon   `json:"icon"`
}

func main() {
	// workflow 路径
	baseDir, _ := os.Getwd()
	dataFile := filepath.Join(baseDir, "emoji.json")
	iconDir := filepath.Join(baseDir, "icons")

	// 读取 emoji.json
	data, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading emoji.json: %v\n", err)
		os.Exit(1)
	}

	var emojis []Emoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing emoji.json: %v\n", err)
		os.Exit(1)
	}

	// 获取用户输入 query
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	var results []Item
	for _, e := range emojis {
		emojiChar := e.Char
		text := strings.ToLower(e.Name + " " + e.Category + " " + e.Group + " " + e.Subgroup + " " + e.Char)

		// 分类过滤 :xxx
		if strings.HasPrefix(query, ":") {
			category := strings.TrimPrefix(query, ":")
			if !strings.Contains(strings.ToLower(e.Category), category) {
				continue
			}
		} else {
			// 普通搜索
			if query != "" && !strings.Contains(text, query) {
				continue
			}
		}

		// 转换 Code 为 PNG 文件路径 (1F600 -> 1f600.png)
		code := strings.ToLower(strings.ReplaceAll(e.Codes, " ", "-"))
		iconPath := filepath.Join(iconDir, code+".png")

		item := Item{
			Title:    emojiChar,                                  // Grid 下方小字：emoji 本身
			Subtitle: fmt.Sprintf("%s | %s", e.Name, e.Category), // 搜索字段：名字 + 类别
			Arg:      emojiChar,                                  // 返回复制的字符
			Icon:     Icon{Path: iconPath},
		}

		results = append(results, item)
	}

	// 如果没有结果
	if len(results) == 0 {
		results = append(results, Item{
			Title:    "❌ 未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	// 输出 Alfred JSON
	output, _ := json.Marshal(map[string]interface{}{"items": results})
	fmt.Println(string(output))
}
