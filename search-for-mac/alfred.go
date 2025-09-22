package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

type AlfredItem struct {
	Uid      string `json:"uid"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Valid    bool   `json:"valid"`
	Icon     struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"icon"`
}

// 文件大小格式化
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(1024), 0
	for n := size / 1024; n >= 1024 && exp < 2; n /= 1024 {
		div *= 1024
		exp++
	}
	value := float64(size) / float64(div)
	switch exp {
	case 0:
		return fmt.Sprintf("%.1fKB", value)
	case 1:
		return fmt.Sprintf("%.1fMB", value)
	case 2:
		return fmt.Sprintf("%.1fGB", value)
	}
	return fmt.Sprintf("%.1fTB", float64(size)/float64(1024*1024*1024*1024))
}

// 构建 Alfred 输出 JSON
func BuildAlfredOutput(results []Result, maxRes int) string {
	items := []AlfredItem{}
	if len(results) == 0 {
		item := AlfredItem{
			Title:    "没有找到匹配结果",
			Subtitle: "请尝试调整关键词或目录设置",
			Valid:    false,
		}
		item.Icon.Type = "icon"
		item.Icon.Path = "icon.png"
		items = append(items, item)
	} else {
		for _, r := range results {
			item := AlfredItem{Uid: r.Path, Title: r.Name, Arg: r.Path, Valid: true}
			parent := filepath.Dir(r.Path)
			if r.IsDir {
				item.Subtitle = fmt.Sprintf("%s", parent)
			} else {
				item.Subtitle = fmt.Sprintf("%s | %s | 修改: %s",
					parent, formatSize(r.Size), r.ModTime.Format("2006-01-02 15:04"))
			}
			item.Icon.Type = "fileicon"
			item.Icon.Path = r.Path
			items = append(items, item)
		}
	}
	data, _ := json.Marshal(map[string]interface{}{"items": items})
	return string(data)
}
