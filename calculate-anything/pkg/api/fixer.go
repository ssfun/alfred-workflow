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

const (
	// Fixer.io API 的最新汇率端点
	fixerAPIURL = "http://data.fixer.io/api/latest?access_key=%s"
	// 汇率数据的缓存键
	fixerCacheKey = "fixer_rates"
)

// FixerResponse 镜像 fixer.io API 的 JSON 响应结构
type FixerResponse struct {
	Success   bool               `json:"success"`
	Timestamp int64              `json:"timestamp"`
	Base      string             `json:"base"` // 基础货币 (免费版通常是 EUR)
	Date      string             `json:"date"`
	Rates     map[string]float64 `json:"rates"`
	Error     struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
}

// GetExchangeRates 从 fixer.io 获取最新汇率，优先使用缓存。
func GetExchangeRates(wf *aw.Workflow, apiKey string, cacheDuration time.Duration) (*FixerResponse, error) {
	// 如果未配置 API 密钥，则返回错误
	if apiKey == "" {
		return nil, fmt.Errorf("Fixer.io API 密钥未配置")
	}

	// 检查是否存在有效缓存
	if wf.Cache.Exists(fixerCacheKey) && !wf.Cache.Expired(fixerCacheKey, cacheDuration) {
		var rates FixerResponse
		if err := wf.Cache.LoadJSON(fixerCacheKey, &rates); err == nil {
			return &rates, nil
		}
	}

	// 如果没有有效缓存，则从 API 获取数据
	url := fmt.Sprintf(fixerAPIURL, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法连接到 Fixer.io API: %w", err)
	}
	defer resp.Body.Close()

	var apiResponse FixerResponse
	// 解析 JSON 响应
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	// 检查 API 是否返回错误
	if !apiResponse.Success {
		return nil, fmt.Errorf("API 错误: %s", apiResponse.Error.Info)
	}

	// 将获取到的新汇率数据存入缓存
	if err := wf.Cache.StoreJSON(fixerCacheKey, apiResponse); err != nil {
		// 记录缓存错误，但不中断主流程
		wf.Logger().Printf("无法缓存汇率数据: %s", err)
	}

	return &apiResponse, nil
}

// ConvertCurrency 使用获取到的汇率数据进行货币转换。
func ConvertCurrency(rates *FixerResponse, from, to string, amount float64) (float64, error) {
	// 统一转换为大写以匹配汇率表
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	// Fixer.io 免费版的基础货币总是 EUR
	base := rates.Base

	// 获取源货币和目标货币相对于基础货币的汇率
	fromRate, okFrom := rates.Rates[from]
	toRate, okTo := rates.Rates[to]

	// 如果源货币或目标货币的代码无效，则返回错误
	if !okFrom {
		return 0, fmt.Errorf("无效的源货币代码: %s", from)
	}
	if !okTo {
		return 0, fmt.Errorf("无效的目标货币代码: %s", to)
	}

	// 转换逻辑：
	// 1. (amount / fromRate) 将输入的金额从源货币转换为基础货币 (EUR)
	// 2. 然后乘以 toRate，将基础货币金额转换为目标货币
	return (amount / fromRate) * toRate, nil
}
