package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// 初始化 Alfred Workflow 输出器
	wf := NewWorkflow()

	if len(os.Args) < 2 {
		wf.NewItem("无效命令").
			Subtitle("请提供一个子命令: stars, repos, gists, search-repos, cache-ctl").
			Valid(false)
		wf.SendFeedback()
		return
	}

	command := os.Args[1]
	query := ""
	if len(os.Args) > 2 {
		query = strings.Join(os.Args[2:], " ")
	}

	// 初始化数据库
	err := initDB()
	if err != nil {
		wf.NewItem("数据库初始化失败").
			Subtitle(err.Error()).
			Valid(false)
		wf.SendFeedback()
		return
	}
	defer closeDB()

	// 根据命令路由到不同的处理器
	var handlerErr error
	switch command {
	case "stars":
		handlerErr = handleStars(wf, query)
	case "repos":
		handlerErr = handleRepos(wf, query)
	case "gists":
		handlerErr = handleGists(wf, query)
	case "search-repos":
		handlerErr = handleSearchRepos(wf, query)
	case "cache-ctl":
		handlerErr = handleCacheCtl(wf, query)
	default:
		wf.NewItem(fmt.Sprintf("未知命令: %s", command)).Valid(false)
	}

	if handlerErr != nil {
		wf.NewItem("执行出错").
			Subtitle(handlerErr.Error()).
			Valid(false)
	}

	wf.SendFeedback()
}
