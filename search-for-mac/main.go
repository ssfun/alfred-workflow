package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mozillazg/go-pinyin"
)

// ---------------- 配置 ----------------
type Config struct {
	SearchDirs      []string
	Excludes        []string
	MaxResults      int
	MaxDepth        int
	MaxCombinations int
	EnableFuzzy     bool
}

func loadConfig() Config {
	homeDir, _ := os.UserHomeDir()
	cfg := Config{
		SearchDirs:      []string{"Documents", "Desktop", "Downloads"},
		Excludes:        []string{".git", "node_modules", "__pycache__", ".DS_Store"},
		MaxResults:      100,
		MaxDepth:        3,
		MaxCombinations: 10,
		EnableFuzzy:     true,
	}

	if v := os.Getenv("SEARCH_DIRS"); v != "" {
		cfg.SearchDirs = strings.Split(v, ",")
	}
	if v := os.Getenv("EXCLUDES"); v != "" {
		cfg.Excludes = strings.Split(v, ",")
	}
	if v := os.Getenv("MAX_RESULTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxResults = n
		}
	}
	if v := os.Getenv("MAX_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxDepth = n
		}
	}
	if v := os.Getenv("MAX_COMBINATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxCombinations = n
		}
	}
	if v := os.Getenv("ENABLE_FUZZY"); v != "" {
		cfg.EnableFuzzy = (strings.ToLower(v) == "true" || v == "1")
	}

	// 转换为绝对目录
	absDirs := []string{}
	for _, d := range cfg.SearchDirs {
		full := filepath.Join(homeDir, strings.TrimSpace(d))
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			absDirs = append(absDirs, full)
		}
	}
	cfg.SearchDirs = absDirs
	return cfg
}

// ---------------- 拼音缓存（支持多音字） ----------------
type PinyinCache struct {
	mu    sync.RWMutex
	cache map[string][2]string
}

func NewPinyinCache() *PinyinCache {
	return &PinyinCache{cache: make(map[string][2]string)}
}

func (pc *PinyinCache) GetAll(name string, maxComb int) ([]string, []string) {
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
	fullList := combineLimited(pyMatrix, maxComb)

	// 首字母矩阵（启用多音字）
	args2 := pinyin.NewArgs()
	args2.Style = pinyin.FirstLetter
	args2.Heteronym = true
	pyMatrix2 := pinyin.Pinyin(name, args2)
	initList := combineLimited(pyMatrix2, maxComb)

	pc.mu.Lock()
	pc.cache[name] = [2]string{
		strings.Join(fullList, ","),
		strings.Join(initList, ","),
	}
	pc.mu.Unlock()
	return fullList, initList
}

// 限制组合数量，避免爆炸
func combineLimited(matrix [][]string, maxComb int) []string {
	results := []string{""}
	for _, choices := range matrix {
		if len(choices) > 2 {
			choices = choices[:2] // 每个字最多取2个拼音
		}
		var newResults []string
		for _, base := range results {
			for _, p := range choices {
				newResults = append(newResults, base+p)
				if len(newResults) > maxComb {
					return newResults[:maxComb]
				}
			}
		}
		results = newResults
	}
	if len(results) > maxComb {
		return results[:maxComb]
	}
	return results
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

// 编辑距离
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
			if dp[i-1][j]+1 < dp[i][j-1]+1 {
				if dp[i-1][j]+1 < dp[i-1][j-1]+cost {
					dp[i][j] = dp[i-1][j] + 1
				} else {
					dp[i][j] = dp[i-1][j-1] + cost
				}
			} else {
				if dp[i][j-1]+1 < dp[i-1][j-1]+cost {
					dp[i][j] = dp[i][j-1] + 1
				} else {
					dp[i][j] = dp[i-1][j-1] + cost
				}
			}
		}
	}
	return dp[len1][len2]
}

func approxMatch(query string, candidates []string, maxDist int) bool {
	for i, cand := range candidates {
		if i >= 3 { // 最多检查3个候选，避免性能问题
			return false
		}
		if editDistance(query, cand) <= maxDist {
			return true
		}
	}
	return false
}

func matchScore(query, name string, pc *PinyinCache, cfg Config) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	scores := []int{}

	// 文件名直匹配
	if fuzzyMatch(q, nameLower) {
		scores = append(scores, 500)
	}

	// 拼音匹配（多音字+首字母）
	fullList, initList := pc.GetAll(name, cfg.MaxCombinations)

	exactHit := false
	for _, full := range fullList {
		if fuzzyMatch(q, full) {
			scores = append(scores, 200)
			exactHit = true
		}
	}
	for _, initials := range initList {
		if fuzzyMatch(q, initials) {
			scores = append(scores, 180)
			exactHit = true
		}
	}

	// 容错
	if cfg.EnableFuzzy && !exactHit && len(q) <= 15 {
		if approxMatch(q, fullList, 2) {
			scores = append(scores, 120)
		}
		if approxMatch(q, initList, 1) {
			scores = append(scores, 100)
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

func searchDir(base string, depthLimit int, query string, pc *PinyinCache, excludes map[string]bool, wg *sync.WaitGroup, resultChan chan<- Result, cfg Config) {
	defer wg.Done()
	filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// 限制深度
		if strings.Count(strings.TrimPrefix(path, base), string(os.PathSeparator)) > depthLimit {
			if d.IsDir() {
				return filepath.SkipDir
			}
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || excludes[name] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		score := matchScore(query, name, pc, cfg)
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
	query := os.Args[1]
	cfg := loadConfig()

	excludesMap := make(map[string]bool)
	for _, e := range cfg.Excludes {
		excludesMap[e] = true
	}

	pc := NewPinyinCache()
	resultChan := make(chan Result, 1000)
	var wg sync.WaitGroup

	for _, d := range cfg.SearchDirs {
		wg.Add(1)
		go searchDir(d, cfg.MaxDepth, query, pc, excludesMap, &wg, resultChan, cfg)
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

	if len(results) > cfg.MaxResults {
		results = results[:cfg.MaxResults]
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
