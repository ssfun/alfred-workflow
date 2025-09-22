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

const maxRecent = 30

func getRecentFile(baseDir string) string {
	return filepath.Join(baseDir, "recent.json")
}

// 读取最近使用
func loadRecent(file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return []string{}
	}
	return list
}

// 保存最近使用
func saveRecent(file string, recent []string) {
	if len(recent) > maxRecent {
		recent = recent[:maxRecent]
	}
	data, _ := json.MarshalIndent(recent, "", "  ")
	_ = os.WriteFile(file, data, 0644)
}

func main() {
	baseDir, _ := os.Getwd()
	dataFile := filepath.Join(baseDir, "emoji.json")
	iconDir := filepath.Join(baseDir, "icons")
	recentFile := getRecentFile(baseDir)

	// ====== 模式 1：更新最近使用 ======
	if len(os.Args) > 2 && os.Args[1] == "--recent" {
		emojiChar := os.Args[2]
		recent := loadRecent(recentFile)
		// 去重再置顶
		newRecent := []string{emojiChar}
		for _, r := range recent {
			if r != emojiChar {
				newRecent = append(newRecent, r)
			}
		}
		saveRecent(recentFile, newRecent)
		return
	}

	// ====== 模式 2：查询 =========
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

	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	// 读取最近使用
	recent := loadRecent(recentFile)

	var results []Item

	// 最近使用（仅 query 为空时展示）
	if query == "" {
		for _, rc := range recent {
			// 简单取第一个 rune 的 code
			code := fmt.Sprintf("%x", []rune(rc)[0])
			iconPath := filepath.Join(iconDir, code+".png")
			// ❌ 没有对应图标的过滤掉
			if _, err := os.Stat(iconPath); os.IsNotExist(err) {
				continue
			}
			results = append(results, Item{
				Title:    rc,
				Subtitle: "最近使用",
				Arg:      rc,
				Match:    "recent " + rc,
				Icon:     Icon{Path: iconPath},
			})
		}
	}

	// 遍历 emoji.json
	for _, e := range emojis {
		emojiChar := e.Char
		searchText := strings.ToLower(e.Name + " " + e.Category + " " + e.Group + " " + e.Subgroup + " " + e.Char)

		// 分类搜索
		if strings.HasPrefix(query, ":") {
			category := strings.TrimPrefix(query, ":")
			if !strings.Contains(strings.ToLower(e.Category), category) {
				continue
			}
		} else {
			if query != "" && !strings.Contains(searchText, query) {
				continue
			}
		}

		// 图标路径
		code := strings.ToLower(strings.ReplaceAll(e.Codes, " ", "-"))
		iconPath := filepath.Join(iconDir, code+".png")

		// ❌ 如果没有对应图标，跳过
		if _, err := os.Stat(iconPath); os.IsNotExist(err) {
			continue
		}

		results = append(results, Item{
			Title:    emojiChar,
			Subtitle: fmt.Sprintf("%s | %s", e.Name, e.Category),
			Arg:      emojiChar,
			Match:    searchText,
			Icon:     Icon{Path: iconPath},
		})
	}

	// 没有结果时提示
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
