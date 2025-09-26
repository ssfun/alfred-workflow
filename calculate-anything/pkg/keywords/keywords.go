// calculate-anything/pkg/keywords/keywords.go
package keywords

import "strings"

// KeywordMap 定义了从用户友好词汇到标准单位/货币代码的映射
var KeywordMap = map[string]string{
	// 货币 (部分示例)
	"dollars":    "USD",
	"dollar":     "USD",
	"euros":      "EUR",
	"euro":       "EUR",
	"yen":        "JPY",
	"pounds":     "GBP",
	"pound":      "GBP",
	"dolares":    "USD", // 西班牙语示例
	"pesos":      "MXN",

	// 单位 (部分示例)
	"kilometers": "km",
	"kilometer":  "km",
	"meters":     "m",
	"meter":      "m",
	"ounces":     "oz",
	"ounce":      "oz",
	"hakunamatata": "year", // README 中的有趣示例

	// 更多...
}

// StopWords 是在解析前需要被移除的词
var StopWords = []string{
	"to", "in", "as", "a", "=", "equals", "como", "en", "es",
}

// PreprocessQuery 清理和规范化查询字符串
func PreprocessQuery(query string) string {
	// 替换关键字
	// 为了避免部分匹配 (e.g., "romanian" 中的 "oman")，我们按词分割处理
	words := strings.Fields(strings.ToLower(query))
	
	// 替换关键字
	for i, word := range words {
		if replacement, ok := KeywordMap[word]; ok {
			words[i] = replacement
		}
	}
	
	// 移除停用词
	var cleanedWords []string
	for _, word := range words {
		isStopWord := false
		for _, stopWord := range StopWords {
			if word == stopWord {
				isStopWord = true
				break
			}
		}
		if !isStopWord {
			cleanedWords = append(cleanedWords, word)
		}
	}

	return strings.Join(cleanedWords, " ")
}
