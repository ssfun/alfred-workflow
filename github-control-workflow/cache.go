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
	_ "modernc.org/sqlite" // 导入 sqlite 驱动
)

var (
	cacheDir = filepath.Join(os.Getenv("HOME"), "Library/Caches/com.runningwithcrayons.Alfred/com.sfun.github")
	dbPath   = filepath.Join(cacheDir, "github_cache.db")
)

// initDB 初始化数据库连接并创建表
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

// createTables 定义并执行建表语句
func createTables(db *sql.DB) error {
	repoTable := `
    CREATE TABLE IF NOT EXISTS repos (
        id INTEGER PRIMARY KEY,
        type TEXT NOT NULL,
        full_name TEXT,
        full_name_norm TEXT,
        description TEXT,
        desc_norm TEXT,
        html_url TEXT,
        clone_url TEXT,
        stars INTEGER,
        updated_at TEXT,
        private INTEGER,
        fork INTEGER,
        archived INTEGER,
        fetched_at INTEGER
    );`
	gistTable := `
    CREATE TABLE IF NOT EXISTS gists (
        id TEXT PRIMARY KEY,
        description TEXT,
        desc_norm TEXT,
        html_url TEXT,
        public INTEGER,
        updated_at TEXT,
        files TEXT,
        files_norm TEXT,
        fetched_at INTEGER
    );`
	if _, err := db.Exec(repoTable); err != nil {
		return err
	}
	if _, err := db.Exec(gistTable); err != nil {
		return err
	}
	return nil
}

// clearCache 清空指定类型的缓存
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
		_, err = db.Exec("DELETE FROM repos")
		if err == nil {
			_, err = db.Exec("DELETE FROM gists")
		}
	default:
		return fmt.Errorf("unknown cache type: %s", cacheType)
	}
	return err
}

// getCacheInfo 获取缓存信息（数量和更新时间）
func getCacheInfo(cacheType string) string {
	db, err := initDB()
	if err != nil {
		return fmt.Sprintf("无法连接数据库: %s", err)
	}
	defer db.Close()

	var query string
	switch cacheType {
	case "stars", "repos":
		query = fmt.Sprintf("SELECT COUNT(*), MAX(fetched_at) FROM repos WHERE type = '%s'", cacheType)
	case "gists":
		query = "SELECT COUNT(*), MAX(fetched_at) FROM gists"
	default:
		return "未知的缓存类型"
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

// saveRepos 缓存仓库列表（包括 stars 和 repos）
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
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, r := range repos {
		desc := r.GetDescription()
		_, err := stmt.Exec(
			r.GetID(), repoType, r.GetFullName(), normalize(r.GetFullName()),
			desc, normalize(desc), r.GetHTMLURL(), r.GetCloneURL(),
			r.GetStargazersCount(), r.GetUpdatedAt().Format(time.RFC3339),
			r.GetPrivate(), r.GetFork(), r.GetArchived(), now,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// queryRepos 从缓存中查询仓库
func queryRepos(repoType, searchQuery string) ([]*github.Repository, error) {
	// ... 实现查询逻辑 ...
	return nil, nil // 占位
}

// saveGists 缓存 Gist 列表
func saveGists(gists []*github.Gist) error {
	// ... 实现 Gist 保存逻辑 ...
	return nil // 占位
}

// queryGists 从缓存中查询 Gist
func queryGists(searchQuery string) ([]*github.Gist, error) {
	// ... 实现 Gist 查询逻辑 ...
	return nil, nil // 占位
}
