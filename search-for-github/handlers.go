package main

import (
	"fmt"
	"strings"
)

// ---------- Stars ----------
func HandleStars(query string) []AlfredItem {
	db := initDB()
	items := []AlfredItem{}

	if query == "" {
		items = append(items,
			AlfredItem{
				Title:    "✪ 打开 Stars 页面",
				Subtitle: fmt.Sprintf("https://github.com/%s?tab=stars", githubUser),
				Arg:      fmt.Sprintf("https://github.com/%s?tab=stars", githubUser),
				Valid:    true,
			},
			AlfredItem{
				Title:    "♻ 刷新 Stars 缓存",
				Subtitle: cacheInfo(db, "stars"),
				Arg:      "refresh:stars",
				Valid:    true,
				Variables: map[string]string{
                    "querysubtitle": cacheInfo(db, "stars"),
                },
			},
		)
	}

	repos := queryRepos(db, "stars", query, maxStars)
	if len(repos) == 0 && query == "" {
		if fresh, err := fetchStars(); err == nil {
			saveRepos(db, fresh, "stars")
			repos = queryRepos(db, "stars", query, maxStars)
		}
	}

	for _, r := range repos {
		title := r.FullName
		if r.Private {
			title += " 🔒"
		}
		sub := fmt.Sprintf("★ %d · 更新时间 %s · %s", r.Stars, formatDate(r.UpdatedAt), r.Description)
		items = append(items, AlfredItem{
			Title:    title,
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
	if len(items) == 0 {
		items = append(items, AlfredItem{Title: "✖ 没有结果", Valid: false})
	}
	return items
}

// ---------- Repos ----------
func HandleRepos(query string) []AlfredItem {
	db := initDB()
	items := []AlfredItem{}

	if query == "" {
		items = append(items,
			AlfredItem{
				Title:    "✪ 打开 Repos 页面",
				Subtitle: fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser),
				Arg:      fmt.Sprintf("https://github.com/%s?tab=repositories", githubUser),
				Valid:    true,
			},
			AlfredItem{
				Title:    "♻ 刷新 Repos 缓存",
				Subtitle: cacheInfo(db, "repos"),
				Arg:      "refresh:repos",
				Valid:    true,
				Variables: map[string]string{
                    "querysubtitle": cacheInfo(db, "repos"),
                },
			},
		)
	}

	repos := queryRepos(db, "repos", query, maxRepos)
	if len(repos) == 0 && query == "" {
		if fresh, err := fetchRepos(); err == nil {
			saveRepos(db, fresh, "repos")
			repos = queryRepos(db, "repos", query, maxRepos)
		}
	}

	for _, r := range repos {
		title := r.FullName
		if r.Private {
			title += " 🔒"
		}
		sub := fmt.Sprintf("★ %d · 更新时间 %s · %s", r.Stars, formatDate(r.UpdatedAt), r.Description)
		items = append(items, AlfredItem{
			Title:    title,
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
	if len(items) == 0 {
		items = append(items, AlfredItem{Title: "✖ 没有结果", Valid: false})
	}
	return items
}

// ---------- Gists ----------
func HandleGists(query string) []AlfredItem {
	db := initDB()
	items := []AlfredItem{}

	if query == "" {
		items = append(items,
			AlfredItem{
				Title:    "✪ 打开 Gists 页面",
				Subtitle: fmt.Sprintf("https://gist.github.com/%s", githubUser),
				Arg:      fmt.Sprintf("https://gist.github.com/%s", githubUser),
				Valid:    true,
			},
			AlfredItem{
				Title:    "♻ 刷新 Gists 缓存",
				Subtitle: cacheInfo(db, "gists"),
				Arg:      "refresh:gists",
				Valid:    true,
				Variables: map[string]string{
                    "querysubtitle": cacheInfo(db, "stars"),
                },
			},
		)
	}

	gists := queryGists(db, query, maxGists)
	if len(gists) == 0 && query == "" {
		if fresh, err := fetchGists(); err == nil {
			saveGists(db, fresh)
			gists = queryGists(db, query, maxGists)
		}
	}

	for _, g := range gists {
		title := g.Description
		if title == "" {
			title = "(无描述)"
		}
		if !g.Public {
			title += " 🔒"
		}
		files := []string{}
		for fname := range g.Files {
			files = append(files, fname)
		}
		filesPreview := strings.Join(files[:min(3, len(files))], ", ")
		if len(files) > 3 {
			filesPreview += "..."
		}
		sub := fmt.Sprintf("%d 个文件: %s | Updated %s", len(files), filesPreview, formatDate(g.UpdatedAt))
		items = append(items, AlfredItem{
			Title:    title,
			Subtitle: sub,
			Arg:      g.HTMLURL,
			Valid:    true,
			Match:    normalize(title + " " + filesPreview),
			Mods: map[string]AlfredMod{
				"cmd": {Arg: g.ID, Subtitle: "复制 Gist ID"},
				"alt": {Arg: g.HTMLURL, Subtitle: "复制 Gist URL"},
			},
		})
	}
	if len(items) == 0 {
		items = append(items, AlfredItem{Title: "✖ 没有结果", Valid: false})
	}
	return items
}

// ---------- Utils for Clear & Refresh ----------
func HandleClear(t string) string {
	db := initDB()
	switch t {
	case "repos", "stars":
		db.Exec("DELETE FROM repos WHERE type=?", t)
	case "gists":
		db.Exec("DELETE FROM gists")
	case "all":
		db.Exec("DELETE FROM repos")
		db.Exec("DELETE FROM gists")
	default:
		return "❓ 未知类型: " + t
	}
	return "✅ 已清空缓存: " + t
}

func HandleRefresh(t string) []AlfredItem {
	db := initDB()
	switch t {
	case "stars":
		if fresh, err := fetchStars(); err == nil {
			saveRepos(db, fresh, "stars")
			triggerAlfred("stars.refresh")
			return []AlfredItem{{
				Title:    "♻ Stars 缓存已刷新",
				Subtitle: cacheInfo(db, "stars"),
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": cacheInfo(db, "stars"),
				},
			}}
		} else {
			return []AlfredItem{{Title: "⚠️ Stars 刷新失败: " + err.Error(), Valid: false}}
		}
	case "repos":
		if fresh, err := fetchRepos(); err == nil {
			saveRepos(db, fresh, "repos")
			triggerAlfred("repos.refresh")
			return []AlfredItem{{
				Title:    "♻ Repos 缓存已刷新",
				Subtitle: cacheInfo(db, "repos"),
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": cacheInfo(db, "repos"),
				},
			}}
		} else {
			return []AlfredItem{{Title: "⚠️ Repos 刷新失败: " + err.Error(), Valid: false}}
		}
	case "gists":
		if fresh, err := fetchGists(); err == nil {
			saveGists(db, fresh)
			triggerAlfred("gists.refresh")
			return []AlfredItem{{
				Title:    "♻ Gists 缓存已刷新",
				Subtitle: cacheInfo(db, "gists"),
				Valid:    false,
				Variables: map[string]string{
					"querysubtitle": cacheInfo(db, "gists"),
				},
			}}
		} else {
			return []AlfredItem{{Title: "⚠️ Gists 刷新失败: " + err.Error(), Valid: false}}
		}
	}
	return []AlfredItem{{Title: "未知类型: " + t, Valid: false}}
}
