// calculate-anything/pkg/calculators/vat.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"strconv"
	"strings"

	"github.com/deanishe/awgo"
)

// HandleVAT 处理增值税（Value Added Tax）计算。
func HandleVAT(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	// 从配置中读取用户设置的 VAT 百分比字符串
	vatString := strings.TrimSpace(cfg.VATValue)
	if vatString == "" {
		alfred.ShowError(wf, fmt.Errorf("未在 Workflow 配置中设置 VAT 百分比"))
		return
	}

	// 清理字符串（移除 % 符号）并转换为浮点数
	vatString = strings.TrimSuffix(vatString, "%")
	vatPercent, err := strconv.ParseFloat(vatString, 64)
	if err != nil {
		alfred.ShowError(wf, fmt.Errorf("无效的 VAT 百分比格式: %s", cfg.VATValue))
		return
	}

	// 解析用户输入的金额
	amount, err := strconv.ParseFloat(p.Input, 64)
	if err != nil {
		alfred.ShowError(wf, fmt.Errorf("无效的 VAT 计算金额: %s", p.Input))
		return
	}

	// 执行计算
	vatRate := vatPercent / 100.0
	vatAmount := amount * vatRate         // 税额
	amountWithVAT := amount + vatAmount   // 税后总额
	amountWithoutVAT := amount / (1 + vatRate) // 税前金额（如果输入的是含税价）

	// 生成三个不同的结果，分别对应原始 README 中的三种情况
	results := []alfred.Result{
		{
			Title:    fmt.Sprintf("VAT 金额 (%.2f%%): %.2f", vatPercent, vatAmount),
			Subtitle: "复制税额",
			Arg:      fmt.Sprintf("%.2f", vatAmount),
			IconPath: "icon.png",
		},
		{
			Title:    fmt.Sprintf("税后总额: %.2f", amountWithVAT),
			Subtitle: "复制金额 + VAT",
			Arg:      fmt.Sprintf("%.2f", amountWithVAT),
			IconPath: "icon.png",
		},
		{
			Title:    fmt.Sprintf("税前金额: %.2f", amountWithoutVAT),
			Subtitle: fmt.Sprintf("如果 %g 是最终价格，则复制税前金额", amount),
			Arg:      fmt.Sprintf("%.2f", amountWithoutVAT),
			IconPath: "icon.png",
		},
	}

	alfred.AddToWorkflow(wf, results)
}
