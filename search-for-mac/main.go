package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mozillazg/go-pinyin"
)

// ---------------- 拼音缓存（支持多音字） ----------------
var a = pinyin.NewArgs()

type PinyinCache struct {
	mu    sync.RWMutex
	cache map[string][2]string
}

func NewPinyinCache() *PinyinCache {
	return &PinyinCache{cache: make(map[string][2]string)}
}

func (pc *PinyinCache) GetAll(name string) ([]string, []string) {
	pc.mu.RLock()
	if val, ok := pc.cache[name]; ok {
		pc.mu.RUnlock()
		return strings.Split(val[0], ","), strings.Split(val[1], ",")
	}
	pc.mu.RUnlock()

	// 全拼矩阵
	args := pinyin.NewArgs()
	pyMatrix := pinyin.Pinyin(name, args)
	fullList := combinePinyin(pyMatrix)

	// 首字母矩阵（修正点）
	args.Style = pinyin.FirstLetter
	pyMatrix2 := pinyin.Pinyin(name, args)
	initList := combinePinyin(pyMatrix2)

	pc.mu.Lock()
	pc.cache[name] = [2]string{
		strings.Join(fullList, ","),
		strings.Join(initList, ","),
	}
	pc.mu.Unlock()

	return fullList, initList
}

// 组合函数保持不变
func combinePinyin(matrix [][]string) []string {
	results := []string{""}
	for _, choices := range matrix {
		var newResults []string
		for _, base := range results {
			for _, p := range choices {
				newResults = append(newResults, base+p)
			}
		}
		results = newResults
	}
	return results
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

	// 文件名直配优先
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

	// 拼音匹配（支持多音字）
	fullList, initList := pc.GetAll(name)

	for _, full := range fullList {
		if fuzzyMatch(q, full) {
			scores = append(scores, 200-abs(len(full)-len(q)))
			break
		}
	}

	for _, initials := range initList {
		if fuzzyMatch(q, initials) {
			scores = append(scores, 150-abs(len(initials)-len(q)))
			break
		}
	}

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
	Score   int
	Path    string
	Name    string
	IsDir   bool
	ModTime time.Time
	Size    int64
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
			info, _ := os.Stat(path)
			resultChan <- Result{
				Score:   score,
				Path:    path,
				Name:    name,
				IsDir:   d.IsDir(),
				ModTime: info.ModTime(),
				Size:    info.Size(),
			}
		}
		return nil
	})
}

// ---------------- Alfred 输出 ----------------
type AlfredItem struct {
	Uid          string `json:"uid"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Arg          string `json:"arg"`
	Quicklookurl string `json:"quicklookurl"`
	Icon         struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"icon"`
}

// ---------------- 主函数 ----------------
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
	seen := make(map[string]bool) // 去重

	for r := range resultChan {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		results = append(results, r)
	}

	// 排序 + 权重优化
	sort.Slice(results, func(i, j int) bool {
		si, sj := results[i].Score, results[j].Score

		// 最近修改加权
		if results[i].ModTime.After(time.Now().AddDate(0, 0, -30)) {
			si += 50
		}
		if results[j].ModTime.After(time.Now().AddDate(0, 0, -30)) {
			sj += 50
		}

		// 类型优先
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

		// 扩展名优先
		if strings.HasPrefix(query.FileType, ".") {
			iMatch := strings.HasSuffix(strings.ToLower(results[i].Path), query.FileType)
			jMatch := strings.HasSuffix(strings.ToLower(results[j].Path), query.FileType)
			if iMatch != jMatch {
				return iMatch
			}
		}

		return si > sj
	})

	if len(results) > maxRes {
		results = results[:maxRes]
	}

	items := []AlfredItem{}
	for _, r := range results {
		item := AlfredItem{
			Uid:          r.Path,
			Title:        r.Name,
			Arg:          r.Path,
			Quicklookurl: r.Path,
		}

		// Subtitle 优化
		parent := filepath.Dir(r.Path)
		if r.IsDir {
			item.Subtitle = fmt.Sprintf("📂 文件夹 | %s", parent)
		} else {
			item.Subtitle = fmt.Sprintf("📄 文件 | %s | %.1fKB | 修改: %s",
				parent, float64(r.Size)/1024,
				r.ModTime.Format("2006-01-02 15:04"))
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
