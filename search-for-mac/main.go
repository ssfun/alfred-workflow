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

var a = pinyin.NewArgs()

// ---------------- 常见多音字表 ----------------
var polyphonic = map[rune][]string{
	'行': {"hang", "xing"},
	'长': {"chang", "zhang"},
	'重': {"chong", "zhong"},
	'乐': {"le", "yue"},
	'处': {"chu", "cu"},
	'还': {"hai", "huan"},
	'藏': {"cang", "zang"},
	'假': {"jia", "jie"},
	'召': {"zhao", "shao"},
}

// ---------------- 拼音缓存 ----------------
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

// ---------------- 多音字重试逻辑 ----------------
func retryPolyphonicMatch(query string, name string, full string) bool {
	runes := []rune(name)
	for i, r := range runes {
		if alts, ok := polyphonic[r]; ok {
			for _, alt := range alts {
				altFull := rebuildPinyin(runes, i, alt)
				if looseMatch(query, altFull) {
					return true
				}
			}
		}
	}
	return false
}

func rebuildPinyin(runes []rune, idx int, alt string) string {
	args := pinyin.NewArgs()
	parts := []string{}
	for i, r := range runes {
		if i == idx {
			parts = append(parts, alt)
		} else {
			py := pinyin.LazyPinyin(string(r), args)
			if len(py) > 0 {
				parts = append(parts, py[0])
			}
		}
	}
	return strings.Join(parts, "")
}

// ---------------- 查询解析 ----------------
type Query struct {
	Keywords string
	FileType string
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

// ---------------- 配置 ----------------
func getConfig() ([]string, []string, int, int) {
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

	// 最大扫描深度
	maxDepth := -1 // -1 表示无限制
	if os.Getenv("MAX_DEPTH") != "" {
		fmt.Sscanf(os.Getenv("MAX_DEPTH"), "%d", &maxDepth)
	}

	// 白名单完整路径
	var wl []string
	for _, d := range dirs {
		full := filepath.Join(homeDir, d)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			wl = append(wl, full)
		}
	}

	return wl, excl, maxRes, maxDepth
}

// ---------------- 匹配算法 ----------------
func looseMatch(query, target string) bool {
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

	// 文件名直配
	if looseMatch(q, nameLower) {
		pos := strings.Index(nameLower, q)
		if nameLower == q {
			scores = append(scores, 500)
		} else if pos == 0 {
			scores = append(scores, 400)
		} else {
			scores = append(scores, 300-pos-abs(len(name)-len(q)))
		}
	}

	// 拼音
	full, initials := pc.Get(name)

	if looseMatch(q, full) {
		scores = append(scores, 200-abs(len(full)-len(q)))
	} else {
		if retryPolyphonicMatch(q, name, full) {
			scores = append(scores, 170) // 多音字重试，权重低一点
		}
	}

	if looseMatch(q, initials) {
		scores = append(scores, 150-abs(len(initials)-len(q)))
	}

	max := 0
	for _, s := range scores {
		if s > max {
			max = s
		}
	}
	return max
}

// ---------------- 搜索逻辑（WalkDir 并发主目录） ----------------
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

func searchDir(base string, baseDepth int, query Query, pc *PinyinCache, excludes map[string]bool, maxDepth int, resultChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// 深度限制
		if maxDepth > -1 {
			curDepth := strings.Count(path, string(os.PathSeparator)) - baseDepth
			if curDepth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
			}
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
	Uid      string `json:"uid"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Icon     struct {
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

	whitelistDirs, excludesList, maxRes, maxDepth := getConfig()
	excludesMap := make(map[string]bool)
	for _, e := range excludesList {
		excludesMap[e] = true
	}

	pc := NewPinyinCache()
	resultChan := make(chan Result, 2000)
	var wg sync.WaitGroup

	for _, d := range whitelistDirs {
		wg.Add(1)
		baseDepth := strings.Count(d, string(os.PathSeparator))
		go searchDir(d, baseDepth, query, pc, excludesMap, maxDepth, resultChan, &wg)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := []Result{}
	seen := make(map[string]bool)
	for r := range resultChan {
		if !seen[r.Path] {
			seen[r.Path] = true
			results = append(results, r)
		}
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		si, sj := results[i].Score, results[j].Score
		if results[i].ModTime.After(time.Now().AddDate(0, 0, -30)) {
			si += 50
		}
		if results[j].ModTime.After(time.Now().AddDate(0, 0, -30)) {
			sj += 50
		}
		return si > sj
	})
	if len(results) > maxRes {
		results = results[:maxRes]
	}

	items := []AlfredItem{}
	for _, r := range results {
		item := AlfredItem{
			Uid:   r.Path,
			Title: r.Name,
			Arg:   r.Path,
		}
		parent := filepath.Dir(r.Path)
		if r.IsDir {
			item.Subtitle = fmt.Sprintf("%s", parent)
		} else {
			item.Subtitle = fmt.Sprintf("%s | %.1fKB | 修改: %s",
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
