package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var (
	apiBase    = "https://api.github.com"
	githubUser = os.Getenv("GITHUB_USER")
	githubTok  = os.Getenv("GITHUB_TOKEN")
)

func githubFetch(url string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	if githubTok != "" {
		req.Header.Set("Authorization", "token "+githubTok)
	}
	client := &http.Client{Timeout: 10 * time.Second}
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

// ---------- API ----------

func fetchStars() ([]Repo, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/user/starred"
	} else {
		url = fmt.Sprintf("%s/users/%s/starred", apiBase, githubUser)
	}
	data, err := githubFetch(url)
	if err != nil {
		return nil, err
	}
	var repos []Repo
	err = json.Unmarshal(data, &repos)
	return repos, err
}

func fetchRepos() ([]Repo, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/user/repos"
	} else {
		url = fmt.Sprintf("%s/users/%s/repos", apiBase, githubUser)
	}
	data, err := githubFetch(url)
	if err != nil {
		return nil, err
	}
	var repos []Repo
	err = json.Unmarshal(data, &repos)
	return repos, err
}

func fetchGists() ([]Gist, error) {
	var url string
	if githubTok != "" {
		url = apiBase + "/gists"
	} else {
		url = fmt.Sprintf("%s/users/%s/gists", apiBase, githubUser)
	}
	data, err := githubFetch(url)
	if err != nil {
		return nil, err
	}
	var gists []Gist
	err = json.Unmarshal(data, &gists)
	return gists, err
}
