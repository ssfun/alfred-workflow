package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var dbPath = filepath.Join(os.Getenv("HOME"),
	"Library", "Caches", "com.runningwithcrayons.Alfred", "github", "github_cache.db")

func initDB() *sql.DB {
	os.MkdirAll(filepath.Dir(dbPath), 0755)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS repos (
		id INTEGER,
		type TEXT,
		full_name TEXT,
		description TEXT,
		html_url TEXT,
		clone_url TEXT,
		stars INTEGER,
		updated_at TEXT,
		private INTEGER,
		normalized TEXT,
		PRIMARY KEY (id,type)
	);`)
	db.Exec(`CREATE TABLE IF NOT EXISTS gists (
		id TEXT PRIMARY KEY,
		description TEXT,
		html_url TEXT,
		public INTEGER,
		updated_at TEXT,
		files TEXT,
		normalized TEXT
	);`)
	db.Exec(`CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT
	);`)
	return db
}

func saveRepos(db *sql.DB, repos []Repo, repoType string) {
	tx, _ := db.Begin()
	tx.Exec("DELETE FROM repos WHERE type=?", repoType)
	for _, r := range repos {
		norm := normalize(r.FullName + " " + r.Description)
		tx.Exec(`INSERT OR REPLACE INTO repos 
			(id,type,full_name,description,html_url,clone_url,stars,updated_at,private,normalized) 
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			r.ID, repoType, r.FullName, r.Description, r.HTMLURL, r.CloneURL, r.Stars, r.UpdatedAt,
			boolToInt(r.Private), norm)
	}
	tx.Commit()
	setMeta(db, "last_"+repoType, time.Now().Format(time.RFC3339))
}

func queryRepos(db *sql.DB, repoType, query string, limit int) []Repo {
	q := "SELECT id,full_name,description,html_url,clone_url,stars,updated_at,private FROM repos WHERE type=?"
	args := []interface{}{repoType}
	if query != "" {
		q += " AND normalized LIKE ?"
		args = append(args, "%"+normalize(query)+"%")
	}
	q += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)
	rows, _ := db.Query(q, args...)
	defer rows.Close()

	var res []Repo
	for rows.Next() {
		var r Repo
		var priv int
		rows.Scan(&r.ID, &r.FullName, &r.Description, &r.HTMLURL, &r.CloneURL, &r.Stars, &r.UpdatedAt, &priv)
		r.Private = priv == 1
		res = append(res, r)
	}
	return res
}

func saveGists(db *sql.DB, gists []Gist) {
	tx, _ := db.Begin()
	tx.Exec("DELETE FROM gists")
	for _, g := range gists {
		filesStr := []string{}
		for fname := range g.Files {
			filesStr = append(filesStr, fname)
		}
		norm := normalize(g.Description + " " + strings.Join(filesStr, " "))
		tx.Exec(`INSERT OR REPLACE INTO gists 
			(id,description,html_url,public,updated_at,files,normalized) 
			VALUES (?,?,?,?,?,?,?)`,
			g.ID, g.Description, g.HTMLURL, boolToInt(g.Public), g.UpdatedAt,
			strings.Join(filesStr, ","), norm)
	}
	tx.Commit()
	setMeta(db, "last_gists", time.Now().Format(time.RFC3339))
}

func queryGists(db *sql.DB, query string, limit int) []Gist {
	q := "SELECT id,description,html_url,public,updated_at,files FROM gists WHERE 1=1"
	args := []interface{}{}
	if query != "" {
		q += " AND normalized LIKE ?"
		args = append(args, "%"+normalize(query)+"%")
	}
	q += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)
	rows, _ := db.Query(q, args...)
	defer rows.Close()

	var res []Gist
	for rows.Next() {
		var g Gist
		var pub int
		var filesStr string
		rows.Scan(&g.ID, &g.Description, &g.HTMLURL, &pub, &g.UpdatedAt, &filesStr)
		g.Public = pub == 1
		g.Files = map[string]interface{}{}
		for _, fn := range strings.Split(filesStr, ",") {
			if fn != "" {
				g.Files[fn] = nil
			}
		}
		res = append(res, g)
	}
	return res
}

// Utility
func getMeta(db *sql.DB, key string) string {
	var val string
	db.QueryRow("SELECT value FROM meta WHERE key=?", key).Scan(&val)
	return val
}
func setMeta(db *sql.DB, key, value string) {
	db.Exec("INSERT OR REPLACE INTO meta(key,value) VALUES(?,?)", key, value)
}
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

//
// ------------ Handlers ------------
//
func HandleStars(query string) []AlfredItem {
	db := initDB()
	repos := queryRepos(db, "stars", query, 50)
	if len(repos) == 0 && query == "" {
		if fresh, err := fetchStars(); err == nil {
			saveRepos(db, fresh, "stars")
			repos = queryRepos(db, "stars", query, 50)
		}
	}

	var items []AlfredItem
	for _, r := range repos {
		title := r.FullName
		if r.Private {
			title += " 🔒"
		}
		sub := fmt.Sprintf("⭐ %d · 更新时间 %s · %s", r.Stars, formatDate(r.UpdatedAt), r.Description)
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
	return items
}

func HandleRepos(query string) []AlfredItem {
	db := initDB()
	repos := queryRepos(db, "repos", query, 50)
	if len(repos) == 0 && query == "" {
		if fresh, err := fetchRepos(); err == nil {
			saveRepos(db, fresh, "repos")
			repos = queryRepos(db, "repos", query, 50)
		}
	}

	var items []AlfredItem
	for _, r := range repos {
		title := r.FullName
		if r.Private {
			title += " 🔒"
		}
		sub := fmt.Sprintf("⭐ %d · 更新时间 %s · %s", r.Stars, formatDate(r.UpdatedAt), r.Description)
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
	return items
}

func HandleGists(query string) []AlfredItem {
	db := initDB()
	gists := queryGists(db, query, 50)
	if len(gists) == 0 && query == "" {
		if fresh, err := fetchGists(); err == nil {
			saveGists(db, fresh)
			gists = queryGists(db, query, 50)
		}
	}

	var items []AlfredItem
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
	return items
}

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
	msg := ""
	ok := false

	switch t {
	case "repos":
		if fresh, err := fetchRepos(); err == nil {
			saveRepos(db, fresh, "repos")
			msg, ok = "✅ Repos 缓存已刷新", true
		} else {
			msg = "⚠️ Repos 刷新失败: " + err.Error()
		}
	case "stars":
		if fresh, err := fetchStars(); err == nil {
			saveRepos(db, fresh, "stars")
			msg, ok = "✅ Stars 缓存已刷新", true
		} else {
			msg = "⚠️ Stars 刷新失败: " + err.Error()
		}
	case "gists":
		if fresh, err := fetchGists(); err == nil {
			saveGists(db, fresh)
			msg, ok = "✅ Gists 缓存已刷新", true
		} else {
			msg = "⚠️ Gists 刷新失败: " + err.Error()
		}
	default:
		return []AlfredItem{{
			Title:    "未知类型: " + t,
			Subtitle: "无法刷新",
			Valid:    false,
		}}
	}

	if ok {
		return []AlfredItem{{
			Title:    msg,
			Subtitle: "数据已更新，正在重新加载...",
			Valid:    false,
			Arg:      "reload:" + t,
		}}
	} else {
		return []AlfredItem{{
			Title: msg,
			Valid: false,
		}}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
