// calculate-anything/pkg/calculators/currency.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/api"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"strings"
	"time"
)

// HandleCurrency processes a currency conversion query
func HandleCurrency(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	cacheDuration := time.Duration(cfg.CurrencyCacheHours) * time.Hour
	rates, err := api.GetExchangeRates(wf, cfg.APIKeyFixer, cacheDuration)
	if err != nil {
		alfred.ShowError(wf, err)
		return
	}

	// 为README中提到的特殊符号进行映射
	fromCurrency := mapCurrencySymbol(p.From)
	toCurrency := mapCurrencySymbol(p.To)

	resultValue, err := api.ConvertCurrency(rates, fromCurrency, toCurrency, p.Amount)
	if err != nil {
		alfred.ShowError(wf, err)
		return
	}

	// 格式化输出
	format := fmt.Sprintf("%%.%df", cfg.CurrencyDecimals)
	resultString := fmt.Sprintf(format, resultValue)

	title := fmt.Sprintf("%s %s = %s %s", formatNumber(p.Amount), fromCurrency, resultString, toCurrency)
	subtitle := fmt.Sprintf("复制 '%s' 到剪贴板", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{
			Title:    title,
			Subtitle: subtitle,
			Arg:      resultString,
		},
	})
}

// formatNumber 简单地格式化数字以便显示
func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// mapCurrencySymbol 将常见的货币符号映射到标准的三字母代码
func mapCurrencySymbol(symbol string) string {
	s := strings.ToUpper(symbol)
	switch s {
	case "€", "EURO", "EUROS":
		return "EUR"
	case "¥", "YEN":
		return "JPY"
	case "$", "DOLLAR", "DOLLARS":
		return "USD"
	case "£", "POUND", "POUNDS":
		return "GBP"
	// 添加更多在 README.md 中列出的符号...
	case "R$":
		return "BRL"
	case "KČ":
		return "CZK"
	case "₹":
		return "INR"
	default:
		return s
	}
}

// IsCurrency 检查一个符号是否是已知的货币
func IsCurrency(symbol string) bool {
    // 这是一个简化实现，一个真实的应用会有一个完整的列表
    _, isKnownSymbol := mapCurrencySymbol(symbol)
    return isKnownSymbol
}
