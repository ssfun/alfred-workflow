package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
)

func handleStars(wf *Workflow, query string) error {
	if query == "" {
		wf.NewItem("🌐 打开 GitHub Stars 页面").
			Subtitle(fmt.Sprintf("https://github.com/%s?tab=stars", githubUser)).
			Arg(fmt.Sprintf("https://github.com/%s?tab=stars", githubUser))
		wf.NewItem("♻ 刷新 Stars 缓存").
			Subtitle(getCacheInfo("stars")).
			Arg("refresh:stars")
	}

	repos, err := queryRepos("stars", query, maxRepos)
	if err != nil {
		return err
	}

	// 如果缓存为空且没有搜索词，则从 API 获取
	if len(repos) == 0 && query == "" {
		client := newGitHubClient()
		freshRepos, fetchErr := client.FetchStars()
		if fetchErr != nil {
			return fetchErr
		}
		if err := saveRepos("stars", freshRepos); err != nil {
			return err
		}
		repos = freshRepos
	}

	for _, r := range repos {
		item := wf.NewItem(r.GetFullName()).
			Subtitle(formatRepoSubtitle(r)).
			Arg(r.GetHTMLURL()).
			Match(makeMatchKeywords(r.GetFullName())).
			UID(fmt.Sprintf("repo-%d", r.GetID())).
			Cmd(r.GetCloneURL(), "复制 Clone URL").
			Alt(r.GetHTMLURL(), "复制 Repo URL")
		if r.GetPrivate() {
			item.Title = fmt.Sprintf("%s 🔒", item.Title)
		}
	}
	return nil
}

func handleRepos(wf *Workflow, query string) error {
	if query == "" {
		wf.NewItem("✪ 打开 Repos 页面").
			Subtitle(fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser)).
			Arg(fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser))
		wf.NewItem("♻ 刷新 Repos 缓存").
			Subtitle(getCacheInfo("repos")).
			Arg("refresh:repos")
	}

	repos, err := queryRepos("repos", query, maxRepos)
	if err != nil {
		return err
	}

	if len(repos) == 0 && query == "" {
		client := newGitHubClient()
		freshRepos, fetchErr := client.FetchRepos()
		if fetchErr != nil {
			return fetchErr
		}
		if err := saveRepos("repos", freshRepos); err != nil {
			return err
		}
		repos = freshRepos
	}

	for _, r := range repos {
		item := wf.NewItem(r.GetFullName()).
			Subtitle(formatRepoSubtitle(r)).
			Arg(r.GetHTMLURL()).
			Match(makeMatchKeywords(r.GetFullName())).
			UID(fmt.Sprintf("repo-%d", r.GetID())).
			Cmd(r.GetCloneURL(), "复制 Clone URL").
			Alt(r.GetHTMLURL(), "复制 Repo URL")
		if r.GetPrivate() {
			item.Title = fmt.Sprintf("%s 🔒", item.Title)
		}
	}
	return nil
}

func handleGists(wf *Workflow, query string) error {
	if query == "" {
		wf.NewItem("✪ 打开 Gists 页面").
			Subtitle(fmt.Sprintf("https://gist.github.com/%s", githubUser)).
			Arg(fmt.Sprintf("https://gist.github.com/%s", githubUser))
		wf.NewItem("♻ 刷新 Gists 缓存").
			Subtitle(getCacheInfo("gists")).
			Arg("refresh:gists")
	}

	gists, err := queryGists(query, maxGists)
	if err != nil {
		return err
	}

	if len(gists) == 0 && query == "" {
		client := newGitHubClient()
		freshGists, fetchErr := client.FetchGists()
		if fetchErr != nil {
			return fetchErr
		}
		if err := saveGists(freshGists); err != nil {
			return err
		}
		gists = freshGists
	}

	for _, g := range gists {
		title := g.GetDescription()
		if title == "" {
			title = "(无描述)"
		}

		item := wf.NewItem(title).
			Subtitle(formatGistSubtitle(g)).
			Arg(g.GetHTMLURL()).
			UID(g.GetID()).
			Cmd(g.GetID(), "复制 Gist ID").
			Alt(g.GetHTMLURL(), "复制 Gist URL")
		if !g.GetPublic() {
			item.Title = fmt.Sprintf("%s 🔒", item.Title)
		}
	}
	return nil
}

