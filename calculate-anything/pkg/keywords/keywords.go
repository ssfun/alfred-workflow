// calculate-anything/pkg/keywords/keywords.go
package keywords

import (
	"calculate-anything/pkg/i18n"
	"strings"
)

// PreprocessQuery 是智能解析器的第一步。
// 它接收原始查询和加载的语言包，然后返回一个清理过的、更易于机器解析的字符串。
// 例如: "100 euros to dollars" -> "100 eur usd"
func PreprocessQuery(query string, langPack *i18n.LanguagePack) string {
	// 如果语言包加载失败，为避免程序崩溃，直接返回原始查询
	if langPack == nil {
		return query
	}

	// 将查询按词分割，并转为小写，以便匹配
	words := strings.Fields(strings.ToLower(query))

	// 步骤 1: 替换关键字
	// 遍历每个词，如果它在语言包的关键字映射中，则替换为标准代码。
	for i, word := range words {
		if replacement, ok := langPack.Keywords[word]; ok {
			words[i] = replacement
		}
	}

	// 步骤 2: 移除停用词
	var cleanedWords []string
	for _, word := range words {
		isStopWord := false
		// 检查当前词是否在停用词列表中
		for _, stopWord := range langPack.StopWords {
			if word == stopWord {
				isStopWord = true
				break
			}
		}
		// 如果不是停用词，则将其保留
		if !isStopWord {
			cleanedWords = append(cleanedWords, word)
		}
	}

	// 将清理后的词重新组合成一个字符串
	return strings.Join(cleanedWords, " ")
}
