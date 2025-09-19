package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`{"items":[]}`)
		return
	}

	cmd := os.Args[1]
	query := ""
	if len(os.Args) > 2 {
		query = os.Args[2]
	}

	var items []AlfredItem

	// --- 特殊命令 Clear & Refresh & Reload ---
	if strings.HasPrefix(cmd, "clear:") {
		t := strings.TrimPrefix(cmd, "clear:")
		msg := HandleClear(t)
		items = []AlfredItem{{Title: msg, Valid: false}}

	} else if strings.HasPrefix(cmd, "refresh:") {
		t := strings.TrimPrefix(cmd, "refresh:")
		items = HandleRefresh(t)

	} else if strings.HasPrefix(cmd, "reload:") {
		t := strings.TrimPrefix(cmd, "reload:")
		switch t {
		case "repos":
			items = HandleRepos("")
		case "stars":
			items = HandleStars("")
		case "gists":
			items = HandleGists("")
		}

	} else {
		// --- 子命令 ---
		switch cmd {
		case "stars":
			items = HandleStars(query)
		case "repos":
			items = HandleRepos(query)
		case "gists":
			items = HandleGists(query)
		default:
			items = []AlfredItem{{
				Title:    "未知命令",
				Subtitle: cmd,
				Valid:    false,
			}}
		}
	}

	out := map[string]interface{}{"items": items}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(out)
}
