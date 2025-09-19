// utils.go
package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// getEnv 从环境变量中读取一个值，如果不存在则返回指定的备用值
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// parseIntEnv 从环境变量中读取一个值并解析为整数，如果失败则返回备用值
func parseIntEnv(key string, fallback int) int {
	s, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

// formatSubtitle 格式化 Alfred item 的副标题
func formatSubtitle(stars int, updatedAt time.Time, desc string) string {
	if desc == "" {
		desc = "(无描述)"
	}
	return fmt.Sprintf("★ %d  ·  更新于 %s  ·  %s", stars, updatedAt.Format("2006-01-02"), desc)
}

// normalize 归一化字符串，用于模糊搜索
func normalize(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[\s\-_]+`)
	return re.ReplaceAllString(s, "")
}
