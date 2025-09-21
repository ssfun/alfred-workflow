package main

import (
	"fmt"
	"os"
	"os/exec"
)

func triggerAlfred(triggerID string) {
	bundleID := os.Getenv("alfred_workflow_bundleid")
	if bundleID == "" {
		return
	}
	script := fmt.Sprintf(`tell application "Alfred 5" to run trigger "%s" in workflow "%s"`, triggerID, bundleID)
	exec.Command("osascript", "-e", script).Run()
}

// HandleCacheCtl 动作：clear:xxx 或 refresh:xxx
func HandleCacheCtl(action string) []AlfredItem {
	if action == "" {
		return []AlfredItem{{
			Title: "用法: cachectl [clear|refresh]:[stars|repos|gists|all]",
			Valid: false,
		}}
	}

	var act, key string
	if i := indexColon(action); i > -1 {
		act, key = action[:i], action[i+1:]
	} else {
		act = action
	}

	db := initDB()

	switch act {
	case "clear":
		switch key {
		case "stars", "repos", "gists":
			HandleClear(key)
			info := cacheInfo(db, "key")
			return []AlfredItem{{
				Title:    fmt.Sprintf("🧹 已清除 %s 缓存", key),
				Subtitle: info,
				Valid:    false,
				Variables: map[string]string{
						"querysubtitle": info,
					},
			}}
		case "all":
			HandleClear("all")
			info := fmt.Sprintf("Stars=%s | Repos=%s | Gists=%s",
					cacheInfo(db, "stars"), cacheInfo(db, "repos"), cacheInfo(db, "gists"))
			return []AlfredItem{{
				Title:    "🧹 已清除所有缓存",
				Subtitle: info,
				Valid: false,
				Variables: map[string]string{
						"querysubtitle": info,
					},
			}}
		default:
			return []AlfredItem{{Title: "未知类型: " + key, Valid: false}}
		}

	case "refresh":
		switch key {
		case "stars":
			HandleClear("stars")
			triggerAlfred("stars.refresh")
			return []AlfredItem{{
				Title:    "♻ 刷新 Stars 缓存",
				Subtitle: cacheInfo(db, "stars"),
				Valid:    false,
			}}
		case "repos":
			HandleClear("repos")
			triggerAlfred("repos.refresh")
			return []AlfredItem{{
				Title:    "♻ 刷新 Repos 缓存",
				Subtitle: cacheInfo(db, "repos"),
				Valid:    false,
			}}
		case "gists":
			HandleClear("gists")
			triggerAlfred("gists.refresh")
			return []AlfredItem{{
				Title:    "♻ 刷新 Gists 缓存",
				Subtitle: cacheInfo(db, "gists"),
				Valid:    false,
			}}
		case "all":
			HandleClear("all")
			for _, trig := range []string{"stars.refresh", "repos.refresh", "gists.refresh"} {
				triggerAlfred(trig)
			}
			return []AlfredItem{{
				Title:    "♻ 刷新所有缓存",
				Subtitle: fmt.Sprintf("Stars=%s | Repos=%s | Gists=%s",
					cacheInfo(db, "stars"), cacheInfo(db, "repos"), cacheInfo(db, "gists")),
				Valid: false,
			}}
		default:
			return []AlfredItem{{Title: "未知类型: " + key, Valid: false}}
		}
	}

	return []AlfredItem{{Title: "未知命令: " + action, Valid: false}}
}

func indexColon(s string) int {
	for i, c := range s {
		if c == ':' {
			return i
		}
	}
	return -1
}
