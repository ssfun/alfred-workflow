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

type Item struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Icon     struct {
		Path string `json:"path"`
	} `json:"icon"`
}

func main() {
	// workflow 目录下的资源
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

	// 获取用户 query
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	var results []Item
	for _, e := range emojis {
		text := strings.ToLower(e.Name + " " + e.Category + " " + e.Group + " " + e.Char)
		if query == "" || strings.Contains(text, query) {
			item := Item{
				Title:    e.Char,           // Grid view 下方小字
				Subtitle: e.Name,           // 辅助信息
				Arg:      e.Char,           // 返回给 workflow 的 emoji
			}
			// icon 文件路径（例：icons/1f600.png）
			code := strings.ToLower(strings.ReplaceAll(e.Codes, " ", "-"))
			item.Icon.Path = filepath.Join(iconDir, code+".png")

			results = append(results, item)
		}
	}

	if len(results) == 0 {
		results = append(results, Item{
			Title:    "未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	output, _ := json.Marshal(map[string]interface{}{"items": results})
	fmt.Println(string(output))
}
