// calculate-anything/pkg/config/config.go
package config

import (
	"os"
	"strconv"
	"strings"
	"github.com/deanishe/awgo"
)

// AppConfig 保存了从 Alfred 获取的所有配置
type AppConfig struct {
	Language                 string
	DecimalSeparator         string
	NumberOutputFormat       string
	Timezone                 string
	CurrencyDecimals         int
	BaseCurrencies           []string
	APIKeyFixer              string
	CurrencyCacheHours       int
	APIKeyCoinMarket         string
	CryptoCurrencyCacheHours int
	CryptoDecimals           int
	VATValue                 string
	DateFormat               string
	PixelsBase               string
	DataStorageForceBinary   bool
}

// Load 从环境变量加载配置
func Load(wf *aw.Workflow) *AppConfig {
	// 从 wf.Config 中读取配置，这是 awgo 推荐的方式
	return &AppConfig{
		Language:                 wf.Config.GetString("language", "en_EN"),
		DecimalSeparator:         wf.Config.GetString("decimal_separator", "dot"),
		NumberOutputFormat:       wf.Config.GetString("number_output_format", "comma_dot"),
		Timezone:                 wf.Config.GetString("timezone", "UTC"),
		CurrencyDecimals:         wf.Config.GetInt("currency_decimals", 2),
		BaseCurrencies:           parseBaseCurrencies(wf.Config.GetString("base_currencies", "USD,EUR")),
		APIKeyFixer:              wf.Config.GetString("apikey_fixer", ""),
		CurrencyCacheHours:       wf.Config.GetInt("currency_cache_hours", 12),
		APIKeyCoinMarket:         wf.Config.GetString("apikey_coinmarket", ""),
		CryptoCurrencyCacheHours: wf.Config.GetInt("cryptocurrency_cache_hours", 12),
		CryptoDecimals:           wf.Config.GetInt("crypto_decimals", -1),
		VATValue:                 wf.Config.GetString("vat_value", "16%"),
		DateFormat:               wf.Config.GetString("date_format", "2 Jan, 2006, 3:04:05 pm"),
		PixelsBase:               wf.Config.GetString("pixels_base", "16px"),
		DataStorageForceBinary:   wf.Config.GetBool("datastorage_force_binary", false),
	}
}

func parseBaseCurrencies(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// GetenvInt 读取一个环境变量并转换为整数
func GetenvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

// GetenvBool 读取一个环境变量并转换为布尔值
func GetenvBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}
