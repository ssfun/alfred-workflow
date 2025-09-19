// main.go
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		wf := NewWorkflow()
		wf.NewItem("无效命令").
			SetSubtitle("请提供一个子命令, 例如: stars, repos, gists, search-repo").
			SetValid(false)
		wf.SendFeedback()
		return
	}

	// 解析命令行参数
	// os.Args[0] 是程序名, os.Args[1] 是子命令
	cmd := os.Args[1]
	query := ""
	if len(os.Args) > 2 {
		query = os.Args[2]
	}

	// 根据子命令路由到不同的处理器
	switch {
	// 缓存控制命令，例如 refresh:stars
	case strings.HasPrefix(cmd, "refresh:"):
		handleCacheCtl(cmd)
	case cmd == "stars":
		handleStars(query)
	case cmd == "repos":
		handleRepos(query)
	case cmd == "gists":
		handleGists(query)
	case cmd == "search-repo":
		handleSearchRepo(query)
	default:
		wf := NewWorkflow()
		wf.NewItem(fmt.Sprintf("未知命令: %s", cmd)).SetValid(false)
		wf.SendFeedback()
	}
}

