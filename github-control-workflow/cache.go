package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v53/github"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() error {
	cacheDir := filepath.Join(os.Getenv("HOME"), "Library", "Caches", "com.runningwithcrayons.Alfred", "com.sfun.github")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	dbPath := filepath.Join(cacheDir, "github_cache.db")
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return createTables()
}

func createTables() error {
	repoTableSQL := `
    CREATE TABLE IF NOT EXISTS repos (
        id INTEGER PRIMARY KEY, type TEXT NOT NULL, full_name TEXT, full_name_norm TEXT,
        description TEXT, desc_norm TEXT, html_url TEXT, clone_url TEXT, stars INTEGER,
        updated_at TEXT, private INTEGER, fork INTEGER, archived INTEGER, fetched_at INTEGER
    );`

	gistTableSQL := `
    CREATE TABLE IF NOT EXISTS gists (
        id TEXT PRIMARY KEY, description TEXT, desc_norm TEXT, html_url TEXT, public INTEGER,
        updated_at TEXT, files TEXT, files_norm TEXT, fetched_at INTEGER
    );`

	if _, err := db.Exec(repoTableSQL); err != nil {
		return err
	}
	if _, err := db.Exec(gistTableSQL); err != nil {
		return err
	}
	return nil
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

// --- Repos Caching ---

func saveRepos(repoType string, repos []*github.Repository) error {
	tx, err := db.Begin()
	if err != nil { return err }
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO repos
        (id, type, full_name, full_name_norm, description, desc_norm, html_url, clone_url,
         stars, updated_at, private, fork, archived, fetched_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil { return err }
	defer stmt.Close()

	now := time.Now().Unix()
	for _, r := range repos {
		_, err := stmt.Exec(r.GetID(), repoType, r.GetFullName(), normalize(r.GetFullName()),
			r.GetDescription(), normalize(r.GetDescription()), r.GetHTMLURL(), r.GetCloneURL(),
			r.GetStargazersCount(), r.GetUpdatedAt().Format(time.RFC3339),
			r.GetPrivate(), r.GetFork(), r.GetArchived(), now,
		)
		if err != nil { return err }
	}
	return tx.Commit()
}

func queryRepos(repoType, search string, limit int) ([]*github.Repository, error) {
	var rows *sql.Rows
	var err error

	normSearch := "%" + normalize(search) + "%"

	if search != "" {
		query := `SELECT id, full_name, description, html_url, clone_url, stars, updated_at, private
                  FROM repos WHERE type = ? AND (full_name_norm LIKE ? OR desc_norm LIKE ?)
                  ORDER BY datetime(updated_at) DESC LIMIT ?`
		rows, err = db.Query(query, repoType, normSearch, normSearch, limit)
	} else {
		query := `SELECT id, full_name, description, html_url, clone_url, stars, updated_at, private
                  FROM repos WHERE type = ? ORDER BY datetime(updated_at) DESC LIMIT ?`
		rows, err = db.Query(query, repoType, limit)
	}
	if err != nil { return nil, err }
	defer rows.Close()

	var repos []*github.Repository
	for rows.Next() {
		var r github.Repository
		var updatedAtStr string
		err := rows.Scan(&r.ID, &r.FullName, &r.Description, &r.HTMLURL, &r.CloneURL,
			&r.StargazersCount, &updatedAtStr, &r.Private)
		if err != nil { return nil, err }
		
		updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)
		r.UpdatedAt = &github.Timestamp{Time: updatedAt}
		repos = append(repos, &r)
	}
	return repos, nil
}

func clearRepos(repoType string) error {
	_, err := db.Exec("DELETE FROM repos WHERE type = ?", repoType)
	return err
}


// --- Gists Caching ---

func saveGists(gists []*github.Gist) error {
	tx, err := db.Begin()
	if err != nil { return err }
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO gists
        (id, description, desc_norm, html_url, public, updated_at, files, files_norm, fetched_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil { return err }
	defer stmt.Close()

	now := time.Now().Unix()
	for _, g := range gists {
		var fileNames []string
		var fileNamesNorm []string
		for fn := range g.Files {
			fileNames = append(fileNames, string(fn))
			fileNamesNorm = append(fileNamesNorm, normalize(string(fn)))
		}
		filesJSON, _ := json.Marshal(fileNames)

		_, err := stmt.Exec(g.GetID(), g.GetDescription(), normalize(g.GetDescription()),
			g.GetHTMLURL(), g.GetPublic(), g.GetUpdatedAt().Format(time.RFC3339),
			string(filesJSON), strings.Join(fileNamesNorm, " "), now,
		)
		if err != nil { return err }
	}
	return tx.Commit()
}

func queryGists(search string, limit int) ([]*github.Gist, error) {
	var rows *sql.Rows
	var err error

	normSearch := "%" + normalize(search) + "%"

	if search != "" {
		query := `SELECT id, description, html_url, public, updated_at, files
                  FROM gists WHERE desc_norm LIKE ? OR files_norm LIKE ?
                  ORDER BY datetime(updated_at) DESC LIMIT ?`
		rows, err = db.Query(query, normSearch, normSearch, limit)
	} else {
		query := `SELECT id, description, html_url, public, updated_at, files
                  FROM gists ORDER BY datetime(updated_at) DESC LIMIT ?`
		rows, err = db.Query(query, limit)
	}

	if err != nil { return nil, err }
	defer rows.Close()

	var gists []*github.Gist
	for rows.Next() {
		var g github.Gist
		var filesJSON string
		var updatedAtStr string
		err := rows.Scan(&g.ID, &g.Description, &g.HTMLURL, &g.Public, &updatedAtStr, &filesJSON)
		if err != nil { return nil, err }
		
		updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)
		g.UpdatedAt = &github.Timestamp{Time: updatedAt}

		var fileNames []string
		json.Unmarshal([]byte(filesJSON), &fileNames)
		g.Files = make(map[github.GistFilename]github.GistFile)
		for _, fn := range fileNames {
			g.Files[github.GistFilename(fn)] = github.GistFile{}
		}
		
		gists = append(gists, &g)
	}
	return gists, nil
}


func clearGists() error {
	_, err := db.Exec("DELETE FROM gists")
	return err
}

// --- Cache Info ---
func getCacheInfo(tableType string) string {
	var query string
	var args []interface{}

	switch tableType {
	case "stars", "repos":
		query = "SELECT COUNT(*), MAX(fetched_at) FROM repos WHERE type = ?"
		args = []interface{}{tableType}
	case "gists":
		query = "SELECT COUNT(*), MAX(fetched_at) FROM gists"
	default:
		return "无效类型"
	}

	var count int
	var fetchedAt sql.NullInt64
	err := db.QueryRow(query, args...).Scan(&count, &fetchedAt)
	if err != nil || count == 0 {
		return "无缓存"
	}

	ts := time.Unix(fetchedAt.Int64, 0).Format("2006-01-02 15:04")
	return fmt.Sprintf("%d 项, 更新于 %s", count, ts)
}
