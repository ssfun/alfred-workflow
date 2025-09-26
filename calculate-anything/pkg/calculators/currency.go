// calculate-anything/pkg/calculators/currency.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/api"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deanishe/awgo"
)

// 货币符号到标准三字母代码的映射表
// 这个映射也用于 IsCurrency 函数来判断一个词是否是货币
var currencySymbolMap = map[string]string{
	"€":   "EUR", "EURO": "EUR", "EUROS": "EUR",
	"¥":   "JPY", "YEN": "JPY",
	"$":   "USD", "DOLLAR": "USD", "DOLLARS": "USD",
	"£":   "GBP", "POUND": "GBP", "POUNDS": "GBP",
	"R$":  "BRL",
	"ЛВ":  "BGN",
	"៛":   "KHR",
	"C¥":  "CNY",
	"₡":   "CRC",
	"₱":   "CUP",
	"KČ":  "CZK",
	"KR":  "DKK",
	"RD$": "DOP",
	"¢":   "GHS",
	"Q":   "GTQ",
	"L":   "HNL",
	"FT":  "HUF",
	"₹":   "INR",
	"RP":  "IDR",
	"﷼":  "IRR",
	"₪":   "ILS",
	"J$":  "JMD",
	"₩":   "KRW",
	"ДЕН": "MKD",
	"RM":  "MYR",
	"MT":  "MZN",
	"Ƒ":   "ANG",
	"C$":  "NIO",
	"₦":   "NGN",
	"B/.": "PAB",
	"GS":  "PYG",
	"S/.": "PEN",
	"₺":   "TRY",
	"TT$": "TTD",
	"₴":   "UAH",
}


// IsCurrency 检查一个符号或词语是否是已知的货币。
func IsCurrency(symbol string) bool {
	s := strings.ToUpper(symbol)
	// 检查是否是标准三字母代码 (一个简化的检查)
	if len(s) == 3 {
		return true // 假设所有三字母大写字符串都是货币代码
	}
	// 检查是否在我们的符号映射表中
	_, exists := currencySymbolMap[s]
	return exists
}

// mapCurrencySymbol 将常见的货币符号或名称映射到标准的三字母代码。
func mapCurrencySymbol(symbol string) string {
	s := strings.ToUpper(symbol)
	if mapped, ok := currencySymbolMap[s]; ok {
		return mapped
	}
	// 如果不在映射表中，则假定它本身就是一个标准代码
	return s
}


// HandleCurrency 处理货币转换查询。
func HandleCurrency(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	// 从配置中获取缓存持续时间
	cacheDuration := time.Duration(cfg.CurrencyCacheHours) * time.Hour
	// 获取汇率数据（可能来自缓存或 API）
	rates, err := api.GetExchangeRates(wf, cfg.APIKeyFixer, cacheDuration)
	if err != nil {
		alfred.ShowError(wf, err)
		return
	}

	// 将查询中的符号/名称转换为标准代码
	fromCurrency := mapCurrencySymbol(p.From)
	toCurrency := mapCurrencySymbol(p.To)

	// 执行转换计算
	resultValue, err := api.ConvertCurrency(rates, fromCurrency, toCurrency, p.Amount)
	if err != nil {
		alfred.ShowError(wf, err)
		return
	}

	// 根据用户配置格式化小数位数
	format := fmt.Sprintf("%%.%df", cfg.CurrencyDecimals)
	resultStringFormatted := fmt.Sprintf(format, resultValue)
	resultStringUnformatted := strconv.FormatFloat(resultValue, 'f', -1, 64)

	title := fmt.Sprintf("%g %s = %s %s", p.Amount, fromCurrency, resultStringFormatted, toCurrency)
	subtitle := fmt.Sprintf("复制 '%s'", resultStringFormatted)

	// 将结果（包括修饰键操作）添加到 Alfred 反馈中
	alfred.AddToWorkflow(wf, []alfred.Result{
		{
			Title:    title,
			Subtitle: subtitle,
			Arg:      resultStringFormatted,
			Modifiers: []alfred.Modifier{
				{
					Key:      "cmd",
					Subtitle: fmt.Sprintf("复制无格式的值 '%s'", resultStringUnformatted),
					Arg:      resultStringUnformatted,
				},
			},
		},
	})
}
