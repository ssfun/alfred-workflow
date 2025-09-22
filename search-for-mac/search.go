package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Result struct {
	Score   int
	Path    string
	Name    string
	IsDir   bool
	ModTime time.Time
	Size    int64
}

// query结构
type Query struct {
	Keywords string
	FileType string
}

// 查询解析
func parseQueryV2(raw string) []Query {
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return []Query{}
	}
	var queries []Query
	q := Query{Keywords: strings.Join(tokens, " ")}
	if len(tokens) > 1 {
		last := strings.ToLower(tokens[len(tokens)-1])
		if last == "dir" || last == "file" || (strings.HasPrefix(last, ".") && len(last) > 1) {
			q.FileType = last
			q.Keywords = strings.Join(tokens[:len(tokens)-1], " ")
		}
	}
	queries = append(queries, q)

	if strings.HasSuffix(q.Keywords, ".") {
		queries = append(queries, Query{Keywords: strings.TrimSuffix(q.Keywords, "."), FileType: q.FileType})
	}
	return queries
}

// 文件类型过滤
func typeFilter(path string, isDir bool, fileType string) bool {
	if fileType == "" {
		return true
	}
	if fileType == "dir" {
		return isDir
	}
	if fileType == "file" {
		return !isDir
	}
	if strings.HasPrefix(fileType, ".") {
		return strings.HasSuffix(strings.ToLower(path), fileType)
	}
	return true
}

// 扫描单个目录
func searchDirOnce(base string, queries []Query, pc *PinyinCache, excludes map[string]bool, resultChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || excludes[name] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		for _, q := range queries {
			if !typeFilter(path, d.IsDir(), q.FileType) {
				continue
			}
			score := matchScore(q.Keywords, name, pc)
			if score > 0 {
				info, _ := os.Stat(path)
				resultChan <- Result{score, path, name, d.IsDir(), info.ModTime(), info.Size()}
			}
		}
		return nil
	})
}

// 并发搜索入口
func RunSearch(dirs []string, excludesList []string, queries []Query, maxRes int) []Result {
	excludesMap := map[string]bool{}
	for _, e := range excludesList {
		excludesMap[e] = true
	}

	pc := NewPinyinCache()
	resultChan := make(chan Result, 2000)
	var wg sync.WaitGroup

	for _, d := range dirs {
		wg.Add(1)
		go searchDirOnce(d, queries, pc, excludesMap, resultChan, &wg)
	}
	go func() { wg.Wait(); close(resultChan) }()

	results := []Result{}
	seen := map[string]int{}
	for r := range resultChan {
		if prev, ok := seen[r.Path]; !ok || r.Score > prev {
			seen[r.Path] = r.Score
			results = append(results, r)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ModTime.After(results[j].ModTime)
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > maxRes {
		return results[:maxRes]
	}
	return results
}
