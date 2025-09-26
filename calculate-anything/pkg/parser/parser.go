// calculate-anything/pkg/parser/parser.go
package parser

import (
	"calculate-anything/pkg/keywords"
	"regexp"
	"strconv"
	"strings"
)

// 修正: 移除了所有 QueryType, UnknownQuery 等常量的重复声明
// 这些定义现在只存在于 types.go 文件中

// 正则表达式集合
var (
	// 用于匹配预处理后的简单转换格式, e.g., "100 usd mxn", "100 km m"
	simpleConversionRegex = regexp.MustCompile(`^([\d.,]+)\s*([a-zA-Zμ°$€¥£\d]+)\s*([a-zA-Zμ°$€¥£\d]+)$`)

	// 用于处理固定结构的查询
	percentageRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([+\-]|plus|minus)\s*([\d.,]+)%$`)
	percentageOfRegex   = regexp.MustCompile(`(?i)^([\d.,]+)%\s*of\s*([\d.,]+)$`)
	percentageAsOfRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*(?:as a|is what)?\s*% of\s*([\d.,]+)$`)
	pxEmRemRegex        = regexp.MustCompile(`(?i)^([\d.,]+)\s*(px|em|rem|pt)(?:\s*(?:to|in)\s*(px|em|rem|pt))?$`)
)

// Parse 接收原始查询字符串并尝试解析它 (更新了函数签名以接收语言包)
func Parse(query string, langPack *i18n.LanguagePack) *ParsedQuery {
	// 策略1: 尝试匹配结构固定的查询 (百分比, Px/Em/Rem)
	if p := parseFixedStructureQueries(query); p != nil {
		return p
	}

	// 策略2: 对于转换类查询，先进行预处理
	processedQuery := keywords.PreprocessQuery(query, langPack)

	matches := simpleConversionRegex.FindStringSubmatch(processedQuery)
	if len(matches) == 4 {
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

// parseFixedStructureQueries 专门处理结构固定的查询
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
		if len(matches) == 4 {
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

// parseAmount 清理数字字符串并转换为 float64
func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// normalizeAction 将 'plus'/'minus' 等词语转换为符号
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
