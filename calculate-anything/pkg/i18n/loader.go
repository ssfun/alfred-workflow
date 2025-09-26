// calculate-anything/pkg/i18n/loader.go
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
)

// LanguagePack 定义了一个语言包的结构，对应于 data/lang/ 目录下的 JSON 文件。
type LanguagePack struct {
	Keywords  map[string]string `json:"keywords"`    // 关键字映射, e.g., "dollars" -> "USD"
	StopWords []string          `json:"stop_words"` // 需要在解析前移除的停用词, e.g., "to", "in"
}

// LoadLanguagePack 根据指定的语言代码 (e.g., "en_US", "es_ES") 加载对应的 JSON 语言文件。
func LoadLanguagePack(langCode string) (*LanguagePack, error) {
	// 在实际应用中，路径应该是相对于可执行文件的。
	// Alfred 会将工作流目录设置为当前工作目录，所以可以直接使用相对路径。
	filePath := fmt.Sprintf("data/lang/%s.json", langCode)

	data, err := os.ReadFile(filePath)
	if err != nil {
		// 如果找不到特定语言的文件，则自动回退到默认的英语语言包。
		if langCode != "en_US" {
			return LoadLanguagePack("en_US")
		}
		return nil, fmt.Errorf("无法加载默认语言文件 (en_US.json): %w", err)
	}

	var pack LanguagePack
	// 解析 JSON 数据到 LanguagePack 结构体
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("解析语言文件 '%s' 失败: %w", filePath, err)
	}

	return &pack, nil
}
