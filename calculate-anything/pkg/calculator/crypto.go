// calculate-anything/pkg/calculators/crypto.go
package calculator

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/api"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"strconv"
	"time"

	"github.com/deanishe/awgo"
)

// HandleCrypto processes a cryptocurrency conversion query
func HandleCrypto(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	cacheDuration := time.Duration(cfg.CryptoCurrencyCacheHours) * time.Hour

	// 检查目标是否是另一种加密货币
	if isCrypto(p.To) {
		// 加密货币 -> 法定货币 -> 加密货币
		// 我们使用 USD 作为中间货币
		const intermediateFiat = "USD"

		// 1. 获取 From -> USD 的汇率
		fromResp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, p.Amount, p.From, intermediateFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}
		amountInUSD := fromResp.Data.Quote[intermediateFiat].Price

		// 2. 获取 1 To -> USD 的汇率，用来计算最终结果
		toResp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, 1, p.To, intermediateFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}
		toRateUSD := toResp.Data.Quote[intermediateFiat].Price
		if toRateUSD == 0 {
			alfred.ShowError(wf, fmt.Errorf("无法获取 %s 的汇率", p.To))
			return
		}

		resultValue := amountInUSD / toRateUSD
		displayResult(wf, cfg, p.Amount, p.From, resultValue, p.To)

	} else {
		// 加密货币 -> 法定货币
		toFiat := mapCurrencySymbol(p.To) // 复用货币符号映射
		resp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, p.Amount, p.From, toFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}

		quote, ok := resp.Data.Quote[toFiat]
		if !ok {
			alfred.ShowError(wf, fmt.Errorf("API 未返回目标货币 '%s' 的价格", toFiat))
			return
		}

		displayResult(wf, cfg, p.Amount, p.From, quote.Price, toFiat)
	}
}

// displayResult 格式化并向 Alfred 添加反馈
func displayResult(wf *aw.Workflow, cfg *config.AppConfig, fromAmount float64, fromSymbol string, toAmount float64, toSymbol string) {
	var resultString string
	if cfg.CryptoDecimals == -1 {
		resultString = strconv.FormatFloat(toAmount, 'f', -1, 64)
	} else {
		format := fmt.Sprintf("%%.%df", cfg.CryptoDecimals)
		resultString = fmt.Sprintf(format, toAmount)
	}

	title := fmt.Sprintf("%g %s = %s %s", fromAmount, fromSymbol, resultString, toSymbol)
	subtitle := fmt.Sprintf("复制 '%s'", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{
			Title:    title,
			Subtitle: subtitle,
			Arg:      resultString,
			IconPath: "icon.png", // 你可以为加密货币准备一个专用图标
		},
	})
}

// isCrypto 是一个简单的辅助函数，用于判断一个字符串是否可能是加密货币
// 在真实世界中，最好是维护一个已知加密货币的列表
func isCrypto(symbol string) bool {
	// 这是一个简化的检查。原始项目有一个巨大的 JSON 文件。
	// 我们可以检查长度或者一些常见的币种。
	knownCryptos := map[string]bool{"BTC": true, "ETH": true, "XRP": true, "LTC": true, "BCH": true, "ADA": true, "DOT": true, "DOGE": true}
	return knownCryptos[symbol]
}
