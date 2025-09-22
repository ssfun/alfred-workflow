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

// ---------------- 多音字字典 ----------------
var polyphonic = map[rune][]string{}

func loadPolyphonicDict(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
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
	if err := json.Unmarshal(data, &tmp); err == nil {
		for k, v := range tmp {
			runes := []rune(k)
			if len(runes) > 0 {
				polyphonic[runes[0]] = v
			}
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

	var fullParts []string
	var initialParts []string

	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			// 多音字优先使用字典配置
			if alts, ok := polyphonic[r]; ok && len(alts) > 0 {
				choose := alts[0]
				fullParts = append(fullParts, choose)
				initialParts = append(initialParts, string(choose[0]))
			} else {
				py := pinyin.LazyPinyin(string(r), a)
				if len(py) > 0 {
					fullParts = append(fullParts, py[0])
					initialParts = append(initialParts, string(py[0][0]))
				}
			}
		}
	}
	full := strings.Join(fullParts, "")
	initials := strings.Join(initialParts, "")

	pc.mu.Lock()
	pc.cache[name] = [2]string{full, initials}
	pc.mu.Unlock()
	return full, initials
}

// ---------------- 工具函数 ----------------
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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

// ---------------- 多音字重试 ----------------
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
	parts := []string{}
	for i, r := range runes {
		if i == idx {
			parts = append(parts, alt)
		} else {
			py := pinyin.LazyPinyin(string(r), a)
			if len(py) > 0 {
				parts = append(parts, py[0])
			}
		}
	}
	return strings.Join(parts, "")
}

// ---------------- Query ----------------
type Query struct {
	Keywords string
	FileType string
}

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
	// 宽松处理
	if strings.HasSuffix(q.Keywords, ".") {
		queries = append(queries, Query{Keywords: strings.TrimSuffix(q.Keywords, "."), FileType: q.FileType})
	}
	if tokens[len(tokens)-1] == "." && len(tokens) > 1 {
		queries = append(queries, Query{Keywords: strings.Join(tokens[:len(tokens)-1], " "), FileType: q.FileType})
	}
	return queries
}

// ---------------- 配置 ----------------
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

// ---------------- 文件类型过滤 ----------------
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

// ---------------- 打分 ----------------
func matchScore(query, name string, pc *PinyinCache) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	score := 0
	debug := os.Getenv("DEBUG") == "1"

	// 英文文件
	if isASCII(name) && !containsChinese(name) {
		if nameLower == q {
			return 500
		}
		if strings.HasPrefix(nameLower, q) {
			return 450
		}
		if strings.Contains(nameLower, q) {
			return 400
		}
		return 0
	}

	// 中文文件
	full, initials := pc.Get(name)

	// 首字母优先
	if strings.EqualFold(q, initials) {
		score = max(score, 380)
	} else if looseMatch(q, initials) {
		score = max(score, 250)
	}

	// 全拼
	if strings.EqualFold(q, full) {
		score = max(score, 350)
	} else if looseMatch(q, full) {
		score = max(score, 120)
	}

	// 多音字
	if retryPolyphonicMatch(q, name, full) {
		score = max(score, 120)
	}

	// Fuzzy 容错
	if len(q) >= 3 && abs(len(q)-len(full)) <= 1 && fuzzyMatchAllowOneError(q, full) {
		score = max(score, 100)
	}

	if debug && score > 0 {
		fmt.Fprintln(os.Stderr, "DEBUG:", name, "→ q:", q, "full:", full, "initials:", initials, "score:", score)
	}
	return score
}

// ---------------- 文件大小 ----------------
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(1024), 0
	for n := size / 1024; n >= 1024 && exp < 2; n /= 1024 {
		div *= 1024
		exp++
	}
	value := float64(size) / float64(div)
	switch exp {
	case 0:
		return fmt.Sprintf("%.1fKB", value)
	case 1:
		return fmt.Sprintf("%.1fMB", value)
	case 2:
		return fmt.Sprintf("%.1fGB", value)
	}
	return fmt.Sprintf("%.1fTB", float64(size)/float64(1024*1024*1024*1024))
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

// ---------------- main ----------------
func main() {
	loadPolyphonicDict("polyphonic.json")
	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}
	queries := parseQueryV2(os.Args[1])
	dirs, excludesList, maxRes, _ := getConfig()
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
		item.Icon.Path = "alert.png"
		items = append(items, item)
	} else {
		for _, r := range results {
			item := AlfredItem{Uid: r.Path, Title: r.Name, Arg: r.Path, Valid: true}
			parent := filepath.Dir(r.Path)
			if r.IsDir {
				item.Subtitle = fmt.Sprintf("%s", parent)
			} else {
				item.Subtitle = fmt.Sprintf("%s | %s | 修改: %s",
					parent, formatSize(r.Size), r.ModTime.Format("2006-01-02 15:04"))
			}
			item.Icon.Type = "fileicon"
			item.Icon.Path = r.Path
			items = append(items, item)
		}
	}
	data, _ := json.Marshal(map[string]interface{}{"items": items})
	fmt.Println(string(data))
}
