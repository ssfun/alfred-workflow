// handlers.go
package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
)

// handleStars processes the 'stars' subcommand.
func handleStars(query string) {
	wf := NewWorkflow()

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

	repos, err := queryRepos("stars", query, maxStars)
	if err != nil {
		wf.NewItem("查询缓存失败").SetSubtitle(err.Error()).SetValid(false)
		wf.SendFeedback()
		return
	}

	if len(repos) == 0 && query != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		fresh, fetchErr := fetchStars(ctx)
		if fetchErr != nil {
			wf.NewItem("从 GitHub API 获取失败").SetSubtitle(fetchErr.Error()).SetValid(false)
			wf.SendFeedback()
			return
		}
		if err := saveRepos(fresh, "stars"); err != nil {
			wf.NewItem("保存缓存失败").SetSubtitle(err.Error()).SetValid(false)
		}
		repos, _ = queryRepos("stars", query, maxStars)
	}

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

// handleRepos processes the 'repos' subcommand.
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

	if len(repos) == 0 && query != "" {
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
		repos, _ = queryRepos("repos", query, maxRepos)
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

// handleGists processes the 'gists' subcommand.
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

	if len(gists) == 0 && query != "" {
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
		gists, _ = queryGists(query, maxGists)
	}

	for _, g := range gists {
		title := g.GetDescription()
		if title == "" {
			for fname := range g.Files {
				title = string(fname)
				break
			}
			if title == "" {
				title = g.GetID()
			}
		}
		if !g.GetPublic() {
			title += " 🔒"
		}

		var filenames []string
		for filename := range g.Files {
			filenames = append(filenames, string(filename))
		}

		subtitle := fmt.Sprintf("%d 个文件 | 更新于 %s", len(filenames), g.GetUpdatedAt().Format("2006-01-02"))

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

// handleSearchRepo processes the 'search-repo' subcommand.
func handleSearchRepo(query string) {
	wf := NewWorkflow()

	if query == "" {
		wf.NewItem("请输入关键词进行搜索").SetValid(false)
		wf.SendFeedback()
		return
	}

	searchURL := "https://github.com/search?q=" + url.QueryEscape(query) + "&type=repositories"
	wf.NewItem("✪ 在 GitHub 打开搜索结果").
		SetSubtitle(searchURL).
		SetArg(searchURL)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Correctly call the exported function from github.go
	repos, err := SearchPublicRepos(ctx, query)
	if err != nil {
		wf.NewItem("搜索仓库失败").SetSubtitle(err.Error()).SetValid(false)
		wf.SendFeedback()
		return
	}

	for _, r := range repos {
		if r.GetFork() || r.GetArchived() {
			continue
		}
		// The PushedAt field should be used for search results as it's more relevant
		pushedAt := r.GetPushedAt().Time
		if r.PushedAt == nil {
			pushedAt = r.GetCreatedAt().Time // Fallback to CreatedAt if PushedAt is nil
		}
		wf.NewItem(r.GetFullName()).
			SetSubtitle(formatSubtitle(r.GetStargazersCount(), pushedAt, r.GetDescription())).
			SetArg(r.GetHTMLURL()).
			SetMatch(normalize(r.GetFullName()+" "+r.GetDescription())).
			SetCmdModifier(r.GetCloneURL(), "复制 Clone URL").
			SetAltModifier(r.GetHTMLURL(), "复制 Repo URL")
	}

	if len(repos) == 0 {
		wf.NewItem(fmt.Sprintf("✖ 未找到匹配结果: %s", query)).SetValid(false)
	}

	wf.SendFeedback()
}

// handleCacheCtl handles cache control commands like 'refresh:stars'.
func handleCacheCtl(arg string) {
	parts := strings.Split(arg, ":")
	if len(parts) != 2 {
		return
	}
	action, cacheType := parts[0], parts[1]

	if action == "refresh" {
		if err := clearCache(cacheType); err != nil {
			fmt.Printf(`{"items":[{"title":"清除 %s 缓存失败","subtitle":"%s","valid":false}]}`, cacheType, err.Error())
			return
		}
		fmt.Printf("缓存 '%s' 已清空。请重新运行命令以刷新。", cacheType)
	}
}

