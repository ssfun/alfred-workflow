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
func getConfig() ([]string, []string, int, int, int) {
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

	// worker 数量
	workers := 8
	if os.Getenv("WORKERS") != "" {
		fmt.Sscanf(os.Getenv("WORKERS"), "%d", &workers)
	}

	// 白名单完整路径
	var wl []string
	for _, d := range dirs {
		full := filepath.Join(homeDir, d)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			wl = append(wl, full)
		}
	}

	return wl, excl, maxRes, maxDepth, workers
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

	full, initials := pc.Get(name)
	if looseMatch(q, full) {
		scores = append(scores, 200-abs(len(full)-len(q)))
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

// ---------------- Worker Pool ----------------
type Task struct {
	Path  string
	Depth int
}

func worker(id int, wg *sync.WaitGroup, tasks <-chan Task, query Query, pc *PinyinCache, excludes map[string]bool, baseDepth, maxDepth int, resultChan chan<- Result) {
	defer wg.Done()
	for task := range tasks {
		entries, err := os.ReadDir(task.Path)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || excludes[name] {
				continue
			}

			fullPath := filepath.Join(task.Path, name)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			// 匹配
			if typeFilter(fullPath, entry.IsDir(), query.FileType) {
				score := matchScore(query.Keywords, name, pc)
				if score > 0 {
					resultChan <- Result{
						Score:   score,
						Path:    fullPath,
						Name:    name,
						IsDir:   entry.IsDir(),
						ModTime: info.ModTime(),
						Size:    info.Size(),
					}
				}
			}

			// 目录且深度允许 -> 加入队列
			if entry.IsDir() {
				if maxDepth == -1 || task.Depth+1 <= maxDepth {
					// 投递新任务
					go func(p string, depth int) {
						// 投递任务（防止阻塞，可以用 select 检查，但这里保证容量足够）
						tasks <- Task{Path: p, Depth: depth}
					}(fullPath, task.Depth+1)
				}
			}
		}
	}
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

	whitelistDirs, excludesList, maxRes, maxDepth, workerCount := getConfig()
	excludesMap := make(map[string]bool)
	for _, e := range excludesList {
		excludesMap[e] = true
	}

	pc := NewPinyinCache()
	resultChan := make(chan Result, 5000)
	tasks := make(chan Task, 1000) // 任务队列

	var wg sync.WaitGroup

	// 启动 worker
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(i, &wg, tasks, query, pc, excludesMap, 0, maxDepth, resultChan)
	}

	// 投递初始任务
	for _, d := range whitelistDirs {
		tasks <- Task{Path: d, Depth: 0}
	}

	// 等待完成
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
