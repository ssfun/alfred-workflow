// calculate-anything/pkg/api/coinmarketcap.go
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
	// CoinMarketCap 专业版 API 的价格转换端点
	coinMarketCapAPIURL = "https://pro-api.coinmarketcap.com/v1/tools/price-conversion"
	// 缓存键的格式，用于区分不同的转换对
	cryptoCacheKey = "coinmarketcap_rates_%s_to_%s" // 例如: coinmarketcap_rates_BTC_to_USD
)

// CMCResponse 镜像 CoinMarketCap API 的 JSON 响应结构
type CMCResponse struct {
	Status struct {
		Timestamp    string `json:"timestamp"`
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Elapsed      int    `json:"elapsed"`
		CreditCount  int    `json:"credit_count"`
	} `json:"status"`
	Data struct {
		ID          int       `json:"id"`
		Symbol      string    `json:"symbol"`
		Name        string    `json:"name"`
		Amount      float64   `json:"amount"`
		LastUpdated string    `json:"last_updated"`
		Quote       map[string]struct {
			Price       float64 `json:"price"`
			LastUpdated string  `json:"last_updated"`
		} `json:"quote"`
	} `json:"data"`
}

// GetCryptoConversion 获取加密货币到指定法币的转换率，优先使用缓存。
func GetCryptoConversion(wf *aw.Workflow, apiKey string, amount float64, fromCrypto, toFiat string, cacheDuration time.Duration) (*CMCResponse, error) {
	// 如果未配置 API 密钥，则返回错误
	if apiKey == "" {
		return nil, fmt.Errorf("CoinMarketCap API 密钥未配置")
	}

	// 统一转换为大写以匹配 API 和缓存键
	fromCrypto = strings.ToUpper(fromCrypto)
	toFiat = strings.ToUpper(toFiat)

	// 生成本次查询的唯一缓存键
	cacheKey := fmt.Sprintf(cryptoCacheKey, fromCrypto, toFiat)

	// 检查是否存在有效缓存
	if wf.Cache.Exists(cacheKey) && !wf.Cache.Expired(cacheKey, cacheDuration) {
		var resp CMCResponse
		if err := wf.Cache.LoadJSON(cacheKey, &resp); err == nil {
			// 如果从缓存加载成功，我们需要根据新的 amount 重新计算总价
			// 缓存中存储的是 amount=1 的价格
			cachedQuote, ok := resp.Data.Quote[toFiat]
			if ok {
				// 更新响应中的 amount 和 quote.price
				resp.Data.Amount = amount
				// (缓存的单位价格 * 新的数量)
				cachedQuote.Price = cachedQuote.Price * amount
				resp.Data.Quote[toFiat] = cachedQuote

				return &resp, nil
			}
		}
	}

	// 如果没有有效缓存，则从 API 获取数据
	// 准备 API 请求
	req, err := http.NewRequest("GET", coinMarketCapAPIURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置 URL 查询参数
	q := req.URL.Query()
	q.Add("amount", "1") // 总是请求 amount=1 的价格，以便缓存和复用
	q.Add("symbol", fromCrypto)
	q.Add("convert", toFiat)
	req.URL.RawQuery = q.Encode()

	// 设置请求头，包含 API 密钥
	req.Header.Set("Accepts", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)

	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接到 CoinMarketCap API: %w", err)
	}
	defer resp.Body.Close()

	var apiResponse CMCResponse
	// 解析 JSON 响应
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	// 检查 API 是否返回错误
	if apiResponse.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("API 错误: %s", apiResponse.Status.ErrorMessage)
	}

	// 将获取到的 amount=1 的结果存入缓存
	if err := wf.Cache.StoreJSON(cacheKey, apiResponse); err != nil {
		// 记录缓存错误，但不中断主流程
		wf.Logger().Printf("无法缓存加密货币数据: %s", err)
	}
	
	// 根据用户输入的实际 amount，计算最终的总价
	baseQuote, ok := apiResponse.Data.Quote[toFiat]
	if ok {
		apiResponse.Data.Amount = amount
		baseQuote.Price = baseQuote.Price * amount
		apiResponse.Data.Quote[toFiat] = baseQuote
	}
	
	return &apiResponse, nil
}
