// calculate-anything/pkg/config/config.go
package config

import (
	"strings"

	"github.com/deanishe/awgo"
)

// AppConfig 结构体用于保存从 Alfred 获取的所有用户配置。
// 它为整个应用程序提供了一个统一的配置访问点。
type AppConfig struct {
	Language                 string   // 偏好语言 (e.g., "en_US")
	DecimalSeparator         string   // 输入时的小数点分隔符 ("dot" or "comma")
	NumberOutputFormat       string   // 数字输出格式
	Timezone                 string   // 时区 (e.g., "America/New_York")
	CurrencyDecimals         int      // 货币转换结果的小数位数
	BaseCurrencies           []string // 默认转换的目标货币 (e.g., ["USD", "EUR"])
	APIKeyFixer              string   // Fixer.io 的 API 密钥
	CurrencyCacheHours       int      // 货币汇率缓存的小时数
	APIKeyCoinMarket         string   // CoinMarketCap 的 API 密钥
	CryptoCurrencyCacheHours int      // 加密货币汇率缓存的小时数
	CryptoDecimals           int      // 加密货币转换结果的小数位数
	VATValue                 string   // 默认的增值税率 (e.g., "16%")
	DateFormat               string   // 时间计算结果的输出格式
	PixelsBase               string   // px/em/rem 转换的基础像素值 (e.g., "16px")
	DataStorageForceBinary   bool     // 是否强制使用二进制模式（1024）进行数据存储单位转换
}

// Load 函数使用 awgo 库从 Alfred 的环境变量和配置文件中加载所有配置项。
// 它为每个配置项提供了默认值，以防用户没有设置。
func Load(wf *aw.Workflow) *AppConfig {
	// wf.Config.Get... 系列方法会首先尝试从环境变量读取，如果失败则回退到默认值。
	return &AppConfig{
		Language:                 wf.Config.GetString("language", "en_US"),
		DecimalSeparator:         wf.Config.GetString("decimal_separator", "dot"),
		NumberOutputFormat:       wf.Config.GetString("number_output_format", "comma_dot"),
		Timezone:                 wf.Config.GetString("timezone", "UTC"),
		CurrencyDecimals:         wf.Config.GetInt("currency_decimals", 2),
		BaseCurrencies:           parseBaseCurrencies(wf.Config.GetString("base_currencies", "USD,EUR")),
		APIKeyFixer:              wf.Config.GetString("apikey_fixer", ""),
		CurrencyCacheHours:       wf.Config.GetInt("currency_cache_hours", 12),
		APIKeyCoinMarket:         wf.Config.GetString("apikey_coinmarket", ""),
		CryptoCurrencyCacheHours: wf.Config.GetInt("cryptocurrency_cache_hours", 6),
		CryptoDecimals:           wf.Config.GetInt("crypto_decimals", -1),
		VATValue:                 wf.Config.GetString("vat_value", "16%"),
		DateFormat:               wf.Config.GetString("date_format", "2006-01-02 15:04:05"), // 使用 Go 的标准时间格式
		PixelsBase:               wf.Config.GetString("pixels_base", "16px"),
		DataStorageForceBinary:   wf.Config.GetBool("datastorage_force_binary", false),
	}
}

// parseBaseCurrencies 是一个辅助函数，用于将逗号分隔的货币字符串解析为字符串切片。
func parseBaseCurrencies(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	// 清理每个货币代码前后的空格
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
