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

// --- 下载 PNG 文件 ---
func downloadFile(url, targetPath string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(targetPath); err == nil {
			return nil // 已存在且无需覆盖
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

// --- 转换函数 ---
// 本地命名规则 (1f469-1f3fc-200d-1f9b2.png)
func codesToLocalFilename(codes string) string {
	parts := strings.Split(codes, " ")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "-") + ".png"
}

// Noto Emoji 命名规则 (emoji_u1f469_1f3fc_200d_1f9b2.png)
func codesToNotoFilename(codes string) string {
	parts := strings.Split(codes, " ")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return "emoji_u" + strings.Join(parts, "_") + ".png"
}

// --- 下载 emoji PNG 图标 ---
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
		localName := codesToLocalFilename(e.Codes) // 本地命名
		notoName := codesToNotoFilename(e.Codes)   // 远端命名

		target := filepath.Join(iconDir, localName)
		url := fmt.Sprintf(
			"https://raw.githubusercontent.com/googlefonts/noto-emoji/main/png/128/%s",
			notoName,
		)

		err := downloadFile(url, target, overwrite)
		if err == nil {
			count++
			fmt.Printf("✅ %s\n", localName)
		} else {
			fmt.Printf("⚠️ 跳过 %s (%v)\n", localName, err)
		}
	}
	fmt.Printf("\n完成: 共下载 %d 个 emoji 图标\n", count)
}

// --- 更新 emoji.json ---
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

// --- 清除最近记录 ---
func clearRecent(baseDir string) {
	recentFile := getRecentFile(baseDir)
	_ = os.WriteFile(recentFile, []byte("[]"), 0644)
	fmt.Println("✅ 已清除最近使用记录")
}

// --- utils 命令入口 ---
func runUtils(baseDir string, args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "update-json":
			updateEmojiJSON(baseDir)
			return
		case "download-skip":
			downloadIcons(baseDir, "skip")
			return
		case "download-overwrite":
			downloadIcons(baseDir, "overwrite")
			return
		case "clear-recent":
			clearRecent(baseDir)
			return
		}
	}

	results := []Item{}

	// 最近更新时间
	jsonFile := filepath.Join(baseDir, "emoji.json")
	info, err := os.Stat(jsonFile)
	lastUpdate := "未知"
	if err == nil {
		lastUpdate = info.ModTime().Format("2006-01-02 15:04:05")
	}

	results = append(results, Item{
		Title:    "⬇️ 更新 emoji.json",
		Subtitle: "最近更新: " + lastUpdate,
		Arg:      "update-json",
	})
	results = append(results, Item{
		Title:    "⬇️ 更新 Emoji 资源 (增补缺失)",
		Subtitle: "只下载缺失的 PNG",
		Arg:      "download-skip",
	})
	results = append(results, Item{
		Title:    "♻️ 更新 Emoji 资源 (覆盖全部)",
		Subtitle: "强制覆盖所有 PNG",
		Arg:      "download-overwrite",
	})
	results = append(results, Item{
		Title:    "🗑️ 清除最近使用记录",
		Subtitle: "清空 recent.json",
		Arg:      "clear-recent",
	})

	output, _ := json.Marshal(map[string]interface{}{"items": results})
	fmt.Println(string(output))
}
