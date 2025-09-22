package main

import (
	"fmt"
	"os"
)

func main() {
	// 加载多音字字典
	loadPolyphonicDict("polyphonic.json")

	// 无参数时直接返回空
	if len(os.Args) < 2 {
		fmt.Println(`{"items": []}`)
		return
	}

	// 解析查询
	queries := parseQueryV2(os.Args[1])

	// 加载配置
	dirs, excludesList, maxRes, _ := getConfig()

	// 执行搜索
	results := RunSearch(dirs, excludesList, queries, maxRes)

	// 生成 Alfred JSON
	output := BuildAlfredOutput(results, maxRes)
	fmt.Println(output)
}
