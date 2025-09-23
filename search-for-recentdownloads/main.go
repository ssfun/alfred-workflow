package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
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

// ---------------- 工具函数 ----------------
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

func abs(x int) int { if x < 0 { return -x }; return x }
func min3(a, b, c int) int {
	if a < b {
		if a < c { return a }
		return c
	}
	if b < c { return b }
	return c
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 { return false }
	}
	return true
}

// ---------------- 文件大小格式化 ----------------
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	if gb < 1024 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	tb := gb / 1024
	return fmt.Sprintf("%.1f TB", tb)
}

// ---------------- 打分函数 ----------------
func matchScore(query, name string, pc *PinyinCache) int {
	if query == "" { return 0 }

	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, q) || strings.Contains(name, query) {
		return 400
	}

	if isASCII(name) {
		if nameLower == q {
			return 500
		}
		if strings.HasPrefix(nameLower, q) {
			return 450
		}
		return 0
	}

	full, initials := pc.Get(name)
	if strings.EqualFold(q, initials) {
		return 380
	} else if looseMatch(q, initials) {
		return 250
	}
	if strings.EqualFold(q, full) {
		return 350
	} else if strings.HasPrefix(full, q) {
		return 300
	}
	if len(q) >= 4 && fuzzyMatchAllowOneError(q, full) {
		return 80
	}
	return 0
}

// ---------------- 结果结构 ----------------
type Result struct {
	Score      int
	Path       string
	Name       string
	IsDir      bool
	ModTime    time.Time
	Size       int64
	CreateTime time.Time
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

// ---------------- 创建时间 (macOS Birthtime) ----------------
func getCreateTime(info os.FileInfo) time.Time {
	stat := info.Sys().(*syscall.Stat_t)
	sec := stat.Birthtimespec.Sec
	nsec := stat.Birthtimespec.Nsec
	return time.Unix(sec, nsec)
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

// ---------------- 排序模式 ----------------
func getSortMode() string {
	keyword := os.Getenv("alfred_workflow_keyword")
	if keyword == "" { return ModeScore }
	modes := []string{
		ModeModTimeDesc,
		ModeModTimeAsc,
		ModeAddTimeDesc,
		ModeAddTimeAsc,
		ModeFilenameAsc,
		ModeFilenameDesc,
	}
	for _, m := range modes {
		if os.Getenv(m) == keyword {
			return m
		}
	}
	return ModeScore
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

// ---------------- 解析 query (.dir/.file/.pdf 任意位置) ----------------
func parseQueryArgs() (query string, fileType string) {
	if len(os.Args) <= 1 {
		return "", ""
	}
	raw := strings.TrimSpace(os.Args[1])
	parts := strings.Fields(raw)

	queryParts := []string{}
	for _, p := range parts {
		lp := strings.ToLower(p)
		if lp == ".dir" {
			fileType = "dir"
		} else if lp == ".file" {
			fileType = "file"
		} else if strings.HasPrefix(lp, ".") {
			fileType = lp
		} else {
			queryParts = append(queryParts, lp)
		}
	}
	return strings.Join(queryParts, " "), fileType
}

// expandPath 展开 ~ 为用户主目录
func expandPath(path string) string {
    if strings.HasPrefix(path, "~") {
        home, err := os.UserHomeDir()
        if err != nil {
            return path
        }
        if path == "~" {
            return home
        }
        // 去掉 "~/"，拼接
        return filepath.Join(home, path[2:])
    }
    return path
}

// ---------------- main ----------------
func main() {
    loadPolyphonicDict("polyphonic.json")
    mode := getSortMode()

    query, fileType := parseQueryArgs()

    searchDir := os.Getenv("SEARCH_DIR")
    if searchDir == "" {
        home, _ := os.UserHomeDir()
        searchDir = filepath.Join(home, "Downloads")
    } else {
        searchDir = expandPath(searchDir) // ✅ 支持 ~
    }

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		fmt.Println(`{"items":[{"title":"目录错误","subtitle":"无法访问搜索目录","valid":false}]}`)
		return
	}

	pc := NewPinyinCache()
	results := []Result{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") { continue }
		info, err := e.Info()
		if err != nil { continue }

		if !typeFilter(filepath.Join(searchDir, e.Name()), e.IsDir(), fileType) {
			continue
		}

		score := matchScore(query, e.Name(), pc)
		if query == "" { score = 100 }
		if score > 0 {
			results = append(results, Result{
				Score:      score,
				Path:       filepath.Join(searchDir, e.Name()),
				Name:       e.Name(),
				IsDir:      e.IsDir(),
				ModTime:    info.ModTime(),
				Size:       info.Size(),
				CreateTime: getCreateTime(info), // ✅ 真正使用 Birthtime
			})
		}
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		switch mode {
		case ModeModTimeDesc:
			return results[i].ModTime.After(results[j].ModTime)
		case ModeModTimeAsc:
			return results[i].ModTime.Before(results[j].ModTime)
		case ModeAddTimeDesc:
			return results[i].CreateTime.After(results[j].CreateTime)
		case ModeAddTimeAsc:
			return results[i].CreateTime.Before(results[j].CreateTime)
		case ModeFilenameAsc:
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		case ModeFilenameDesc:
			return strings.ToLower(results[i].Name) > strings.ToLower(results[j].Name)
		default:
			return results[i].ModTime.After(results[j].ModTime)
		}
	})

	// 输出
	items := []AlfredItem{}
	if len(results) == 0 {
		item := AlfredItem{
			Title:    "没有找到匹配结果",
			Subtitle: "请尝试调整关键词或目录",
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

			var timeLabel string
			switch mode {
			case ModeAddTimeAsc, ModeAddTimeDesc:
				timeLabel = fmt.Sprintf("添加时间: %s", r.CreateTime.Format("2006-01-02 15:04"))
			default: // 包括 ModTime 和 Filename 模式
				timeLabel = fmt.Sprintf("修改时间: %s", r.ModTime.Format("2006-01-02 15:04"))
			}

			item.Subtitle = fmt.Sprintf("%s | %s",
				formatSize(r.Size), timeLabel)

			item.Icon.Type = "fileicon"
			item.Icon.Path = r.Path
			items = append(items, item)
		}
	}

	data, _ := json.Marshal(map[string]interface{}{"items": items})
	fmt.Println(string(data))
}
