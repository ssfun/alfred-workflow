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
	coinMarketCapAPIURL = "https://pro-api.coinmarketcap.com/v1/tools/price-conversion"
	cryptoCacheKey      = "coinmarketcap_rates_%s_to_%s" // e.g., coinmarketcap_rates_BTC_to_USD
)

// CMCResponse mirrors the JSON structure for the price conversion endpoint
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

// GetCryptoConversion fetches the conversion rate for a cryptocurrency to a fiat currency.
func GetCryptoConversion(wf *aw.Workflow, apiKey string, amount float64, fromCrypto, toFiat string, cacheDuration time.Duration) (*CMCResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("CoinMarketCap API 密钥未配置")
	}

	fromCrypto = strings.ToUpper(fromCrypto)
	toFiat = strings.ToUpper(toFiat)
	
	cacheKey := fmt.Sprintf(cryptoCacheKey, fromCrypto, toFiat)

	// 使用 awgo 的缓存机制
	if wf.Cache.Exists(cacheKey) && !wf.Cache.Expired(cacheKey, cacheDuration) {
		var resp CMCResponse
		if err := wf.Cache.LoadJSON(cacheKey, &resp); err == nil {
			// 更新缓存中的 amount 和 quote
			resp.Data.Amount = amount
			resp.Data.Quote[toFiat] = struct{Price float64 `json:"price"`; LastUpdated string `json:"last_updated"`}{
				Price: resp.Data.Quote[toFiat].Price / resp.Data.Amount * amount,
				LastUpdated: resp.Data.Quote[toFiat].LastUpdated,
			}
			return &resp, nil
		}
	}

	// 准备 API 请求
	req, err := http.NewRequest("GET", coinMarketCapAPIURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("amount", fmt.Sprintf("%f", amount))
	q.Add("symbol", fromCrypto)
	q.Add("convert", toFiat)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accepts", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)

	// 发送请求
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
	
	// 为了方便缓存，我们将 amount=1 的结果存起来
	if amount != 1.0 {
		// 为了缓存复用，请求一次 amount=1 的结果
		baseResp, err := GetCryptoConversion(wf, apiKey, 1.0, fromCrypto, toFiat, cacheDuration)
		if err != nil {
			wf.Logf("无法缓存基础汇率: %v", err) // 记录错误但继续
		} else {
			// 更新原始响应中的价格，使其对应传入的 amount
			price, ok := baseResp.Data.Quote[toFiat]
			if ok {
				apiResponse.Data.Quote[toFiat] = struct{Price float64 `json:"price"`; LastUpdated string `json:"last_updated"`}{
					Price: price.Price * amount,
					LastUpdated: price.LastUpdated,
				}
			}
		}
		// 存储 amount=1 的结果
		wf.Cache.StoreJSON(cacheKey, *baseResp)
	} else {
		// 如果 amount 本身就是 1, 直接缓存
		if err := wf.Cache.StoreJSON(cacheKey, apiResponse); err != nil {
			wf.Logf("无法缓存加密货币数据: %s", err)
		}
	}

	return &apiResponse, nil
}
