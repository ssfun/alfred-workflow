// calculate-anything/pkg/api/coinmarketcap.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	// 修正：统一使用 "github.com/deanishe/awgo"，不使用任何别名
	"github.com/deanishe/awgo"
)

const (
	coinMarketCapAPIURL = "https://pro-api.coinmarketcap.com/v1/tools/price-conversion"
	cryptoCacheKey      = "coinmarketcap_rates_%s_to_%s"
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
		ID          int     `json:"id"`
		Symbol      string  `json:"symbol"`
		Name        string  `json:"name"`
		Amount      float64 `json:"amount"`
		LastUpdated string  `json:"last_updated"`
		Quote       map[string]struct {
			Price       float64 `json:"price"`
			LastUpdated string  `json:"last_updated"`
		} `json:"quote"`
	} `json:"data"`
}

// GetCryptoConversion 获取加密货币到指定法币的转换率，优先使用缓存。
func GetCryptoConversion(wf *awgo.Workflow, apiKey string, amount float64, fromCrypto, toFiat string, cacheDuration time.Duration) (*CMCResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("CoinMarketCap API 密钥未配置")
	}

	fromCrypto = strings.ToUpper(fromCrypto)
	toFiat = strings.ToUpper(toFiat)
	cacheKey := fmt.Sprintf(cryptoCacheKey, fromCrypto, toFiat)

	// 修正：wf 的类型是 *awgo.Workflow
	if wf.Cache.Exists(cacheKey) && !wf.Cache.Expired(cacheKey, cacheDuration) {
		var resp CMCResponse
		if err := wf.Cache.LoadJSON(cacheKey, &resp); err == nil {
			if cachedQuote, ok := resp.Data.Quote[toFiat]; ok {
				resp.Data.Amount = amount
				cachedQuote.Price *= amount
				resp.Data.Quote[toFiat] = cachedQuote
				return &resp, nil
			}
		}
	}

	req, err := http.NewRequest("GET", coinMarketCapAPIURL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("amount", "1")
	q.Add("symbol", fromCrypto)
	q.Add("convert", toFiat)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accepts", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接到 CoinMarketCap API: %w", err)
	}
	defer resp.Body.Close()

	var apiResponse CMCResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}
	if apiResponse.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("API 错误: %s", apiResponse.Status.ErrorMessage)
	}

	if err := wf.Cache.StoreJSON(cacheKey, apiResponse); err != nil {
		wf.Logger().Printf("无法缓存加密货币数据: %s", err)
	}

	if baseQuote, ok := apiResponse.Data.Quote[toFiat]; ok {
		apiResponse.Data.Amount = amount
		baseQuote.Price *= amount
		apiResponse.Data.Quote[toFiat] = baseQuote
	}

	return &apiResponse, nil
}
