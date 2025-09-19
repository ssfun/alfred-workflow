package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	apiBase    = "https://api.github.com"
	githubUser = os.Getenv("GITHUB_USER")
	githubTok  = os.Getenv("GITHUB_TOKEN")

	maxRepos = getenvInt("MAX_REPOS", 50)
	maxStars = getenvInt("MAX_STARS", 50)
	maxGists = getenvInt("MAX_GISTS", 50)
)

func githubFetch(url string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	if githubTok != "" {
		req.Header.Set("Authorization", "token "+githubTok)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API 错误: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// ---------- 数据结构 ----------

type Repo struct {
	ID          int64  `json:"id"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	Stars       int    `json:"stargazers_count"`
	UpdatedAt   string `json:"updated_at"`
	Private     bool   `json:"private"`
}

type Gist struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	HTMLURL     string                 `json:"html_url"`
	Public      bool                   `json:"public"`
	UpdatedAt   string                 `json:"updated_at"`
	Files       map[string]interface{} `json:"files"`
}

// ---------- 并发分页获取 ----------
func fetchAll(url string, maxItems int, result interface{}) error {
	perPage := 100
	pages := (maxItems + perPage - 1) / perPage
	var wg sync.WaitGroup
	ch := make(chan []byte, pages)

	for p := 1; p <= pages; p++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			fullURL := fmt.Sprintf("%s?per_page=%d&page=%d", url, perPage, page)
			data, err := githubFetch(fullURL)
			if err == nil {
				ch <- data
			}
		}(p)
	}
	wg.Wait()
	close(ch)

	var all []json.RawMessage
	for data := range ch {
		var arr []json.RawMessage
		json.Unmarshal(data, &arr)
		all = append(all, arr...)
	}
	if len(all) > maxItems {
		all = all[:maxItems]
	}
	final, _ := json.Marshal(all)
	return json.Unmarshal(final, result)
}

// ---------- API 封装 ----------
func fetchStars() ([]Repo, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/user/starred"
	} else {
		url = fmt.Sprintf("%s/users/%s/starred", apiBase, githubUser)
	}
	var repos []Repo
	err := fetchAll(url, maxStars, &repos)
	return repos, err
}

func fetchRepos() ([]Repo, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/user/repos"
	} else {
		url = fmt.Sprintf("%s/users/%s/repos", apiBase, githubUser)
	}
	var repos []Repo
	err := fetchAll(url, maxRepos, &repos)
	return repos, err
}

func fetchGists() ([]Gist, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/gists"
	} else {
		url = fmt.Sprintf("%s/users/%s/gists", apiBase, githubUser)
	}
	var gists []Gist
	err := fetchAll(url, maxGists, &gists)
	return gists, err
}
