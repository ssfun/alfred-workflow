package main

import (
	"fmt"
	"os"
	"strings"
)

// 匹配打分逻辑
func matchScore(query, name string, pc *PinyinCache) int {
	q := strings.ToLower(query)
	nameLower := strings.ToLower(name)
	score := 0
	debug := os.Getenv("DEBUG") == "1"

	// 英文文件：严格匹配
	if isASCII(name) && !containsChinese(name) {
		if nameLower == q {
			return 500
		}
		if strings.HasPrefix(nameLower, q) {
			return 450
		}
		if strings.Contains(nameLower, q) {
			return 400
		}
		return 0
	}

	// 中文文件
	full, initials := pc.Get(name)

	// 1. 首字母优先
	if strings.EqualFold(q, initials) {
		score = max(score, 380)
	} else if looseMatch(q, initials) {
		score = max(score, 250)
	}

	// 2. 全拼完全/前缀
	if strings.EqualFold(q, full) {
		score = max(score, 350)
	} else if strings.HasPrefix(full, q) {
		score = max(score, 300) // 前缀匹配
	}

	// 3. 多音字（收紧）
	if len(q) <= 6 && abs(len(q)-len(full)) <= 2 && retryPolyphonicMatch(q, name, full) {
		score = max(score, 180)
	}

	// 4. Fuzzy 容错（收紧）
	if len(q) >= 4 && abs(len(q)-len(full)) <= 1 && fuzzyMatchAllowOneError(q, full) {
		score = max(score, 80)
	}

	if debug && score > 0 {
		fmt.Fprintln(os.Stderr,
			"DEBUG:", name, "→ q:", q, "full:", full, "initials:", initials, "score:", score)
	}
	return score
}
