// calculate-anything/pkg/calculators/time.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/deanishe/awgo"
)

// 正则表达式用于匹配不同类型的时间查询
var (
	// 匹配 "time +15 days", "time now - 2 hours"
	timeRelativeRegex = regexp.MustCompile(`(?i)^\s*(now|today)?\s*([+\-])\s*(\d+)\s*(year|month|day|week|hr|min|s)s?\b`)
	// 匹配 10 位 Unix 时间戳
	timestampRegex = regexp.MustCompile(`^\s*(\d{10})\s*$`)
)

// HandleTime 处理所有与时间相关的查询。
func HandleTime(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	// 加载用户配置的时区，如果失败则使用 UTC
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		loc = time.UTC
		wf.Logger().Printf("警告: 无效的时区 '%s', 回退到 UTC", cfg.Timezone)
	}
	now := time.Now().In(loc)
	input := p.Input

	// --- 场景 1: 尝试解析时间戳 ---
	matches := timestampRegex.FindStringSubmatch(input)
	if len(matches) == 2 {
		ts, _ := strconv.ParseInt(matches[1], 10, 64)
		t := time.Unix(ts, 0).In(loc)
		// 使用用户配置的日期格式进行格式化
		resultString := t.Format(cfg.DateFormat)
		title := fmt.Sprintf("时间戳转换结果: %s", resultString)
		alfred.AddToWorkflow(wf, []alfred.Result{
			{Title: title, Subtitle: "复制日期", Arg: resultString, IconPath: "clock.png"},
		})
		return
	}

	// --- 场景 2: 尝试解析相对时间 ---
	matches = timeRelativeRegex.FindStringSubmatch(input)
	if len(matches) == 5 {
		operator := matches[2]
		amount, _ := strconv.Atoi(matches[3])
		unit := strings.ToLower(matches[4])

		var futureTime time.Time
		if operator == "-" {
			amount = -amount // 如果是减号，则数量为负
		}

		// 根据单位进行相应的日期/时间增减
		switch unit {
		case "year":
			futureTime = now.AddDate(amount, 0, 0)
		case "month":
			futureTime = now.AddDate(0, amount, 0)
		case "week":
			futureTime = now.AddDate(0, 0, amount*7)
		case "day":
			futureTime = now.AddDate(0, 0, amount)
		case "hr":
			futureTime = now.Add(time.Duration(amount) * time.Hour)
		case "min":
			futureTime = now.Add(time.Duration(amount) * time.Minute)
		case "s":
			futureTime = now.Add(time.Duration(amount) * time.Second)
		default:
			alfred.ShowError(wf, fmt.Errorf("未知的时间单位: %s", unit))
			return
		}

		resultString := futureTime.Format(cfg.DateFormat)
		title := fmt.Sprintf("结果: %s", resultString)
		subtitle := "复制日期到剪贴板"

		alfred.AddToWorkflow(wf, []alfred.Result{
			{Title: title, Subtitle: subtitle, Arg: resultString, IconPath: "clock.png"},
		})
		return
	}
	
	// --- 其他场景: 如 "start of year", "days until 31 december" ---
	// 这需要更复杂的自然语言日期解析，超出了当前范围，但可以在此扩展。

	// 如果所有解析都失败，显示帮助信息
	wf.NewItem("无效的时间查询").Subtitle("请尝试 'time +3 days', 'time -2 months', 或 'time 1577836800'").Valid(false)
}
