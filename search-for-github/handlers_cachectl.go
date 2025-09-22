package main

import (
	"encoding/json"
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

// 输出 Alfred Script Filter JSON
func writeAlfredItems(items []AlfredItem) {
	out := map[string]interface{}{"items": items}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(out)
}

// HandleCacheCtl 根据 clear/refresh 自动选择输出模式
func HandleCacheCtl(action string) {
	if action == "" {
		// clear 模式：直接输出环境变量
		fmt.Println("querysubtitle=用法: cachectl [clear|refresh]:[stars|repos|gists|all]")
		return
	}

	var act, key string
	if i := indexColon(action); i > -1 {
		act, key = action[:i], action[i+1:]
	} else {
		act = action
	}

	db := initDB()

	switch act {
	// ---------- CLEAR (Run Script → 输出变量格式) ----------
	case "clear":
		switch key {
		case "stars", "repos", "gists":
			HandleClear(key)
			info := cacheInfo(db, key)
			fmt.Printf("已清除 %s 缓存 · %s\n", key, info)
			return
		case "all":
			HandleClear("all")
			infoStars := cacheInfo(db, "stars")
			infoRepos := cacheInfo(db, "repos")
			infoGists := cacheInfo(db, "gists")
			summary := fmt.Sprintf("Stars=%s | Repos=%s | Gists=%s", infoStars, infoRepos, infoGists)
			fmt.Printf("已清除所有缓存 · %s\n", summary)
			return
		default:
			fmt.Printf("querysubtitle=未知类型: %s\n", key)
			return
		}

	// ---------- REFRESH (Script Filter → JSON) ----------
	case "refresh":
		switch key {
		case "stars":
			if fresh, err := fetchStars(); err == nil {
				saveRepos(db, fresh, "stars")
				triggerAlfred("stars.refresh")
				info := cacheInfo(db, "stars")
				writeAlfredItems([]AlfredItem{{
					Title:    "♻ Stars 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}})
				return
			}
		case "repos":
			if fresh, err := fetchRepos(); err == nil {
				saveRepos(db, fresh, "repos")
				triggerAlfred("repos.refresh")
				info := cacheInfo(db, "repos")
				writeAlfredItems([]AlfredItem{{
					Title:    "♻ Repos 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}})
				return
			}
		case "gists":
			if fresh, err := fetchGists(); err == nil {
				saveGists(db, fresh)
				triggerAlfred("gists.refresh")
				info := cacheInfo(db, "gists")
				writeAlfredItems([]AlfredItem{{
					Title:    "♻ Gists 缓存已刷新",
					Subtitle: info,
					Valid:    false,
					Variables: map[string]string{
						"querysubtitle": info,
					},
				}})
				return
			}
		case "all":
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
			writeAlfredItems([]AlfredItem{{
				Title:    "♻ 所有缓存已刷新",
				Subtitle: summary,
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": summary,
				},
			}})
			return
		}
	}

	// ---------- Unknown ----------
	fmt.Println("querysubtitle=未知命令: " + action)
}

// util：找冒号
func indexColon(s string) int {
	for i, c := range s {
		if c == ':' {
			return i
		}
	}
	return -1
}
