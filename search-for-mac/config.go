package main

import (
	"os"
	"path/filepath"
	"strings"
)

func getConfig() ([]string, []string, int, int) {
	homeDir, _ := os.UserHomeDir()
	dirs := []string{"Documents", "Desktop", "Downloads"}
	if env := os.Getenv("SEARCH_DIRS"); env != "" {
		dirs = strings.Split(env, ",")
	}

	// 转成绝对路径
	fullDirs := []string{}
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if !filepath.IsAbs(d) {
			d = filepath.Join(homeDir, d)
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			fullDirs = append(fullDirs, d)
		}
	}

	excludes := []string{".git", "__pycache__", "node_modules", ".DS_Store"}
	if env := os.Getenv("EXCLUDES"); env != "" {
		excludes = strings.Split(env, ",")
	}

	maxRes := 100
	maxDepth := -1
	return fullDirs, excludes, maxRes, maxDepth
}
