package main

import (
	"encoding/json"
	"os"
)

func HandleClearCache() {
	db := initDB()
	items := []AlfredItem{}

	// 全部清除
	items = append(items, AlfredItem{
		Title:    "清除所有缓存",
		Subtitle: "清除 stars / repos / gists 全部缓存",
		Arg:      "clear:all",
		Valid:    true,
	})

	// Stars 缓存
	items = append(items, AlfredItem{
		Title:    "清除 Stars 缓存",
		Subtitle: cacheInfo(db, "stars"),
		Arg:      "clear:stars",
		Valid:    true,
	})

	// Repos 缓存
	items = append(items, AlfredItem{
		Title:    "清除 Repos 缓存",
		Subtitle: cacheInfo(db, "repos"),
		Arg:      "clear:repos",
		Valid:    true,
	})

	// Gists 缓存
	items = append(items, AlfredItem{
		Title:    "清除 Gists 缓存",
		Subtitle: cacheInfo(db, "gists"),
		Arg:      "clear:gists",
		Valid:    true,
	})

	// 📂 打开缓存目录
	cacheDir := getCacheDir()
	items = append(items, AlfredItem{
		Title:    "打开缓存目录",
		Subtitle: cacheDir,
		Arg:      "open:"+cacheDir,  // 用 open: 前缀标记
		Valid:    true,
	})

	// 输出 JSON
	out := map[string]interface{}{"items": items}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(out)
}
