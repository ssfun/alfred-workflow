package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mozillazg/go-pinyin"
)

// ---------------- CONFIG ----------------
var (
	homeDir, _     = os.UserHomeDir()
	whitelistDirs  = []string{"Documents", "Desktop", "Downloads"}
	excludes       = map[string]bool{".git": true, "__pycache__": true, "node_modules": true}
	maxResults     = 100
)

// ---------------- 拼音工具 ----------------
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
	initials := strings.Join(pinyin.LazyPinyin(name, pinyin.NewArgs(pinyin.Args{Style: pinyin.FirstLetter})), "")

	pc.mu.Lock()
	pc.cache[name] = [2]string{full, initials}
	pc.mu.Unlock()
	return full, initials
}

// ---------------- 匹配评分 ----------------
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

	if fuzzyMatch(q, nameLower) {
		pos := strings.Index(nameLower, q)
		posScore := 50
		if pos >= 0 {
			posScore = pos
		}
		scores = append(scores, 300-posScore-abs(len(name)-len(q)))
	}

	full, initials := pc.Get(name)

	if fuzzyMatch(q, full) {
		scores = append(scores, 200-abs(len(full)-len(q)))
	}
	if fuzzyMatch(q, initials) {
		scores = append(scores, 100-abs(len(initials)-len(q)))
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
	Score int
	Path  string
	Name  string
}

func searchDir(base string, query string, pc *PinyinCache, wg *sync.WaitGroup, resultChan chan<- Result) {
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
			resultChan <- Result{score, path, name}
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
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}
	query := os.Args[1]
	pc := NewPinyinCache()

	resultChan := make(chan Result, 1000)
	var wg sync.WaitGroup

	for _, d := range whitelistDirs {
		fullPath := filepath.Join(homeDir, d)
		if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
			wg.Add(1)
			go searchDir(fullPath, query, pc, &wg, resultChan)
		}
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := []Result{}
	for r := range resultChan {
		results = append(results, r)
	}

	// 排序（简单按分数降序）
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	items := []AlfredItem{}
	for _, r := range results {
		items = append(items, AlfredItem{
			Uid:      r.Path,
			Title:    r.Name,
			Subtitle: r.Path,
			Arg:      r.Path,
		})
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
