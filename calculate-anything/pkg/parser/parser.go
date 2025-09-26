// calculate-anything-go/pkg/parser/parser.go
package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// 正则表达式集合
var (
	// e.g., "100 usd to mxn", "100 euros in dollars"
	currencyRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*([a-zA-Z$€¥£]+)\s*(?:to|in|as|a)\s*([a-zA-Z$€¥£]+)$`)
	// e.g., "100km m", "10 years to seconds"
	unitRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([a-zA-Z]+)\s*(?:to|in|as|=)?\s*([a-zA-Z]+)$`)
	// e.g., "100gb mb", "2tb in gb"
	dataRegex     = regexp.MustCompile(`(?i)^([\d.,]+)\s*([a-zA-Z]+)\s*(?:to|in|as|=)?\s*([a-zA-Z]+)$`)
	// e.g., "120 + 30%", "15% of 50"
	percentageRegex = regexp.MustCompile(`(?i)^([\d.,]+)\s*([+\-]|plus|minus)\s*([\d.,]+)%$`)
	percentageOfRegex = regexp.MustCompile(`(?i)^([\d.,]+)% of ([\d.,]+)$`)
	// e.g., "12px", "2rem to pt"
	pxEmRemRegex  = regexp.MustCompile(`(?i)^([\d.,]+)\s*(px|em|rem|pt)(?:\s*(?:to|in)\s*(px|em|rem|pt))?$`)
)

// Parse 接收原始查询字符串并尝试解析它
func Parse(query string) *ParsedQuery {
	query = strings.TrimSpace(query)

	// 尝试匹配每一种查询类型
	if p := parseCurrency(query); p != nil {
		return p
	}
	if p := parseUnit(query); p != nil {
		return p
	}
	if p := parsePercentage(query); p != nil {
		return p
	}
	if p := parsePxEmRem(query); p != nil {
		return p
	}
	// ... 可以继续添加其他解析器

	return &ParsedQuery{Type: UnknownQuery, Input: query}
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}


func parseCurrency(q string) *ParsedQuery {
	matches := currencyRegex.FindStringSubmatch(q)
	if len(matches) == 4 {
		return &ParsedQuery{
			Type:   CurrencyQuery,
			Input:  q,
			Amount: parseAmount(matches[1]),
			From:   strings.ToUpper(matches[2]),
			To:     strings.ToUpper(matches[3]),
		}
	}
	return nil
}

func parseUnit(q string) *ParsedQuery {
    // 这里需要一个单位列表来区分是单位查询还是数据存储查询
    // 为了简化，我们暂时假设可以区分
	matches := unitRegex.FindStringSubmatch(q)
	if len(matches) == 4 {
		// 在真实场景中，你需要检查 matches[2] 和 matches[3] 是否是有效的物理单位
		return &ParsedQuery{
			Type:   UnitQuery,
			Input:  q,
			Amount: parseAmount(matches[1]),
			From:   matches[2],
			To:     matches[3],
		}
	}
	return nil
}

func parsePercentage(q string) *ParsedQuery {
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

	return nil
}

func parsePxEmRem(q string) *ParsedQuery {
    matches := pxEmRemRegex.FindStringSubmatch(q)
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
            To:     toUnit, // 如果没有指定 "to"，这里会是空字符串
        }
    }
    return nil
}

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
