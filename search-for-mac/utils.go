package main

import (
	"github.com/mozillazg/go-pinyin"
)

// 是否为纯 ASCII
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// 是否包含中文
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// 两数最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 三数最小值
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// 子序列匹配
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

// 模糊匹配，允许 1 个错误
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

// 多音字重试，允许用替代拼音重组比较
func retryPolyphonicMatch(query string, name string, full string) bool {
	runes := []rune(name)
	for i, r := range runes {
		if alts, ok := polyphonic[r]; ok {
			for _, alt := range alts {
				altFull := rebuildPinyin(runes, i, alt)
				if looseMatch(query, altFull) {
					return true
				}
			}
		}
	}
	return false
}

func rebuildPinyin(runes []rune, idx int, alt string) string {
	parts := []string{}
	for i, r := range runes {
		if i == idx {
			parts = append(parts, alt)
		} else {
			py := pinyin.LazyPinyin(string(r), a)
			if len(py) > 0 {
				parts = append(parts, py[0])
			}
		}
	}
	return join(parts)
}

func join(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}
