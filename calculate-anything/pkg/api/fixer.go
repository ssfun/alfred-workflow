// calculate-anything/pkg/api/fixer.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/deanishe/awgo"
)

const fixerAPIURL = "http://data.fixer.io/api/latest?access_key=%s"
const fixerCacheKey = "fixer_rates"

// FixerResponse mirrors the JSON structure from the fixer.io API
type FixerResponse struct {
	Success   bool               `json:"success"`
	Timestamp int64              `json:"timestamp"`
	Base      string             `json:"base"`
	Date      string             `json:"date"`
	Rates     map[string]float64 `json:"rates"`
	Error     struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

// GetExchangeRates fetches exchange rates from fixer.io, using cache if available.
func GetExchangeRates(wf *aw.Workflow, apiKey string, cacheDuration time.Duration) (*FixerResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Fixer.io API 密钥未配置")
	}

	// 使用 awgo 的缓存机制
	cachedRates := func() (*FixerResponse, error) {
		var rates FixerResponse
		if err := wf.Cache.LoadJSON(fixerCacheKey, &rates); err != nil {
			return nil, err
		}
		return &rates, nil
	}

	if wf.Cache.Exists(fixerCacheKey) && !wf.Cache.Expired(fixerCacheKey, cacheDuration) {
		return cachedRates()
	}

	// 如果缓存不存在或已过期，则从 API 获取
	url := fmt.Sprintf(fixerAPIURL, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法连接到 Fixer.io API: %w", err)
	}
	defer resp.Body.Close()

	var apiResponse FixerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	if !apiResponse.Success {
		return nil, fmt.Errorf("API 错误: %s", apiResponse.Error.Info)
	}

	// 存入缓存
	if err := wf.Cache.StoreJSON(fixerCacheKey, apiResponse); err != nil {
		// 即使缓存失败也继续执行，只是下次会重新请求
		wf.Logf("无法缓存汇率数据: %s", err)
	}

	return &apiResponse, nil
}

// ConvertCurrency performs the currency conversion using the fetched rates.
func ConvertCurrency(rates *FixerResponse, from, to string, amount float64) (float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	
    // Fixer.io 的免费计划基础货币是 EUR
	base := rates.Base 

	// 如果源货币就是基础货币
	if from == base {
		toRate, ok := rates.Rates[to]
		if !ok {
			return 0, fmt.Errorf("无效的目标货币代码: %s", to)
		}
		return amount * toRate, nil
	}

	// 如果目标货币是基础货币
	if to == base {
		fromRate, ok := rates.Rates[from]
		if !ok {
			return 0, fmt.Errorf("无效的源货币代码: %s", from)
		}
		return amount / fromRate, nil
	}


	// 通过基础货币进行转换
	fromRate, okFrom := rates.Rates[from]
	toRate, okTo := rates.Rates[to]

	if !okFrom {
		return 0, fmt.Errorf("无效的源货币代码: %s", from)
	}
	if !okTo {
		return 0, fmt.Errorf("无效的目标货币代码: %s", to)
	}

	// (amount / fromRate) 将 amount 转换为基础货币(EUR)
	// 然后乘以 toRate 转换为目标货币
	return (amount / fromRate) * toRate, nil
}
