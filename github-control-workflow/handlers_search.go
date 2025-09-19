package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"io/ioutil"
)

var maxResults = getenvInt("MAX_RESULTS", 30)

type SearchRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	Stars       int    `json:"stargazers_count"`
	PushedAt    string `json:"pushed_at"`
	Fork        bool   `json:"fork"`
	Archived    bool   `json:"archived"`
}

func searchRepos(query string, token string, perPage int) ([]SearchRepo, error) {
	apiURL := "https://api.github.com/search/repositories?q=" + url.QueryEscape(query) + "&per_page=" + fmt.Sprint(perPage)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub search API error: %d %s", resp.StatusCode, string(body))
	}
	var result struct {
		Items []SearchRepo `json:"items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func HandleSearchRepo(query string) []AlfredItem {
	if query == "" {
		return []AlfredItem{{
			Title:    "请输入关键词进行搜索",
			Valid:    false,
		}}
	}

	repos, err := searchRepos(query, githubTok, maxResults)
	if err != nil {
		return []AlfredItem{{
			Title:    "GitHub Search API 出错",
			Subtitle: err.Error(),
			Valid:    false,
		}}
	}

	// 顶部入口
	searchURL := fmt.Sprintf("https://github.com/search?q=%s&type=repositories", url.QueryEscape(query))
	items := []AlfredItem{{
		Title:    "✪ 打开搜索结果页",
		Subtitle: searchURL,
		Arg:      searchURL,
		Valid:    true,
	}}

	// 过滤 fork、archived
	filtered := []SearchRepo{}
	for _, r := range repos {
		if !r.Fork && !r.Archived {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		items = append(items, AlfredItem{Title: fmt.Sprintf("✖ 未找到匹配结果: %s", query), Valid: false})
		return items
	}

	for _, r := range filtered {
		desc := r.Description
		if desc == "" {
			desc = "(无描述)"
		}
		sub := fmt.Sprintf("★ %d · 更新时间 %s · %s", r.Stars, formatDate(r.PushedAt), desc)
		items = append(items, AlfredItem{
			Title:    r.FullName,
			Subtitle: sub,
			Arg:      r.HTMLURL,
			Valid:    true,
			Match:    normalize(r.FullName),
			Mods: map[string]AlfredMod{
				"cmd": {Arg: r.CloneURL, Subtitle: "复制 Clone URL"},
				"alt": {Arg: r.HTMLURL, Subtitle: "复制 Repo URL"},
			},
		})
	}
	return items
}
