package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

var (
	githubUser  = getEnv("GITHUB_USER", "")
	githubToken = getEnv("GITHUB_TOKEN", "")
	maxRepos    = parseInt(getEnv("MAX_REPOS", "300"), 300)
	maxGists    = parseInt(getEnv("MAX_GISTS", "100"), 100)
	maxResults  = parseInt(getEnv("MAX_RESULTS", "30"), 30)
)

type GitHubClient struct {
	*github.Client
}

func newGitHubClient() *GitHubClient {
	var httpClient *http.Client
	if githubToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
		httpClient = oauth2.NewClient(ctx, ts)
	}
	return &GitHubClient{
		Client: github.NewClient(httpClient),
	}
}

// fetchAllPages 并发获取所有分页数据
func (c *GitHubClient) fetchAllPages(fetcher func(page int) ([]interface{}, *github.Response, error)) ([]interface{}, error) {
	var allItems []interface{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 先获取第一页，得到总页数
	items, resp, err := fetcher(1)
	if err != nil {
		return nil, err
	}
	allItems = append(allItems, items...)

	if resp.LastPage == 0 { // 如果没有分页
		return allItems, nil
	}

	for page := 2; page <= resp.LastPage; page++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return // 超时或取消
			default:
				pageItems, _, pageErr := fetcher(p)
				if pageErr != nil {
					select {
					case errChan <- pageErr:
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

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}
	return allItems, nil
}

func (c *GitHubClient) FetchStars() ([]*github.Repository, error) {
	log.Println("Fetching stars from GitHub API...")
	var allRepos []*github.Repository
	opts := &github.ActivityListStarredOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := c.Activity.ListStarred(context.Background(), "", opts)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 || len(allRepos) >= maxRepos {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos[:min(len(allRepos), maxRepos)], nil
}

func (c *GitHubClient) FetchRepos() ([]*github.Repository, error) {
	log.Println("Fetching repos from GitHub API...")
	var allRepos []*github.Repository
	opts := &github.RepositoryListOptions{Affiliation: "owner", ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := c.Repositories.List(context.Background(), "", opts)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 || len(allRepos) >= maxRepos {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos[:min(len(allRepos), maxRepos)], nil
}

func (c *GitHubClient) FetchGists() ([]*github.Gist, error) {
	log.Println("Fetching gists from GitHub API...")
	var allGists []*github.Gist
	opts := &github.GistListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		gists, resp, err := c.Gists.List(context.Background(), "", opts)
		if err != nil {
			return nil, err
		}
		allGists = append(allGists, gists...)
		if resp.NextPage == 0 || len(allGists) >= maxGists {
			break
		}
		opts.Page = resp.NextPage
	}
	return allGists[:min(len(allGists), maxGists)], nil
}

func (c *GitHubClient) SearchRepos(query string) ([]*github.Repository, error) {
	log.Printf("Searching repos with query: %s\n", query)
	opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: maxResults}}
	result, _, err := c.Search.Repositories(context.Background(), query, opts)
	if err != nil {
		return nil, err
	}
	// 过滤掉 fork 和 archived 的
	var repos []*github.Repository
	for _, r := range result.Repositories {
		if !r.GetFork() && !r.GetArchived() {
			repos = append(repos, r)
		}
	}
	return repos, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
