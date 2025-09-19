package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func HandleClearCache() {
	db := initDB()
	items := []AlfredItem{}

	// 全部清除
	items = append(items, AlfredItem{
		Title:    "🧹 清除所有缓存",
		Subtitle: "清除 stars / repos / gists 全部缓存",
		Arg:      "clear:all",
		Valid:    true,
	})

	// Stars
	items = append(items, AlfredItem{
		Title:    "🧹 清除 Stars 缓存",
		Subtitle: "当前缓存: " + getMeta(db, "last_stars"),
		Arg:      "clear:stars",
		Valid:    true,
	})

	// Repos
	items = append(items, AlfredItem{
		Title:    "🧹 清除 Repos 缓存",
		Subtitle: "当前缓存: " + getMeta(db, "last_repos"),
		Arg:      "clear:repos",
		Valid:    true,
	})

	// Gists
	items = append(items, AlfredItem{
		Title:    "🧹 清除 Gists 缓存",
		Subtitle: "当前缓存: " + getMeta(db, "last_gists"),
		Arg:      "clear:gists",
		Valid:    true,
	})

	out := map[string]interface{}{"items": items}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(out)
}
