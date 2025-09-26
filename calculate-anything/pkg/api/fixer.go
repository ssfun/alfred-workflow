// calculate-anything/pkg/api/fixer.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	// 修正：根据官方文档，统一使用 aw 别名导入
	aw "github.com/deanishe/awgo"
)

const (
	fixerAPIURL   = "http://data.fixer.io/api/latest?access_key=%s"
	fixerCacheKey = "fixer_rates"
)

// FixerResponse 镜像 fixer.io API 的 JSON 响应结构
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

// GetExchangeRates 从 fixer.io 获取最新汇率，优先使用缓存。
func GetExchangeRates(wf *aw.Workflow, apiKey string, cacheDuration time.Duration) (*FixerResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Fixer.io API 密钥未配置")
	}

	// 修正：wf 的类型是 *aw.Workflow
	if wf.Cache.Exists(fixerCacheKey) && !wf.Cache.Expired(fixerCacheKey, cacheDuration) {
		var rates FixerResponse
		if err := wf.Cache.LoadJSON(fixerCacheKey, &rates); err == nil {
			return &rates, nil
		}
	}

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

	// 修正：使用正确的 Logger() 方法获取日志记录器
	if err := wf.Cache.StoreJSON(fixerCacheKey, apiResponse); err != nil {
		wf.Logger().Printf("无法缓存汇率数据: %s", err)
	}

	return &apiResponse, nil
}

// ConvertCurrency 使用获取到的汇率数据进行货币转换。
func ConvertCurrency(rates *FixerResponse, from, to string, amount float64) (float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	
	fromRate, okFrom := rates.Rates[from]
	toRate, okTo := rates.Rates[to]

	if !okFrom {
		return 0, fmt.Errorf("无效的源货币代码: %s", from)
	}
	if !okTo {
		return 0, fmt.Errorf("无效的目标货币代码: %s", to)
	}
	if fromRate == 0 {
		return 0, fmt.Errorf("源货币 '%s' 的汇率为零，无法计算", from)
	}

	return (amount / fromRate) * toRate, nil
}
