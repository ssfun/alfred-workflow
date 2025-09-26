// calculate-anything/pkg/calculators/crypto.go
package calculators

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

// 已知的加密货币列表（简化版，用于区分加密货币和法币）
// 原始项目使用一个巨大的 JSON 文件，这里为了性能和简洁性，只列出常见的。
var knownCryptos = map[string]bool{
	"BTC": true, "ETH": true, "XRP": true, "LTC": true, "BCH": true, "ADA": true,
	"DOT": true, "DOGE": true, "USDT": true, "BNB": true, "SOL": true, "AVAX": true,
}

// IsCrypto 检查一个符号是否是已知的加密货币。
func IsCrypto(symbol string) bool {
	return knownCryptos[symbol]
}

// HandleCrypto 处理加密货币转换查询。
func HandleCrypto(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	// 从配置中获取缓存持续时间
	cacheDuration := time.Duration(cfg.CryptoCurrencyCacheHours) * time.Hour

	// 统一将符号转为大写
	fromCrypto := p.From
	toTarget := p.To

	// 场景 1: 目标是另一种加密货币 (加密货币 -> 法币 -> 加密货币)
	if IsCrypto(toTarget) {
		// 我们使用一个稳定的法币（如 USD）作为中间转换媒介
		const intermediateFiat = "USD"

		// 步骤 1: 获取 "源加密货币 -> USD" 的汇率
		fromResp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, p.Amount, fromCrypto, intermediateFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}
		amountInUSD := fromResp.Data.Quote[intermediateFiat].Price

		// 步骤 2: 获取 "1 单位目标加密货币 -> USD" 的汇率，用于计算最终结果
		toResp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, 1, toTarget, intermediateFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}
		toRateUSD := toResp.Data.Quote[intermediateFiat].Price
		if toRateUSD == 0 {
			alfred.ShowError(wf, fmt.Errorf("无法获取 %s 的汇率", toTarget))
			return
		}

		// 最终结果 = (源加密货币的USD总值) / (目标加密货币的USD单价)
		resultValue := amountInUSD / toRateUSD
		displayResult(wf, cfg, p.Amount, fromCrypto, resultValue, toTarget)

	} else {
		// 场景 2: 目标是法币 (加密货币 -> 法币)
		// 复用货币符号映射函数，将 "dollars", "€" 等转换为标准代码
		toFiat := mapCurrencySymbol(toTarget)
		resp, err := api.GetCryptoConversion(wf, cfg.APIKeyCoinMarket, p.Amount, fromCrypto, toFiat, cacheDuration)
		if err != nil {
			alfred.ShowError(wf, err)
			return
		}

		quote, ok := resp.Data.Quote[toFiat]
		if !ok {
			alfred.ShowError(wf, fmt.Errorf("API 未返回目标货币 '%s' 的价格", toFiat))
			return
		}

		displayResult(wf, cfg, p.Amount, fromCrypto, quote.Price, toFiat)
	}
}

// displayResult 格式化加密货币的计算结果并将其添加到 Alfred 反馈中。
func displayResult(wf *aw.Workflow, cfg *config.AppConfig, fromAmount float64, fromSymbol string, toAmount float64, toSymbol string) {
	var resultString string
	// 根据用户配置决定小数位数，-1 表示显示所有小数
	if cfg.CryptoDecimals == -1 {
		resultString = strconv.FormatFloat(toAmount, 'f', -1, 64)
	} else {
		format := fmt.Sprintf("%%.%df", cfg.CryptoDecimals)
		resultString = fmt.Sprintf(format, toAmount)
	}
	
	resultStringUnformatted := strconv.FormatFloat(toAmount, 'f', -1, 64)

	title := fmt.Sprintf("%g %s = %s %s", fromAmount, fromSymbol, resultString, toSymbol)
	subtitle := fmt.Sprintf("复制 '%s'", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{
			Title:    title,
			Subtitle: subtitle,
			Arg:      resultString,
			IconPath: "icon.png", // 可以为加密货币准备一个专用图标
			Modifiers: []alfred.Modifier{
				{
					Key: aw.ModCmd,
					Subtitle: fmt.Sprintf("复制无格式的值 '%s'", resultStringUnformatted),
					Arg: resultStringUnformatted,
				},
			},
		},
	})
}
