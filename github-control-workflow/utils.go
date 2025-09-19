package main

import (
	"regexp"
	"strings"
)

var nonAlphanumericRegex = regexp.MustCompile(`[\s\-_]+`)

// normalize 归一化字符串，用于模糊匹配
func normalize(s string) string {
	return nonAlphanumericRegex.ReplaceAllString(strings.ToLower(s), "")
}

// makeMatchKeywords 为 Alfred 的 match 字段生成关键字
// 在这个简单实现中，它与 normalize 相同，但可以扩展
func makeMatchKeywords(s string) string {
	// 可以增加更复杂的逻辑，比如拆分单词
	return s
}
