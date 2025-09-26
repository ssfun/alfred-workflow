// calculate-anything-go/pkg/parser/parser.go
package parser

import (
	"calculate-anything-go/pkg/keywords"
	"regexp"
	"strconv"
	"strings"
)

// QueryType 定义了查询的类型
type QueryType int

const (
	UnknownQuery QueryType = iota
	CurrencyQuery
	CryptoQuery
	UnitQuery
	DataStorageQuery
	PercentageQuery
	PxEmRemQuery
	TimeQuery
	VATQuery
)

// ParsedQuery 是解析自然语言查询后的结果
type ParsedQuery struct {
	Type        QueryType // 查询类型
	Input       string    // 原始输入
	Amount      float64   // 数值
	From        string    // 源单位/货币
	To          string    // 目标单位/货币
	Action      string    // 附加动作 (e.g., "+", "-")
	Percent     float64   // 百分比值
	BaseValue   float64   // 百分比计算的基础值
}


// 正则表达式集合
var (
	// 用于匹配预处理后的简单转换格式, e.g., "100 usd mxn", "100 km m"
	// 允许更广泛的字符集以匹配特殊单位如 'μs'
	simpleConversionRegex = regexp.MustCompile(`^([\d.,]+)\s*([a-zA-Zμ°$€¥£\d]+)\s*([a-zA-Zμ°$€¥£\d]+)$`)

	// 用于处理固定结构的查询，这些查询不应被预处理
	percentageRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([+\-]|plus|minus)\s*([\d.,]+)%$`)
	percentageOfRegex   = regexp.MustCompile(`(?i)^([\d.,]+)%\s*of\s*([\d.,]+)$`)
	percentageAsOfRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*(?:as a|is what)?\s*% of\s*([\d.,]+)$`)
	pxEmRemRegex        = regexp.MustCompile(`(?i)^([\d.,]+)\s*(px|em|rem|pt)(?:\s*(?:to|in)\s*(px|em|rem|pt))?$`)
)

// Parse 接收原始查询字符串并尝试解析它 (重构后版本)
func Parse(query string) *ParsedQuery {
	// 针对不同类型的查询，采用不同的解析策略

	// 策略1: 尝试匹配结构固定的查询 (百分比, Px/Em/Rem)
	// 这类查询包含 'of', '+', '%' 等关键字符，不应被预处理移除
	if p := parseFixedStructureQueries(query); p != nil {
		return p
	}

	// 策略2: 对于转换类查询，先进行预处理
	// '100 euros to dollars' -> '100 eur usd'
	processedQuery := keywords.PreprocessQuery(query)

	matches := simpleConversionRegex.FindStringSubmatch(processedQuery)
	if len(matches) == 4 {
		// 解析成功，但此时还无法确定具体是货币、单位还是数据存储
		// 我们暂时将其标记为 UnitQuery，由上层逻辑（cmd/root.go）根据单位具体内容来决定最终类型
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
	// 匹配 "120 + 30%"
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
	// 匹配 "15% of 50"
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
	// 匹配 "40 as a % of 50"
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
	// 为了国际化，同时移除逗号和处理用逗号作小数点的欧洲格式
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
