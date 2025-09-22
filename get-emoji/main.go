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

const maxRecent = 8

// --- recent.json 管理 ---
func getRecentFile(baseDir string) string {
	return filepath.Join(baseDir, "recent.json")
}

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

func saveRecent(file string, recent []string) {
	if len(recent) > maxRecent {
		recent = recent[:maxRecent]
	}
	data, _ := json.MarshalIndent(recent, "", "  ")
	_ = os.WriteFile(file, data, 0644)
}

// --- 查询逻辑 ---
func queryEmoji(baseDir, query string) {
	dataFile := filepath.Join(baseDir, "emoji.json")
	iconDir := filepath.Join(baseDir, "icons")
	recentFile := getRecentFile(baseDir)

	data, err := os.ReadFile(dataFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 emoji.json 出错: %v\n", err)
		os.Exit(1)
	}
	var emojis []Emoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		fmt.Fprintf(os.Stderr, "解析 emoji.json 出错: %v\n", err)
		os.Exit(1)
	}

	recent := loadRecent(recentFile)
	var results []Item

	// 最近使用
	if query == "" {
		for _, rc := range recent {
			code := fmt.Sprintf("%x", []rune(rc)[0])
			localName := code + ".png"
			iconPath := filepath.Join(iconDir, localName)
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

	// emoji 总表
	for _, e := range emojis {
		emojiChar := e.Char
		searchText := strings.ToLower(
			e.Name + " " + e.Category + " " + e.Group + " " + e.Subgroup + " " + e.Char,
		)

		if strings.HasPrefix(query, ":") {
			category := strings.TrimPrefix(query, ":")
			if !strings.Contains(strings.ToLower(e.Category), category) {
				continue
			}
		} else if query != "" && !strings.Contains(searchText, query) {
			continue
		}

		localName := codesToLocalFilename(e.Codes) // ✅ 从 utils.go 调用
		iconPath := filepath.Join(iconDir, localName)
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

	if len(results) == 0 {
		results = append(results, Item{
			Title:    "❌ 未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	output, _ := json.Marshal(map[string]interface{}{"items": results})
	fmt.Println(string(output))
}

// --- Main ---
func main() {
	baseDir, _ := os.Getwd()
	recentFile := getRecentFile(baseDir)

	// 更新最近使用
	if len(os.Args) > 2 && os.Args[1] == "--recent" {
		emojiChar := os.Args[2]
		recent := loadRecent(recentFile)
		newRecent := []string{emojiChar}
		for _, r := range recent {
			if r != emojiChar {
				newRecent = append(newRecent, r)
			}
		}
		saveRecent(recentFile, newRecent)
		return
	}

	// 工具模式
	if len(os.Args) > 1 && os.Args[1] == "utils" {
		runUtils(baseDir, os.Args[2:])
		return
	}

	// 查询模式
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}
	queryEmoji(baseDir, query)
}
