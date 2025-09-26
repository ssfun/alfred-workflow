// calculate-anything-go/pkg/calculators/time.go
package calculators

import (
	"calculate-anything-go/pkg/alfred"
	"calculate-anything-go/pkg/config"
	"calculate-anything-go/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeRegex = regexp.MustCompile(`(?i)^\s*(now|today)?\s*([+\-])\s*(\d+)\s*(year|month|day|week|hr|min|s)s?\b`)

func HandleTime(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		alfred.ShowError(wf, fmt.Errorf("无效的时区设置: %s", cfg.Timezone))
		return
	}
	now := time.Now().In(loc)

	// 匹配 "time +15 days"
	matches := timeRegex.FindStringSubmatch(p.Input)
	if len(matches) == 5 {
		operator := matches[2]
		amount, _ := strconv.Atoi(matches[3])
		unit := strings.ToLower(matches[4])

		var futureTime time.Time
		if operator == "-" {
			amount = -amount
		}

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
		subtitle := fmt.Sprintf("复制日期到剪贴板")

		alfred.AddToWorkflow(wf, []alfred.Result{
			{Title: title, Subtitle: subtitle, Arg: resultString, IconPath: "clock.png"},
		})
		return
	}

	// 还可以添加对时间戳转换等的支持...

	wf.NewItem("无效的时间查询").Subtitle("请尝试 'time +3 days' 或 'time -2 months'").Valid(false)
}
