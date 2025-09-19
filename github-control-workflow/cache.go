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

func getCachePath() string {
	// 优先从环境获取
	cacheDir := os.Getenv("GITHUB_CACHE_DIR")
	if cacheDir == "" {
		bundleID := os.Getenv("alfred_workflow_bundleid")
		if bundleID == "" {
			bundleID = "default.githubwf"
		}
		cacheDir = filepath.Join(os.Getenv("HOME"),
			"Library", "Caches", "com.runningwithcrayons.Alfred", bundleID)
	}
	os.MkdirAll(cacheDir, 0755)
	return filepath.Join(cacheDir, "github_cache.db")
}

var dbPath = getCachePath()

func initDB() *sql.DB {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	// 初始化表
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

// ------------ Repos/Gists 存取逻辑 ------------
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
	setMeta(db, "last_"+repoType, time.Now().Format("2006-01-02 15:04"))
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
	setMeta(db, "last_gists", time.Now().Format("2006-01-02 15:04"))
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

// ------------ Meta & Utils ------------
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

// 统计缓存条目 + 最近更新时间
func cacheInfo(db *sql.DB, t string) string {
    count := 0
    last := ""

    switch t {
    case "stars":
        db.QueryRow("SELECT COUNT(*) FROM repos WHERE type='stars'").Scan(&count)
    case "repos":
        db.QueryRow("SELECT COUNT(*) FROM repos WHERE type='repos'").Scan(&count)
    case "gists":
        db.QueryRow("SELECT COUNT(*) FROM gists").Scan(&count)
    }

    last = getMeta(db, "last_"+t)
    if last == "" {
        return fmt.Sprintf("无缓存记录")
    }
    return fmt.Sprintf("%d 条 · 最近更新 %s", count, last)
}
