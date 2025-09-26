// calculate-anything/pkg/parser/parser.go
package parser

import (
	"calculate-anything/pkg/i18n" // 修正了导入路径，之前是 "calculate-anything/pkg/i18n"
	"calculate-anything/pkg/keywords"
	"regexp"
	"strconv"
	"strings"
)

// 正则表达式集合
var (
	simpleConversionRegex = regexp.MustCompile(`^([\d.,]+)\s*([a-zA-Zμ°$€¥£\d]+)\s*([a-zA-Zμ°$€¥£\d]+)$`)
	percentageRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([+\-]|plus|minus)\s*([\d.,]+)%$`)
	percentageOfRegex   = regexp.MustCompile(`(?i)^([\d.,]+)%\s*of\s*([\d.,]+)$`)
	percentageAsOfRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*(?:as a|is what)?\s*% of\s*([\d.,]+)$`)
	pxEmRemRegex        = regexp.MustCompile(`(?i)^([\d.,]+)\s*(px|em|rem|pt)(?:\s*(?:to|in)\s*(px|em|rem|pt))?$`)
)

// Parse 是主解析函数，它接收原始查询和加载的语言包，返回一个结构化的 ParsedQuery。
// 修正：修复了 i1A.LanguagePack 的拼写错误
func Parse(query string, langPack *i18n.LanguagePack) *ParsedQuery {
	if p := parseFixedStructureQueries(query); p != nil {
		return p
	}

	processedQuery := keywords.PreprocessQuery(query, langPack)

	matches := simpleConversionRegex.FindStringSubmatch(processedQuery)
	if len(matches) == 4 {
		return &ParsedQuery{
			Type:   UnitQuery,
			Input:  query,
			Amount: parseAmount(matches[1]),
			From:   matches[2],
			To:     matches[3],
		}
	}

	return &ParsedQuery{Type: UnknownQuery, Input: query}
}

// parseFixedStructureQueries 专门处理结构固定的查询。
func parseFixedStructureQueries(q string) *ParsedQuery {
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
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// normalizeAction 将词语统一转换成符号。
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
