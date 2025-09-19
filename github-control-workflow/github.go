// github.go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

var (
	githubUser = getEnv("GITHUB_USER", "default")
	githubToken = getEnv("GITHUB_TOKEN", "")
	maxStars   = parseIntEnv(getEnv("MAX_REPOS", "300"))
	maxRepos   = parseIntEnv(getEnv("MAX_REPOS", "300"))
	maxGists   = parseIntEnv(getEnv("MAX_GISTS", "100"))
)

// getGitHubClient creates a new GitHub API client
func getGitHubClient(ctx context.Context) *github.Client {
	var tc *http.Client
	if githubToken != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
		tc = oauth2.NewClient(ctx, ts)
	}
	return github.NewClient(tc)
}

// fetchAll is a generic function to fetch all pages of a paginated GitHub API endpoint.
func fetchAll[T any](ctx context.Context, initialURL string) ([]T, error) {
	client := getGitHubClient(ctx)
	req, err := client.NewRequest("GET", initialURL, nil)
	if err != nil {
		return nil, err
	}

	var allItems []T
	for {
		var items []T
		resp, err := client.Do(ctx, req, &items)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if resp.NextPage == 0 {
			break
		}
		req.URL.RawQuery = resp.Request.URL.RawQuery
		req.URL.Path = fmt.Sprintf("/user/starred?page=%d", resp.NextPage)
	}
	return allItems, nil
}

// fetchStars fetches all starred repositories for the user.
func fetchStars(ctx context.Context) ([]*github.Repository, error) {
	client := getGitHubClient(ctx)
	var allRepos []*github.Repository
	opts := &github.ActivityListStarredOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := client.Activity.ListStarred(ctx, "", opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			allRepos = append(allRepos, r.Repository)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos, nil
}

// fetchRepos fetches all repositories for the user.
func fetchRepos(ctx context.Context) ([]*github.Repository, error) {
	client := getGitHubClient(ctx)
	var allRepos []*github.Repository
	opts := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := client.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos, nil
}

// fetchGists fetches all gists for the user.
func fetchGists(ctx context.Context) ([]*github.Gist, error) {
	client := getGitHubClient(ctx)
	var allGists []*github.Gist
	opts := &github.GistListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		gists, resp, err := client.Gists.List(ctx, "", opts)
		if err != nil {
			return nil, err
		}
		allGists = append(allGists, gists...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allGists, nil
}

// SearchPublicRepos searches for public repositories on GitHub.
func SearchPublicRepos(ctx context.Context, query string) ([]*github.Repository, error) {
	client := getGitHubClient(ctx)
	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 30, // Alfred doesn't need more than this for a quick search
		},
	}
	result, _, err := client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	return result.Repositories, nil
}

