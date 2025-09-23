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
		// 默认多音字表
		polyphonic = map[rune][]string{
			'行': {"hang", "xing"},
			'长': {"chang", "zhang"},
			'重': {"chong", "zhong"},
			'乐': {"le", "yue"},
			'处': {"chu", "cu"},
		}
		return
	}
	tmp := make(map[string][]string)
	if err := json.Unmarshal(data, &tmp); err == nil {
		for k, v := range tmp {
			if len([]rune(k)) > 0 {
				polyphonic[[]rune(k)[0]] = v
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
	var initials []string
	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			if alts, ok := polyphonic[r]; ok && len(alts) > 0 {
				fullParts = append(fullParts, alts[0])
				initials = append(initials, string(alts[0][0]))
			} else {
				py := pinyin.LazyPinyin(string(r), a)
				if len(py) > 0 {
					fullParts = append(fullParts, py[0])
					initials = append(initials, string(py[0][0]))
				}
			}
		}
	}
	full := strings.Join(fullParts, "")
	initialStr := strings.Join(initials, "")
	pc.mu.Lock()
	pc.cache[name] = [2]string{full, initialStr}
	pc.mu.Unlock()
	return full, initialStr
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

// ---------------- 打分函数（核心修复） ----------------
func matchScore(query, name string, pc *PinyinCache) int {
	if query == "" {
		return 0
	}

	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)

	// 1. 直接包含（支持中文/英文）
	if strings.Contains(nameLower, q) || strings.Contains(name, query) {
		return 400
	}

	// 2. 英文名逻辑
	if isASCII(name) {
		if nameLower == q {
			return 500
		}
		if strings.HasPrefix(nameLower, q) {
			return 450
		}
		return 0
	}

	// 3. 中文 -> 拼音逻辑
	full, initials := pc.Get(name)
	score := 0
	if strings.EqualFold(q, initials) {
		score = 380
	} else if looseMatch(q, initials) {
		score = 250
	}

	if strings.EqualFold(q, full) {
		score = 350
	} else if strings.HasPrefix(full, q) {
		score = 300
	}

	if len(q) >= 4 && fuzzyMatchAllowOneError(q, full) {
		score = 80
	}

	return score
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// ---------------- 结果结构 ----------------
type Result struct {
	Score   int
	Path    string
	Name    string
	IsDir   bool
	ModTime time.Time
	Size    int64
}

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

// ---------------- 模式枚举 ----------------
const (
	ModeScore        = "score"
	ModeModTimeDesc  = "mod_time_desc"
	ModeModTimeAsc   = "mod_time_asc"
	ModeAddTimeDesc  = "add_time_desc"
	ModeAddTimeAsc   = "add_time_asc"
	ModeFilenameAsc  = "filename_asc"
	ModeFilenameDesc = "filename_desc"
)

// ---------------- 从环境配置找到模式 ----------------
func getSortMode() string {
	keyword := os.Getenv("alfred_workflow_keyword")
	if keyword == "" {
		return ModeScore
	}
	modes := []string{
		ModeModTimeDesc,
		ModeModTimeAsc,
		ModeAddTimeDesc,
		ModeAddTimeAsc,
		ModeFilenameAsc,
		ModeFilenameDesc,
	}
	for _, m := range modes {
		if os.Getenv(m) == keyword { // Alfred 配置里：变量名=模式，值=keyword
			return m
		}
	}
	return ModeScore
}

// ---------------- main ----------------
func main() {
	loadPolyphonicDict("polyphonic.json")

	mode := getSortMode()

	// 搜索词
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.TrimSpace(os.Args[1]))
	}

	// 搜索目录
	searchDir := os.Getenv("SEARCH_DIR")
	if searchDir == "" {
		home, _ := os.UserHomeDir()
		searchDir = filepath.Join(home, "Downloads")
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		fmt.Println(`{"items":[{"title":"目录错误","subtitle":"无法访问搜索目录","valid":false}]}`)
		return
	}

	pc := NewPinyinCache()
	results := []Result{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}

		// 文件类型过滤
		if !fileTypeFilter(e, info) {
			continue
		}

		score := matchScore(query, e.Name(), pc)
		if query == "" {
			score = 100 // 没输入 query 时，全部展示
		}
		if score > 0 {
			results = append(results, Result{
				Score:   score,
				Path:    filepath.Join(searchDir, e.Name()),
				Name:    e.Name(),
				IsDir:   e.IsDir(),
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})
		}
	}

	// 排序：先比分数，再按 mode
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		switch mode {
		case ModeModTimeDesc:
			return results[i].ModTime.After(results[j].ModTime)
		case ModeModTimeAsc:
			return results[i].ModTime.Before(results[j].ModTime)
		case ModeFilenameAsc:
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		case ModeFilenameDesc:
			return strings.ToLower(results[i].Name) > strings.ToLower(results[j].Name)
		default:
			return results[i].ModTime.After(results[j].ModTime)
		}
	})

	// 输出 JSON 给 Alfred
	items := []AlfredItem{}
	if len(results) == 0 {
		item := AlfredItem{
			Title:    "没有找到匹配结果",
			Subtitle: "请尝试调整关键词或目录",
			Valid:    false,
		}
		item.Icon.Type = "icon"
		item.Icon.Path = "alert.png"
		items = append(items, item)
	} else {
		for _, r := range results {
			item := AlfredItem{
				Uid:   r.Path,
				Title: r.Name,
				Arg:   r.Path,
				Valid: true,
			}
			// Subtitle 只显示大小 + 修改时间
			item.Subtitle = fmt.Sprintf("%d bytes | 修改时间: %s",
				r.Size,
				r.ModTime.Format("2006-01-02 15:04"))
			item.Icon.Type = "fileicon"
			item.Icon.Path = r.Path
			items = append(items, item)
		}
	}

	data, _ := json.Marshal(map[string]interface{}{"items": items})
	fmt.Println(string(data))
}

// ---------------- 文件类型过滤 ----------------
func fileTypeFilter(entry os.DirEntry, info os.FileInfo) bool {
	ft := strings.ToLower(os.Getenv("SEARCH_FILETYPE")) // 例如: dir, file, .pdf, .png
	if ft == "" {
		return true
	}
	switch ft {
	case "dir":
		return entry.IsDir()
	case "file":
		return !entry.IsDir()
	default:
		return strings.HasSuffix(strings.ToLower(info.Name()), ft)
	}
}
