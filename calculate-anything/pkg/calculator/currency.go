package calculator

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const fixerAPI = "http://data.fixer.io/api/latest?access_key=%s"

// FixerResponse API 返回的结构体
type FixerResponse struct {
	Success bool               `json:"success"`
	Rates   map[string]float64 `json:"rates"`
}

// ConvertCurrency 实现货币转换
func ConvertCurrency(apiKey string, from string, to string, amount float64) (float64, error) {
	// awgo 提供了缓存机制，可以优先使用
	// wf.Cache.LoadOrStore(...)

	resp, err := http.Get(fmt.Sprintf(fixerAPI, apiKey))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data FixerResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	if !data.Success {
		return 0, fmt.Errorf("fixer.io API error")
	}

	// 汇率计算 (通常以 EUR 为基准)
	fromRate, okFrom := data.Rates[strings.ToUpper(from)]
	toRate, okTo := data.Rates[strings.ToUpper(to)]

	if !okFrom || !okTo {
		return 0, fmt.Errorf("invalid currency code")
	}

	return (amount / fromRate) * toRate, nil
}
