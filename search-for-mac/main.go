package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/mozillazg/go-pinyin"
)

// ---------------- CONFIG ----------------

// 拼音缓存
var a = pinyin.NewArgs()

type PinyinCache struct {
	mu    sync.RWMutex
	cache map[string][2]string
}

func NewPinyinCache() *PinyinCache {
	return &PinyinCache{cache: make(map[string][2]string)}
}

func (pc *PinyinCache) Get(name string) (string, string) {
	pc.mu.RLock()
	if val, ok := pc.cache[name]; ok {
		pc.mu.RUnlock()
		return val[0], val[1]
	}
	pc.mu.RUnlock()

	full := strings.Join(pinyin.LazyPinyin(name, a), "")
	args := pinyin.NewArgs()
	args.Style = pinyin.FirstLetter
	initials := strings.Join(pinyin.LazyPinyin(name, args), "")

	pc.mu.Lock()
	pc.cache[name] = [2]string{full, initials}
	pc.mu.Unlock()
	return full, initials
}

// ---------------- 查询解析 ----------------
type Query struct {
	Keywords string
	FileType string // "dir" / "file" / ".ext"
}

func parseQuery(raw string) Query {
	tokens := strings.Fields(raw)
	q := Query{}
	keywords := []string{}
	for _, t := range tokens {
		low := strings.ToLower(t)
		if low == "dir" || low == "file" || strings.HasPrefix(low, ".") {
			q.FileType = low
		} else {
			keywords = append(keywords, t)
		}
	}
	q.Keywords = strings.Join(keywords, " ")
	return q
}

// ---------------- 配置读取（环境变量） ----------------
func getConfig() ([]string, []string, int) {
	homeDir, _ := os.UserHomeDir()

	// 搜索目录
	dirEnv := os.Getenv("SEARCH_DIRS")
	var dirs []string
	if dirEnv != "" {
		for _, d := range strings.Split(dirEnv, ",") {
			dirs = append(dirs, strings.TrimSpace(d))
		}
	} else {
		dirs = []string{"Documents", "Desktop", "Downloads"}
	}

	// 忽略目录
	exclEnv := os.Getenv("EXCLUDES")
	var excl []string
	if exclEnv != "" {
		for _, e := range strings.Split(exclEnv, ",") {
			excl = append(excl, strings.TrimSpace(e))
		}
	} else {
		excl = []string{".git", "__pycache__", "node_modules", ".DS_Store"}
	}

	// 最大结果数
	maxRes := 100
	if os.Getenv("MAX_RESULTS") != "" {
		fmt.Sscanf(os.Getenv("MAX_RESULTS"), "%d", &maxRes)
	}

	// 白名单完整路径
	var wl []string
	for _, d := range dirs {
		full := filepath.Join(homeDir, d)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			wl = append(wl, full)
		}
	}

	return wl, excl, maxRes
}

// ---------------- 匹配算法 ----------------
func fuzzyMatch(query, target string) bool {
	i, j := 0, 0
	for i < len(query) && j < len(target) {
		if query[i] == target[j] {
			i++
		}
		j++
	}
	return i == len(query)
}

func matchScore(query, name string, pc *PinyinCache) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	scores := []int{}

	// 文件名直配优先 - 完全匹配 / 前缀匹配加权
	if fuzzyMatch(q, nameLower) {
		pos := strings.Index(nameLower, q)
		if nameLower == q {
			scores = append(scores, 500)
		} else if pos == 0 {
			scores = append(scores, 400)
		} else {
			scores = append(scores, 300-pos-abs(len(name)-len(q)))
		}
	}

	// 拼音全拼 + 首字母
	full, initials := pc.Get(name)
	if fuzzyMatch(q, full) {
		scores = append(scores, 200-abs(len(full)-len(q)))
	}
	if fuzzyMatch(q, initials) {
		scores = append(scores, 150-abs(len(initials)-len(q)))
	}

	// 返回最大分
	max := 0
	for _, s := range scores {
		if s > max {
			max = s
		}
	}
	return max
}

// ---------------- 搜索逻辑 ----------------
type Result struct {
	Score int
	Path  string
	Name  string
	IsDir bool
}

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

func searchDir(base string, query Query, pc *PinyinCache, excludes map[string]bool, wg *sync.WaitGroup, resultChan chan<- Result) {
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
		if !typeFilter(path, d.IsDir(), query.FileType) {
			return nil
		}
		score := matchScore(query.Keywords, name, pc)
		if score > 0 {
			resultChan <- Result{
				Score: score,
				Path:  path,
				Name:  name,
				IsDir: d.IsDir(),
			}
		}
		return nil
	})
}

// ---------------- Alfred 输出 ----------------
type AlfredItem struct {
	Uid      string `json:"uid"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Icon     struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"icon"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}
	rawQuery := os.Args[1]
	query := parseQuery(rawQuery)

	whitelistDirs, excludesList, maxRes := getConfig()
	excludesMap := make(map[string]bool)
	for _, e := range excludesList {
		excludesMap[e] = true
	}

	pc := NewPinyinCache()
	resultChan := make(chan Result, 1000)
	var wg sync.WaitGroup

	for _, d := range whitelistDirs {
		wg.Add(1)
		go searchDir(d, query, pc, excludesMap, &wg, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := []Result{}
	for r := range resultChan {
		results = append(results, r)
	}

	// 排序 + 权重优化
	sort.Slice(results, func(i, j int) bool {
		si, sj := results[i].Score, results[j].Score

		// 文件夹/文件优先权重
		if query.FileType == "dir" {
			if results[i].IsDir && !results[j].IsDir {
				return true
			}
			if !results[i].IsDir && results[j].IsDir {
				return false
			}
		}
		if query.FileType == "file" {
			if !results[i].IsDir && results[j].IsDir {
				return true
			}
			if results[i].IsDir && !results[j].IsDir {
				return false
			}
		}

		// 扩展名筛选权重
		if strings.HasPrefix(query.FileType, ".") {
			iMatch := strings.HasSuffix(strings.ToLower(results[i].Path), query.FileType)
			jMatch := strings.HasSuffix(strings.ToLower(results[j].Path), query.FileType)
			if iMatch != jMatch {
				return iMatch
			}
		}

		// 默认分数优先
		return si > sj
	})

	if len(results) > maxRes {
		results = results[:maxRes]
	}

	items := []AlfredItem{}
	for _, r := range results {
		item := AlfredItem{
			Uid:      r.Path,
			Title:    r.Name,
			Subtitle: r.Path,
			Arg:      r.Path,
		}
		item.Icon.Type = "fileicon"
		item.Icon.Path = r.Path
		items = append(items, item)
	}

	data, _ := json.Marshal(map[string]interface{}{"items": items})
	fmt.Println(string(data))
}

// ---------------- 工具函数 ----------------
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
