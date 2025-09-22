package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// === Utility ===
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

// === 命名规则转换 ===
// 本地文件名规则 eg: 1f469-1f3fc-200d-1f9b2.png
func codesToLocalFilename(codes string) string {
	parts := strings.Split(codes, " ")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "-") + ".png"
}

// Noto Emoji 官方远程命名 eg: emoji_u1f469_1f3fc_200d_1f9b2.png
func codesToNotoFilename(codes string) string {
	parts := strings.Split(codes, " ")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return "emoji_u" + strings.Join(parts, "_") + ".png"
}

// 下载文件
func downloadFile(url, targetPath string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(targetPath); err == nil {
			return nil // 已存在，且无需覆盖
		}
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// === 命令实现 ===
// 下载 PNG icon
func downloadIcons(baseDir string, mode string) {
	dataFile := filepath.Join(baseDir, "emoji.json")
	iconDir := filepath.Join(baseDir, "icons")

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

	overwrite := (mode == "overwrite")
	count := 0
	for _, e := range emojis {
		localName := codesToLocalFilename(e.Codes)
		notoName := codesToNotoFilename(e.Codes)
		target := filepath.Join(iconDir, localName)
		url := fmt.Sprintf("https://raw.githubusercontent.com/googlefonts/noto-emoji/main/png/128/%s", notoName)

		err := downloadFile(url, target, overwrite)
		if err == nil {
			count++
			fmt.Printf("✅ %s\n", localName)
		} else {
			fmt.Printf("⚠️  跳过 %s (%v)\n", localName, err)
		}
	}
	fmt.Printf("\n完成: 共下载 %d 个 emoji 图标\n", count)
}

// 更新 emoji.json
func updateEmojiJSON(baseDir string) {
	target := filepath.Join(baseDir, "emoji.json")
	url := "https://raw.githubusercontent.com/amio/emoji.json/master/emoji.json"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "下载 emoji.json 出错: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "下载失败: %s\n", resp.Status)
		os.Exit(1)
	}
	data, _ := io.ReadAll(resp.Body)
	_ = os.WriteFile(target, data, 0644)

	fmt.Printf("✅ 已更新 emoji.json (%d 字节)\n", len(data))
}

// === Script Filter 查询 ===
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

	// 最近使用（仅 query 为空时展示）
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

	for _, e := range emojis {
		emojiChar := e.Char
		searchText := strings.ToLower(e.Name + " " + e.Category + " " + e.Group + " " + e.Subgroup + " " + e.Char)

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

		localName := codesToLocalFilename(e.Codes)
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

// === main ===
func main() {
	baseDir, _ := os.Getwd()
	recentFile := getRecentFile(baseDir)

	if len(os.Args) > 2 && os.Args[1] == "--download" {
		downloadIcons(baseDir, os.Args[2])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--update-json" {
		updateEmojiJSON(baseDir)
		return
	}
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

	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}
	queryEmoji(baseDir, query)
}
