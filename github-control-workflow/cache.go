// cache.go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	_ "modernc.org/sqlite"
)

var (
	cacheDir = filepath.Join(os.Getenv("HOME"), "Library/Caches/com.runningwithcrayons.Alfred/com.sfun.github")
	dbPath   = filepath.Join(cacheDir, "github_cache.db")
)

// initDB 打开数据库连接并确保表已创建
func initDB() (*sql.DB, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	return db, nil
}

// createTables 创建数据库表
func createTables(db *sql.DB) error {
	repoTable := `
    CREATE TABLE IF NOT EXISTS repos (
        id INTEGER PRIMARY KEY, type TEXT NOT NULL, full_name TEXT, full_name_norm TEXT,
        description TEXT, desc_norm TEXT, html_url TEXT, clone_url TEXT, stars INTEGER,
        updated_at TEXT, private INTEGER, fork INTEGER, archived INTEGER, fetched_at INTEGER
    );`
	gistTable := `
    CREATE TABLE IF NOT EXISTS gists (
        id TEXT PRIMARY KEY, description TEXT, desc_norm TEXT, html_url TEXT, public INTEGER,
        updated_at TEXT, files TEXT, files_norm TEXT, fetched_at INTEGER
    );`
	if _, err := db.Exec(repoTable); err != nil {
		return err
	}
	if _, err := db.Exec(gistTable); err != nil {
		return err
	}
	return nil
}

// clearCache 清除指定类型的缓存
func clearCache(cacheType string) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()

	switch cacheType {
	case "stars":
		_, err = db.Exec("DELETE FROM repos WHERE type = 'stars'")
	case "repos":
		_, err = db.Exec("DELETE FROM repos WHERE type = 'repos'")
	case "gists":
		_, err = db.Exec("DELETE FROM gists")
	case "all":
		if _, err = db.Exec("DELETE FROM repos"); err != nil {
			return err
		}
		_, err = db.Exec("DELETE FROM gists")
	default:
		err = fmt.Errorf("unknown cache type: %s", cacheType)
	}
	return err
}

// getCacheInfo 获取缓存信息（条数和更新时间）
func getCacheInfo(cacheType string) string {
	db, err := initDB()
	if err != nil {
		return fmt.Sprintf("DB Error: %s", err)
	}
	defer db.Close()

	var query string
	switch cacheType {
	case "stars", "repos":
		query = fmt.Sprintf("SELECT COUNT(*), MAX(fetched_at) FROM repos WHERE type = '%s'", cacheType)
	case "gists":
		query = "SELECT COUNT(*), MAX(fetched_at) FROM gists"
	default:
		return "未知类型"
	}

	var count int
	var fetchedAt sql.NullInt64
	if err := db.QueryRow(query).Scan(&count, &fetchedAt); err != nil || count == 0 {
		return "无缓存"
	}

	updateTime := "未知"
	if fetchedAt.Valid {
		updateTime = time.Unix(fetchedAt.Int64, 0).Format("2006-01-02 15:04")
	}
	return fmt.Sprintf("%d 项, 更新于 %s", count, updateTime)
}

// saveRepos 保存仓库列表到数据库
func saveRepos(repos []*github.Repository, repoType string) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO repos
        (id, type, full_name, full_name_norm, description, desc_norm, html_url, clone_url, stars, updated_at, private, fork, archived, fetched_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, r := range repos {
		desc := r.GetDescription()
		if _, err := stmt.Exec(r.GetID(), repoType, r.GetFullName(), normalize(r.GetFullName()), desc, normalize(desc), r.GetHTMLURL(), r.GetCloneURL(), r.GetStargazersCount(), r.GetUpdatedAt().Format(time.RFC3339), r.GetPrivate(), r.GetFork(), r.GetArchived(), now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// queryRepos 从数据库查询仓库
func queryRepos(repoType, searchQuery string, limit int) ([]*github.Repository, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	normQuery := "%" + normalize(searchQuery) + "%"
	query := `SELECT id, full_name, description, html_url, clone_url, stars, updated_at, private, fork, archived FROM repos
              WHERE type = ? AND (full_name_norm LIKE ? OR desc_norm LIKE ?)
              ORDER BY datetime(updated_at) DESC LIMIT ?`
	rows, err := db.Query(query, repoType, normQuery, normQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*github.Repository
	for rows.Next() {
		var r github.Repository
		var id sql.NullInt64
		var fullName, desc, htmlURL, cloneURL, updatedAtStr sql.NullString
		var stars sql.NullInt64
		var private, fork, archived sql.NullBool
		if err := rows.Scan(&id, &fullName, &desc, &htmlURL, &cloneURL, &stars, &updatedAtStr, &private, &fork, &archived); err != nil {
			return nil, err
		}
		r.ID = github.Int64(id.Int64)
		r.FullName = github.String(fullName.String)
		r.Description = github.String(desc.String)
		r.HTMLURL = github.String(htmlURL.String)
		r.CloneURL = github.String(cloneURL.String)
		r.StargazersCount = github.Int(int(stars.Int64))
		if t, err := time.Parse(time.RFC3339, updatedAtStr.String); err == nil {
			r.UpdatedAt = &github.Timestamp{Time: t}
		}
		r.Private = github.Bool(private.Bool)
		r.Fork = github.Bool(fork.Bool)
		r.Archived = github.Bool(archived.Bool)
		repos = append(repos, &r)
	}
	return repos, nil
}

// saveGists 保存 Gist 列表到数据库
func saveGists(gists []*github.Gist) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO gists
        (id, description, desc_norm, html_url, public, updated_at, files, files_norm, fetched_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, g := range gists {
		var filenames []string
		var normFilenames []string
		for f := range g.Files {
			filenames = append(filenames, string(f))
			normFilenames = append(normFilenames, normalize(string(f)))
		}
		filesJSON, _ := json.Marshal(filenames)
		desc := g.GetDescription()
		if _, err := stmt.Exec(g.GetID(), desc, normalize(desc), g.GetHTMLURL(), g.GetPublic(), g.GetUpdatedAt().Format(time.RFC3339), string(filesJSON), strings.Join(normFilenames, " "), now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// queryGists 从数据库查询 Gists
func queryGists(searchQuery string, limit int) ([]*github.Gist, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	normQuery := "%" + normalize(searchQuery) + "%"
	query := `SELECT id, description, html_url, public, updated_at, files FROM gists
              WHERE desc_norm LIKE ? OR files_norm LIKE ?
              ORDER BY datetime(updated_at) DESC LIMIT ?`
	rows, err := db.Query(query, normQuery, normQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gists []*github.Gist
	for rows.Next() {
		var g github.Gist
		var id, desc, htmlURL, updatedAtStr, filesJSON sql.NullString
		var public sql.NullBool
		if err := rows.Scan(&id, &desc, &htmlURL, &public, &updatedAtStr, &filesJSON); err != nil {
			return nil, err
		}
		g.ID = github.String(id.String)
		g.Description = github.String(desc.String)
		g.HTMLURL = github.String(htmlURL.String)
		g.Public = github.Bool(public.Bool)
		if t, err := time.Parse(time.RFC3339, updatedAtStr.String); err == nil {
			g.UpdatedAt = &t
		}
		var filenames []string
		if err := json.Unmarshal([]byte(filesJSON.String), &filenames); err == nil {
			g.Files = make(map[github.GistFilename]github.GistFile)
			for _, f := range filenames {
				g.Files[github.GistFilename(f)] = github.GistFile{}
			}
		}
		gists = append(gists, &g)
	}
	return gists, nil
}

