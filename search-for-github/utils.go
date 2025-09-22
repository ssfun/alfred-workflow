package main

import (
	"os"
	"strconv"
	"strings"
	"os/exec"
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

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}

func openPath(path string) error {
	cmd := exec.Command("open", path)
	return cmd.Run()
}
