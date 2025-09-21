package main

import (
	"fmt"
	"os"
	"os/exec"
)

// 触发 Alfred External Trigger
func triggerAlfred(triggerID string) {
	bundleID := os.Getenv("alfred_workflow_bundleid")
	if bundleID == "" {
		return
	}
	script := fmt.Sprintf(`tell application "Alfred 5" to run trigger "%s" in workflow "%s"`, triggerID, bundleID)
	exec.Command("osascript", "-e", script).Run()
}

// cachectl: clear:xxx 或 refresh:xxx
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
	// ----------------- CLEAR -----------------
	case "clear":
		switch key {
		case "stars", "repos", "gists":
			HandleClear(key)
			info := cacheInfo(db, key)
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
			infoStars := cacheInfo(db, "stars")
			infoRepos := cacheInfo(db, "repos")
			infoGists := cacheInfo(db, "gists")
			summary := fmt.Sprintf("Stars=%s | Repos=%s | Gists=%s", infoStars, infoRepos, infoGists)
			return []AlfredItem{{
				Title:    "🧹 已清除所有缓存",
				Subtitle: summary,
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": summary,
				},
			}}
		}

	// ----------------- REFRESH -----------------
	case "refresh":
		switch key {
		case "stars":
			if fresh, err := fetchStars(); err == nil {
				saveRepos(db, fresh, "stars")
				triggerAlfred("stars.refresh")
				info := cacheInfo(db, "stars")
				return []AlfredItem{{
					Title:    "♻ Stars 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}}
			} else {
				return []AlfredItem{{
					Title: "⚠️ Stars 刷新失败: " + err.Error(),
					Valid: false,
				}}
			}

		case "repos":
			if fresh, err := fetchRepos(); err == nil {
				saveRepos(db, fresh, "repos")
				triggerAlfred("repos.refresh")
				info := cacheInfo(db, "repos")
				return []AlfredItem{{
					Title:    "♻ Repos 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}}
			} else {
				return []AlfredItem{{
					Title: "⚠️ Repos 刷新失败: " + err.Error(),
					Valid: false,
				}}
			}

		case "gists":
			if fresh, err := fetchGists(); err == nil {
				saveGists(db, fresh)
				triggerAlfred("gists.refresh")
				info := cacheInfo(db, "gists")
				return []AlfredItem{{
					Title:    "♻ Gists 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}}
			} else {
				return []AlfredItem{{
					Title: "⚠️ Gists 刷新失败: " + err.Error(),
					Valid: false,
				}}
			}

		case "all":
			// all 要分别刷新三类
			if stars, err := fetchStars(); err == nil {
				saveRepos(db, stars, "stars")
				triggerAlfred("stars.refresh")
			}
			if repos, err := fetchRepos(); err == nil {
				saveRepos(db, repos, "repos")
				triggerAlfred("repos.refresh")
			}
			if gists, err := fetchGists(); err == nil {
				saveGists(db, gists)
				triggerAlfred("gists.refresh")
			}
			infoStars := cacheInfo(db, "stars")
			infoRepos := cacheInfo(db, "repos")
			infoGists := cacheInfo(db, "gists")
			summary := fmt.Sprintf("Stars=%s | Repos=%s | Gists=%s", infoStars, infoRepos, infoGists)
			return []AlfredItem{{
				Title:    "♻ 所有缓存已刷新",
				Subtitle: summary,
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": summary,
				},
			}}
		}
	}

	return []AlfredItem{{
		Title: "未知命令: " + action,
		Valid: false,
	}}
}

// 小工具：找冒号
func indexColon(s string) int {
	for i, c := range s {
		if c == ':' {
			return i
		}
	}
	return -1
}
