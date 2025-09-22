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

	// 全拼矩阵（启用多音字）
	args1 := pinyin.NewArgs()
	args1.Heteronym = true
	pyMatrix := pinyin.Pinyin(name, args1)
	fullList := combinePinyin(pyMatrix)

	// 首字母矩阵（启用多音字）
	args2 := pinyin.NewArgs()
	args2.Style = pinyin.FirstLetter
	args2.Heteronym = true
	pyMatrix2 := pinyin.Pinyin(name, args2)
	initList := combinePinyin(pyMatrix2)

	pc.mu.Lock()
	pc.cache[name] = [2]string{
		strings.Join(fullList, ","),
		strings.Join(initList, ","),
	}
	pc.mu.Unlock()

	return fullList, initList
}

// 展开多音字组合 [["yin"], ["hang","xing"], ["xin"], ["xi"]]
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

// ---------------- 配置读取 ----------------
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

// ---------------- 匹配逻辑 ----------------

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

// 编辑距离 (Levenshtein Distance)
func editDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)
	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
	}
	for i := 0; i <= len1; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, min(dp[i][j-1]+1, dp[i-1][j-1]+cost))
		}
	}
	return dp[len1][len2]
}

func matchScore(query, name string, pc *PinyinCache) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	scores := []int{}

	// 文件名直配
	if fuzzyMatch(q, nameLower) {
		scores = append(scores, 500)
	}

	// 拼音匹配（支持多音字+容错）
	fullList, initList := pc.GetAll(name)

	// 精确匹配
	for _, full := range fullList {
		if fuzzyMatch(q, full) {
			scores = append(scores, 200)
		}
	}
	for _, initials := range initList {
		if fuzzyMatch(q, initials) {
			scores = append(scores, 180)
		}
	}

	// 容错匹配（仅在未命中时触发，性能优化）
	if len(scores) == 0 {
		for _, full := range fullList {
			if editDistance(q, full) <= 2 { // 全拼容错
				scores = append(scores, 120)
				break
			}
		}
		for _, initials := range initList {
			if editDistance(q, initials) <= 1 { // 首字母容错
				scores = append(scores, 100)
				break
			}
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

func searchDir(base string, query string, pc *PinyinCache, excludes map[string]bool, wg *sync.WaitGroup, resultChan chan<- Result) {
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
		score := matchScore(query, name, pc)
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}
	query := os.Args[1]

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
	seen := make(map[string]bool)
	for r := range resultChan {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		results = append(results, r)
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
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
			item.Subtitle = fmt.Sprintf("📂 %s | %s", r.Name, parent)
		} else {
			item.Subtitle = fmt.Sprintf("📄 %s | %.1fKB | 修改:%s",
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
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