func handleSearchRepos(wf *Workflow, query string) error {
	if query == "" {
		wf.NewItem("请输入关键词进行搜索").Valid(false)
		return nil
	}

	searchURL := fmt.Sprintf("https://github.com/search?q=%s&type=repositories", url.QueryEscape(query))
	wf.NewItem("✪ 在 GitHub 打开搜索结果").
		Subtitle(searchURL).
		Arg(searchURL)

	client := newGitHubClient()
	repos, err := client.SearchRepos(query)
	if err != nil {
		return err
	}

	for _, r := range repos {
		item := wf.NewItem(r.GetFullName()).
			Subtitle(formatRepoSubtitle(r)).
			Arg(r.GetHTMLURL()).
			Match(makeMatchKeywords(r.GetFullName())).
			UID(fmt.Sprintf("repo-%d", r.GetID())).
			Cmd(r.GetCloneURL(), "复制 Clone URL").
			Alt(r.GetHTMLURL(), "复制 Repo URL")
		if r.GetPrivate() {
			item.Title = fmt.Sprintf("%s 🔒", item.Title)
		}
	}
	return nil
}


func handleCacheCtl(wf *Workflow, query string) error {
	parts := strings.SplitN(query, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("无效的缓存控制命令: %s", query)
	}
	action, key := parts[0], parts[1]

	switch action {
	case "clear":
		switch key {
		case "stars":
			clearRepos("stars")
			wf.NewItem("Stars 缓存已清除")
		case "repos":
			clearRepos("repos")
			wf.NewItem("Repos 缓存已清除")
		case "gists":
			clearGists()
			wf.NewItem("Gists 缓存已清除")
		case "all":
			clearRepos("stars")
			clearRepos("repos")
			clearGists()
			wf.NewItem("所有缓存已清除")
		default:
			return fmt.Errorf("无效的清除目标: %s", key)
		}
	case "refresh":
		// 在 Go 中直接刷新，不再需要调用外部 AppleScript
		client := newGitHubClient()
		switch key {
		case "stars":
			wf.NewItem("正在刷新 Stars...").Valid(false)
			clearRepos("stars")
			stars, err := client.FetchStars()
			if err != nil { return err }
			saveRepos("stars", stars)
			wf.NewItem("Stars 缓存已刷新")
		case "repos":
			wf.NewItem("正在刷新 Repos...").Valid(false)
			clearRepos("repos")
			repos, err := client.FetchRepos()
			if err != nil { return err }
			saveRepos("repos", repos)
			wf.NewItem("Repos 缓存已刷新")
		case "gists":
			wf.NewItem("正在刷新 Gists...").Valid(false)
			clearGists()
			gists, err := client.FetchGists()
			if err != nil { return err }
			saveGists(gists)
			wf.NewItem("Gists 缓存已刷新")
		default:
			return fmt.Errorf("无效的刷新目标: %s", key)
		}
	default:
		return fmt.Errorf("无效的操作: %s", action)
	}

	return nil
}

// --- Subtitle Formatters ---

func formatRepoSubtitle(r *github.Repository) string {
	return fmt.Sprintf("★ %d · 更新于 %s · %s",
		r.GetStargazersCount(),
		r.GetUpdatedAt().Format("2006-01-02"),
		r.GetDescription(),
	)
}

func formatGistSubtitle(g *github.Gist) string {
	var fileNames []string
	for fn := range g.Files {
		fileNames = append(fileNames, string(fn))
	}

	filesPreview := strings.Join(fileNames[:min(len(fileNames), 3)], ", ")
	if len(fileNames) > 3 {
		filesPreview += "..."
	}

	return fmt.Sprintf("%d 个文件: %s | 更新于 %s",
		len(fileNames),
		filesPreview,
		g.GetUpdatedAt().Format("2006-01-02"),
	)
}
