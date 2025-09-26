// calculate-anything/pkg/parser/parser.go
package parser

import (
	"calculate-anything/pkg/i18n"
	"calculate-anything/pkg/keywords"
	"regexp"
	"strconv"
	"strings"
)

// 正则表达式集合
var (
	// 用于匹配经过预处理后的简单转换格式, e.g., "100 usd mxn", "100 km m"。
	// 它期望的格式是 "数字 单位 单位"。
	simpleConversionRegex = regexp.MustCompile(`^([\d.,]+)\s*([a-zA-Zμ°$€¥£\d]+)\s*([a-zA-Zμ°$€¥£\d]+)$`)

	// 用于处理结构固定的查询，这些查询不应该被预处理，因为其中的词（如 "of"）是关键部分。
	percentageRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([+\-]|plus|minus)\s*([\d.,]+)%$`)
	percentageOfRegex   = regexp.MustCompile(`(?i)^([\d.,]+)%\s*of\s*([\d.,]+)$`)
	percentageAsOfRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*(?:as a|is what)?\s*% of\s*([\d.,]+)$`)
	pxEmRemRegex        = regexp.MustCompile(`(?i)^([\d.,]+)\s*(px|em|rem|pt)(?:\s*(?:to|in)\s*(px|em|rem|pt))?$`)
)

// Parse 是主解析函数，它接收原始查询和加载的语言包，返回一个结构化的 ParsedQuery。
func Parse(query string, langPack *i18n.LanguagePack) *ParsedQuery {
	// 策略 1: 优先尝试匹配结构固定的查询（百分比, Px/Em/Rem）。
	// 这类查询包含 'of', '+', '%' 等关键字符，不应被预处理移除。
	if p := parseFixedStructureQueries(query); p != nil {
		return p
	}

	// 策略 2: 对于转换类查询，先使用关键字和停用词系统进行预处理。
	// e.g., "100 euros to dollars" -> "100 eur usd"
	processedQuery := keywords.PreprocessQuery(query, langPack)

	// 使用简单的正则表达式匹配预处理后的字符串
	matches := simpleConversionRegex.FindStringSubmatch(processedQuery)
	if len(matches) == 4 {
		// 解析成功。此时还无法确定具体是货币、单位还是数据存储。
		// 我们暂时将其标记为 UnitQuery，由上层逻辑（cmd/root.go）根据单位的具体内容来决定最终类型。
		return &ParsedQuery{
			Type:   UnitQuery, // 默认为单位查询，等待上层逻辑细化
			Input:  query,
			Amount: parseAmount(matches[1]),
			From:   matches[2],
			To:     matches[3],
		}
	}

	// 如果所有策略都失败，返回未知类型
	return &ParsedQuery{Type: UnknownQuery, Input: query}
}

// parseFixedStructureQueries 专门处理结构固定的查询，避免被预处理破坏。
func parseFixedStructureQueries(q string) *ParsedQuery {
	// 尝试匹配百分比
	matches := percentageRegex.FindStringSubmatch(q)
	if len(matches) == 4 {
		return &ParsedQuery{
			Type:      PercentageQuery,
			Input:     q,
			BaseValue: parseAmount(matches[1]),
			Action:    normalizeAction(matches[2]),
			Percent:   parseAmount(matches[3]),
		}
	}
	matches = percentageOfRegex.FindStringSubmatch(q)
	if len(matches) == 3 {
		return &ParsedQuery{
			Type:      PercentageQuery,
			Input:     q,
			Action:    "of",
			Percent:   parseAmount(matches[1]),
			BaseValue: parseAmount(matches[2]),
		}
	}
	matches = percentageAsOfRegex.FindStringSubmatch(q)
	if len(matches) == 3 {
		return &ParsedQuery{
			Type:      PercentageQuery,
			Input:     q,
			Action:    "as % of",
			Amount:    parseAmount(matches[1]),
			BaseValue: parseAmount(matches[2]),
		}
	}

	// 尝试匹配 Px/Em/Rem
	matches = pxEmRemRegex.FindStringSubmatch(q)
	if len(matches) > 0 {
		toUnit := ""
		if len(matches) == 4 && matches[3] != "" {
			toUnit = matches[3]
		}
		return &ParsedQuery{
			Type:   PxEmRemQuery,
			Input:  q,
			Amount: parseAmount(matches[1]),
			From:   matches[2],
			To:     toUnit,
		}
	}

	return nil
}

// parseAmount 清理数字字符串并将其转换为 float64。
func parseAmount(s string) float64 {
	// 为了支持国际化格式（如 1,234.56），先移除所有逗号
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// normalizeAction 将 'plus'/'minus' 等词语统一转换成符号。
func normalizeAction(action string) string {
	action = strings.ToLower(action)
	switch action {
	case "plus":
		return "+"
	case "minus":
		return "-"
	default:
		return action
	}
}
