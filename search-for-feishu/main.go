package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	pinyin "github.com/mozillazg/go-pinyin"
)

// ---------------- 多音字字典 ----------------
var polyphonic = map[rune][]string{
	'行': {"hang", "xing"},
	'长': {"chang", "zhang"},
	'重': {"chong", "zhong"},
	'乐': {"le", "yue"},
	'处': {"chu", "cu"},
}

func loadPolyphonicDict(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // 保留默认
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
	cache map[string]struct {
		fullComb    []string
		initialComb []string
	}
}

func NewPinyinCache() *PinyinCache {
	return &PinyinCache{cache: make(map[string]struct {
		fullComb    []string
		initialComb []string
	})}
}

var pyArgs = pinyin.NewArgs()

func (pc *PinyinCache) GetAll(name string) ([]string, []string) {
	pc.mu.RLock()
	if val, ok := pc.cache[name]; ok {
		pc.mu.RUnlock()
		return val.fullComb, val.initialComb
	}
	pc.mu.RUnlock()

	var fullOptions [][]string
	var initialOptions [][]string

	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			if alts, ok := polyphonic[r]; ok && len(alts) > 0 {
				var fulls []string
				var initials []string
				for _, alt := range alts {
					fulls = append(fulls, alt)
					initials = append(initials, string(alt[0]))
				}
				fullOptions = append(fullOptions, fulls)
				initialOptions = append(initialOptions, initials)
			} else {
				py := pinyin.LazyPinyin(string(r), pyArgs)
				if len(py) > 0 {
					fullOptions = append(fullOptions, []string{py[0]})
					initialOptions = append(initialOptions, []string{string(py[0][0])})
				}
			}
		} else {
			fullOptions = append(fullOptions, []string{string(r)})
			initialOptions = append(initialOptions, []string{string(r)})
		}
	}

	fullComb := cartesianProduct(fullOptions)
	initialComb := cartesianProduct(initialOptions)

	pc.mu.Lock()
	pc.cache[name] = struct {
		fullComb    []string
		initialComb []string
	}{fullComb, initialComb}
	pc.mu.Unlock()

	return fullComb, initialComb
}

func cartesianProduct(arr [][]string) []string {
	if len(arr) == 0 {
		return []string{}
	}
	res := []string{""}
	for _, choices := range arr {
		var tmp []string
		for _, prefix := range res {
			for _, c := range choices {
				tmp = append(tmp, prefix+c)
			}
		}
		res = tmp
	}
	return res
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

// ---------------- 打分函数 ----------------
func matchScore(query, name string, pc *PinyinCache) int {
	if query == "" {
		return 0
	}
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)

	// 中文直接包含
	if strings.Contains(nameLower, q) {
		return 500
	}

	// 拼音匹配
	fullList, initialsList := pc.GetAll(name)

	for _, initials := range initialsList {
		if initials == q {
			return 480
		} else if strings.HasPrefix(initials, q) {
			return 450
		} else if strings.Contains(initials, q) {
			return 320
		}
		if fuzzyMatchAllowOneError(q, initials) {
			return 200
		}
	}

	for _, full := range fullList {
		if full == q {
			return 470
		} else if strings.HasPrefix(full, q) {
			return 420
		} else if strings.Contains(full, q) {
			return 300
		}
		if len(q) >= 3 && fuzzyMatchAllowOneError(q, full) {
			return 180
		}
	}

	return 0
}

// ---------------- Feishu 结构 ----------------
type Document struct {
	Title     string              `json:"title"`
	Preview   string              `json:"preview,omitempty"`
	OpenTime  int64               `json:"open_time"`
	EditName  string              `json:"edit_name"`
	Type      int                 `json:"type"`
	URL       string              `json:"url"`
	WikiInfos []map[string]string `json:"wiki_infos"`
}

type AlfredItem struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Arg      string `json:"arg"`
	Icon     struct {
		Path string `json:"path"`
	} `json:"icon"`
}

type AlfredOutput struct {
	Items []AlfredItem `json:"items"`
}

func removeEmTags(input string) string {
	re := regexp.MustCompile(`</?em>`)
	return re.ReplaceAllString(input, "")
}

func getIconPath(docType int) string {
	switch docType {
	case 22:
		return "icon_file_doc_type_22.png"
	case 3:
		return "icon_file_sheet_type_3.png"
	case 12:
		return "icon_file_PPT_type_12.png"
	case 11:
		return "icon_file_mindnote_type_11.png"
	default:
		return "icon.png"
	}
}

func formatForAlfred(docs []Document) AlfredOutput {
	items := []AlfredItem{}
	for _, doc := range docs {
		item := AlfredItem{
			Title:    removeEmTags(doc.Title),
			Subtitle: fmt.Sprintf("Last opened: %s by %s", time.Unix(doc.OpenTime, 0).Format("2006-01-02 15:04:05"), doc.EditName),
			Arg:      doc.URL,
		}
		if item.Arg == "" && len(doc.WikiInfos) > 0 {
			item.Arg = doc.WikiInfos[0]["wiki_url"]
		}
		item.Icon.Path = getIconPath(doc.Type)
		items = append(items, item)
	}
	return AlfredOutput{Items: items}
}

func main() {
	session := os.Getenv("FEISHU_SESSION")
	apiURL := os.Getenv("FEISHU_API_URL")
	if session == "" || apiURL == "" {
		output, _ := json.Marshal(AlfredOutput{
			Items: []AlfredItem{{Title: "Error", Subtitle: "环境变量 FEISHU_SESSION 或 FEISHU_API_URL 未设置"}},
		})
		fmt.Println(string(output))
		return
	}

	// 参数
	args := os.Args[1:]
	query := ""
	searchAll := false
	if len(args) > 0 {
		query = strings.Join(args, " ")
		if strings.HasSuffix(query, " -a") {
			query = strings.TrimSuffix(query, " -a")
			searchAll = true
		}
	}

	// 请求 Feishu API
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Cookie", fmt.Sprintf("session=%s; session_list=%s", session, session))
	q := req.URL.Query()
	if query != "" {
		q.Add("query", query)
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println(`{"items":[{"title":"Error","subtitle":"请求 Feishu 接口失败"}]}`)
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil || data["code"].(float64) != 0 {
		fmt.Println(`{"items":[{"title":"Error","subtitle":"解析响应失败"}]}`)
		return
	}

	entities := data["data"].(map[string]interface{})["entities"].(map[string]interface{})
	rawDocs := entities["objs"].(map[string]interface{})
	docs := []Document{}
	for _, v := range rawDocs {
		b, _ := json.Marshal(v)
		var d Document
		_ = json.Unmarshal(b, &d)
		docs = append(docs, d)
	}

	// 过滤 + 打分排序
	pc := NewPinyinCache()
	type scoredDoc struct {
		Doc   Document
		Score int
	}
	var results []scoredDoc
	for _, doc := range docs {
		title := removeEmTags(doc.Title)
		preview := removeEmTags(doc.Preview)
		score := matchScore(query, title, pc)
		if searchAll && score == 0 {
			score = matchScore(query, preview, pc)
		}
		if query == "" || score > 0 {
			results = append(results, scoredDoc{Doc: doc, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Doc.OpenTime > results[j].Doc.OpenTime
		}
		return results[i].Score > results[j].Score
	})

	finalDocs := []Document{}
	for _, r := range results {
		finalDocs = append(finalDocs, r.Doc)
	}

	output := formatForAlfred(finalDocs)
	j, _ := json.Marshal(output)
	fmt.Println(string(j))
}
