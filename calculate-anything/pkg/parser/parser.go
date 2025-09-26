package parser

import (
	"regexp"
	"strings"
)

// 定义查询的结构
type ParsedQuery struct {
	Value      float64
	FromUnit   string
	ToUnit     string
	Action     string // e.g., "currency", "unit", "percentage"
}

// 示例：解析货币转换查询，如 "100 usd to mxn"
func ParseCurrency(q string) (*ParsedQuery, bool) {
	// 正则表达式需要仔细设计，以匹配多种自然语言格式
	re := regexp.MustCompile(`(?i)^(\d+\.?\d*)\s*([a-zA-Z€$¥£]+)\s*(to|in|as)\s*([a-zA-Z€$¥£]+)$`)
	matches := re.FindStringSubmatch(q)

	if len(matches) == 5 {
		// ... 将字符串转换为 float64 等
		return &ParsedQuery{
			// ... 填充字段
		}, true
	}

	return nil, false
}
