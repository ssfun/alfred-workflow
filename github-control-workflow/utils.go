package main

import (
	"strings"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func formatDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
