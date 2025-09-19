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

func HandleCacheCtl(action string) string {
	if action == "" {
		return "用法: cachectl [clear|refresh]:[stars|repos|gists|all]"
	}
	var act, key string
	if i := indexColon(action); i > -1 {
		act, key = action[:i], action[i+1:]
	} else {
		act = action
	}

	switch act {
	case "clear":
		switch key {
		case "stars":
			HandleClear("stars")
		case "repos":
			HandleClear("repos")
		case "gists":
			HandleClear("gists")
		case "all":
			HandleClear("all")
		}
		return "✅ 清除完成"
	case "refresh":
		switch key {
		case "stars":
			HandleClear("stars")
			triggerAlfred("stars.refresh")
		case "repos":
			HandleClear("repos")
			triggerAlfred("repos.refresh")
		case "gists":
			HandleClear("gists")
			triggerAlfred("gists.refresh")
		case "all":
			HandleClear("all")
			for _, trig := range []string{"stars.refresh", "repos.refresh", "gists.refresh"} {
				triggerAlfred(trig)
			}
		}
		return "✅ 刷新完成"
	}
	return "未知命令: " + action
}

func indexColon(s string) int {
	for i, c := range s {
		if c == ':' {
			return i
		}
	}
	return -1
}
