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

// 全局多音字映射表
var polyphonic = map[rune][]string{}

// 加载 polyphonic.json 文件
func loadPolyphonicDict(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("⚠️ 未找到 polyphonic.json，使用内置最小字典")
		polyphonic = map[rune][]string{
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
		return
	}

	tmp := make(map[string][]string)
	if err := json.Unmarshal(data, &tmp); err != nil {
		fmt.Println("⚠️ polyphonic.json 解析失败:", err)
		return
	}

	for k, v := range tmp {
		runes := []rune(k)
		if len(runes) > 0 {
			polyphonic[runes[0]] = v
		}
	}
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

// parseQuery —— 允许空格和后缀过滤器
// parseQuery —— 改进版，支持 "yhxx ." 这种输入
func parseQuery(raw string) Query {
	tokens := strings.Fields(raw)
	q := Query{}
	if len(tokens) == 0 {
		return q
	}

	// 默认关键字取第一个
	q.Keywords = tokens[0]

	if len(tokens) > 1 {
		last := tokens[len(tokens)-1]
		lastLower := strings.ToLower(last)

		// 如果最后一个是单独的 "."，忽略掉
		if last == "." {
			q.Keywords = strings.Join(tokens[:len(tokens)-1], " ")
			return q
		}

		// 如果最后一个是过滤器
		if lastLower == "dir" || lastLower == "file" || (strings.HasPrefix(lastLower, ".") && len(lastLower) > 1) {
			q.FileType = lastLower
			q.Keywords = strings.Join(tokens[:len(tokens)-1], " ")
			return q
		}

		// 否则拼接所有 token
		q.Keywords = strings.Join(tokens, " ")
	}
	return q
}

// ---------------- 配置 ----------------
func getConfig() ([]string, []string, int, int) {
	homeDir, _ := os.UserHomeDir()

	dirEnv := os.Getenv("SEARCH_DIRS")
	var dirs []string
	if dirEnv != "" {
		for _, d := range strings.Split(dirEnv, ",") {
			dirs = append(dirs, strings.TrimSpace(d))
		}
	} else {
		dirs = []string{"Documents", "Desktop", "Downloads"}
	}

	exclEnv := os.Getenv("EXCLUDES")
	var excl []string
	if exclEnv != "" {
		for _, e := range strings.Split(exclEnv, ",") {
			excl = append(excl, strings.TrimSpace(e))
		}
	} else {
		excl = []string{".git", "__pycache__", "node_modules", ".DS_Store"}
	}

	maxRes := 100
	if os.Getenv("MAX_RESULTS") != "" {
		fmt.Sscanf(os.Getenv("MAX_RESULTS"), "%d", &maxRes)
	}

	maxDepth := -1
	if os.Getenv("MAX_DEPTH") != "" {
		fmt.Sscanf(os.Getenv("MAX_DEPTH"), "%d", &maxDepth)
	}

	var wl []string
	for _, d := range dirs {
		full := filepath.Join(homeDir, d)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			wl = append(wl, full)
		}
	}
	return wl, excl, maxRes, maxDepth
}

// ---------------- 匹配逻辑 ----------------
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

func fuzzyMatchAllowOneError(query, target string) bool {
	m, n := len(query), len(target)
	if abs(m-n) > 1 {
		return false
	}
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if query[i-1] == target[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min3(dp[i-1][j-1]+1, dp[i-1][j]+1, dp[i][j-1]+1)
			}
		}
	}
	return dp[m][n] <= 1
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func matchScore(query, name string, pc *PinyinCache) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	scores := []int{}

	if looseMatch(q, nameLower) {
		scores = append(scores, 300)
	}

	full, initials := pc.Get(name)
	if looseMatch(q, full) {
		scores = append(scores, 200)
	} else if retryPolyphonicMatch(q, name, full) {
		scores = append(scores, 170)
	} else if fuzzyMatchAllowOneError(q, full) {
		scores = append(scores, 140)
	}

	if looseMatch(q, initials) {
		scores = append(scores, 150)
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

func searchDir(base string, baseDepth int, query Query, pc *PinyinCache, excludes map[string]bool, maxDepth int, resultChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
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
	Valid    bool   `json:"valid"`
	Icon     struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"icon"`
}

func main() {
	loadPolyphonicDict("polyphonic.json")

	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}
	query := parseQuery(os.Args[1])

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
	if len(results) == 0 {
		item := AlfredItem{
			Title:    "没有找到匹配结果",
			Subtitle: "请尝试调整关键词或目录设置",
			Valid:    false,
		}
		item.Icon.Type = "icon"
		item.Icon.Path = "icon.png"
		items = append(items, item)
	} else {
		for _, r := range results {
			item := AlfredItem{
				Uid:   r.Path,
				Title: r.Name,
				Arg:   r.Path,
				Valid: true,
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
