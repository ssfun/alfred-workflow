// github.go
package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

// 从环境变量中读取配置
var (
	githubUser = getEnv("GITHUB_USER", "")
	githubToken = getEnv("GITHUB_TOKEN", "")
	maxRepos   = parseIntEnv("MAX_REPOS", 300)
	maxStars   = parseIntEnv("MAX_STARS", 300) // 假设 star 和 repo 使用不同的最大值变量
	maxGists   = parseIntEnv("MAX_GISTS", 100)
)

// getGitHubClient 根据环境变量中的 token 创建一个 GitHub API 客户端
func getGitHubClient(ctx context.Context) *github.Client {
	if githubToken == "" {
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// fetchAllPages 并发地从 GitHub API 获取所有分页数据
func fetchAllPages[T any](ctx context.Context, fetchFunc func(page int) ([]T, *github.Response, error)) ([]T, error) {
	var allItems []T
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make(chan error, 1)

	// 先获取第一页来确定总页数
	items, resp, err := fetchFunc(1)
	if err != nil {
		return nil, err
	}
	allItems = append(allItems, items...)

	if resp.LastPage > 1 {
		for page := 2; page <= resp.LastPage; page++ {
			wg.Add(1)
			go func(p int) {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					pageItems, _, pageErr := fetchFunc(p)
					if pageErr != nil {
						select {
						case errs <- pageErr:
						default:
						}
						return
					}
					mu.Lock()
					allItems = append(allItems, pageItems...)
					mu.Unlock()
				}
			}(page)
		}
	}

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		return nil, <-errs
	}

	return allItems, nil
}

// fetchStars 获取用户收藏的仓库
func fetchStars(ctx context.Context) ([]*github.Repository, error) {
	client := getGitHubClient(ctx)
	opts := &github.ActivityListStarredOptions{ListOptions: github.ListOptions{PerPage: 100}}

	starredRepos, err := fetchAllPages(ctx, func(page int) ([]*github.StarredRepository, *github.Response, error) {
		opts.Page = page
		// "starred" 表示按收藏时间排序
		opts.Sort = "created"
		opts.Direction = "desc"
		starred, resp, err := client.Activity.ListStarred(ctx, githubUser, opts)
		return starred, resp, err
	})
	if err != nil {
		return nil, err
	}

	// 关键修复：从 StarredRepository 中提取出 Repository
	var repos []*github.Repository
	for _, starred := range starredRepos {
		repos = append(repos, starred.Repository)
	}

	if len(repos) > maxStars {
		return repos[:maxStars], nil
	}
	return repos, nil
}

// fetchRepos 获取用户自己的仓库
func fetchRepos(ctx context.Context) ([]*github.Repository, error) {
	client := getGitHubClient(ctx)
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "pushed", // 按推送时间排序
		Direction:   "desc",
	}

	repos, err := fetchAllPages(ctx, func(page int) ([]*github.Repository, *github.Response, error) {
		opts.Page = page
		r, resp, err := client.Repositories.List(ctx, githubUser, opts)
		return r, resp, err
	})
	if err != nil {
		return nil, err
	}

	if len(repos) > maxRepos {
		return repos[:maxRepos], nil
	}
	return repos, nil
}

// fetchGists 获取用户的 Gists
func fetchGists(ctx context.Context) ([]*github.Gist, error) {
	client := getGitHubClient(ctx)
	opts := &github.GistListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	gists, err := fetchAllPages(ctx, func(page int) ([]*github.Gist, *github.Response, error) {
		opts.Page = page
		g, resp, err := client.Gists.List(ctx, githubUser, opts)
		return g, resp, err
	})
	if err != nil {
		return nil, err
	}

	if len(gists) > maxGists {
		return gists[:maxGists], nil
	}
	return gists, nil
}
