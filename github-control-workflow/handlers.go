// handlers.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v39/github"
)

// handleStars 处理 'stars' 子命令
func handleStars(query string) {
	wf := NewWorkflow()

	// 如果没有查询词，显示默认菜单项
	if query == "" {
		wf.NewItem("🌐 打开 GitHub Stars 页面").
			SetSubtitle(fmt.Sprintf("https://github.com/%s?tab=stars", githubUser)).
			SetArg(fmt.Sprintf("https://github.com/%s?tab=stars", githubUser))
		wf.NewItem("♻ 刷新 Stars 缓存").
			SetSubtitle(getCacheInfo("stars")).
			SetArg("refresh:stars")
		wf.SendFeedback()
		return
	}

	// 1. 从缓存查询
	repos, err := queryRepos("stars", query, maxStars)
	if err != nil {
		wf.NewItem("查询缓存失败").SetSubtitle(err.Error()).SetValid(false)
		wf.SendFeedback()
		return
	}

	// 2. 如果缓存为空，则从 API 获取
	if len(repos) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fresh, fetchErr := fetchStars(ctx)
		if fetchErr != nil {
			wf.NewItem("从 GitHub API 获取失败").SetSubtitle(fetchErr.Error()).SetValid(false)
			wf.SendFeedback()
			return
		}
		// 保存到缓存
		if err := saveRepos(fresh, "stars"); err != nil {
			wf.NewItem("保存缓存失败").SetSubtitle(err.Error()).SetValid(false)
		}
		repos = fresh // 使用新获取的数据
	}

	// 3. 将结果转换为 Alfred Items
	for _, r := range repos {
		wf.NewItem(r.GetFullName()).
			SetSubtitle(formatSubtitle(r.GetStargazersCount(), r.GetUpdatedAt().Time, r.GetDescription())).
			SetArg(r.GetHTMLURL()).
			SetMatch(normalize(r.GetFullName()+" "+r.GetDescription())).
			SetCmdModifier(r.GetCloneURL(), "复制 Clone URL").
			SetAltModifier(r.GetHTMLURL(), "复制 Repo URL")
	}

	if len(repos) == 0 {
		wf.NewItem(fmt.Sprintf("✖ 未找到匹配: %s", query)).SetValid(false)
	}

	wf.SendFeedback()
}

// handleRepos 处理 'repos' 子命令
func handleRepos(query string) {
	wf := NewWorkflow()

	if query == "" {
		wf.NewItem("✪ 打开 Repos 页面").
			SetSubtitle(fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser)).
			SetArg(fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser))
		wf.NewItem("♻ 刷新 Repos 缓存").
			SetSubtitle(getCacheInfo("repos")).
			SetArg("refresh:repos")
		wf.SendFeedback()
		return
	}

	repos, err := queryRepos("repos", query, maxRepos)
	if err != nil {
		wf.NewItem("查询缓存失败").SetSubtitle(err.Error()).SetValid(false)
		wf.SendFeedback()
		return
	}

	if len(repos) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fresh, fetchErr := fetchRepos(ctx)
		if fetchErr != nil {
			wf.NewItem("从 GitHub API 获取失败").SetSubtitle(fetchErr.Error()).SetValid(false)
			wf.SendFeedback()
			return
		}
		if err := saveRepos(fresh, "repos"); err != nil {
			wf.NewItem("保存缓存失败").SetSubtitle(err.Error()).SetValid(false)
		}
		repos = fresh
	}

	for _, r := range repos {
		title := r.GetFullName()
		if r.GetPrivate() {
			title += " 🔒"
		}
		wf.NewItem(title).
			SetSubtitle(formatSubtitle(r.GetStargazersCount(), r.GetUpdatedAt().Time, r.GetDescription())).
			SetArg(r.GetHTMLURL()).
			SetMatch(normalize(r.GetFullName()+" "+r.GetDescription())).
			SetCmdModifier(r.GetCloneURL(), "复制 Clone URL").
			SetAltModifier(r.GetHTMLURL(), "复制 Repo URL")
	}

	if len(repos) == 0 {
		wf.NewItem(fmt.Sprintf("✖ 未找到匹配: %s", query)).SetValid(false)
	}

	wf.SendFeedback()
}

// handleGists 处理 'gists' 子命令
func handleGists(query string) {
	wf := NewWorkflow()

	if query == "" {
		wf.NewItem("✪ 打开 Gists 页面").
			SetSubtitle(fmt.Sprintf("https://gist.github.com/%s", githubUser)).
			SetArg(fmt.Sprintf("https://gist.github.com/%s", githubUser))
		wf.NewItem("♻ 刷新 Gists 缓存").
			SetSubtitle(getCacheInfo("gists")).
			SetArg("refresh:gists")
		wf.SendFeedback()
		return
	}

	gists, err := queryGists(query, maxGists)
	if err != nil {
		wf.NewItem("查询 Gist 缓存失败").SetSubtitle(err.Error()).SetValid(false)
		wf.SendFeedback()
		return
	}

	if len(gists) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fresh, fetchErr := fetchGists(ctx)
		if fetchErr != nil {
			wf.NewItem("获取 Gists 失败").SetSubtitle(fetchErr.Error()).SetValid(false)
			wf.SendFeedback()
			return
		}
		if err := saveGists(fresh); err != nil {
			wf.NewItem("保存 Gist 缓存失败").SetSubtitle(err.Error()).SetValid(false)
		}
		gists = fresh
	}

	for _, g := range gists {
		title := g.GetDescription()
		if title == "" {
			title = "无描述的 Gist"
		}
		if !g.GetPublic() {
			title += " 🔒"
		}

		var filenames []string
		for filename := range g.Files {
			filenames = append(filenames, string(filename))
		}

		subtitle := fmt.Sprintf("%d 个文件: %s | 更新于 %s", len(filenames), time.Time(g.GetUpdatedAt()).Format("2006-01-02"))

		wf.NewItem(title).
			SetSubtitle(subtitle).
			SetArg(g.GetHTMLURL()).
			SetCmdModifier(g.GetID(), "复制 Gist ID").
			SetAltModifier(g.GetHTMLURL(), "复制 Gist URL")
	}

	if len(gists) == 0 {
		wf.NewItem(fmt.Sprintf("✖ 未找到匹配 Gist: %s", query)).SetValid(false)
	}

	wf.SendFeedback()
}

// handleCacheCtl 处理缓存控制命令，如 'refresh:stars'
func handleCacheCtl(arg string) {
	wf := NewWorkflow()
	parts := strings.Split(arg, ":")
	if len(parts) != 2 {
		return // 无效参数，不做任何事
	}
	action, cacheType := parts[0], parts[1]

	if action == "refresh" {
		if err := clearCache(cacheType); err != nil {
			wf.NewItem(fmt.Sprintf("清除 %s 缓存失败", cacheType)).SetSubtitle(err.Error()).SetValid(false)
			wf.SendFeedback()
			return
		}
		// 这里可以通过 osascript 触发 Alfred 的外部刷新，但为了简化，我们仅提示用户
		fmt.Printf("缓存 '%s' 已清空，请重新运行命令以刷新。", cacheType)
	}
}
