// calculate-anything/pkg/calculators/vat.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"strconv"
	"strings"
)

// HandleVAT 处理增值税计算
func HandleVAT(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	vatString := strings.TrimSpace(cfg.VATValue)
	if vatString == "" {
		alfred.ShowError(wf, fmt.Errorf("未在 Workflow 配置中设置 VAT 百分比"))
		return
	}

	vatString = strings.TrimSuffix(vatString, "%")
	vatPercent, err := strconv.ParseFloat(vatString, 64)
	if err != nil {
		alfred.ShowError(wf, fmt.Errorf("无效的 VAT 百分比格式: %s", cfg.VATValue))
		return
	}

	amount, err := strconv.ParseFloat(p.Input, 64)
	if err != nil {
		alfred.ShowError(wf, fmt.Errorf("无效的 VAT 计算金额: %s", p.Input))
		return
	}

	vatRate := vatPercent / 100
	vatAmount := amount * vatRate
	amountWithVAT := amount + vatAmount
	amountWithoutVAT := amount / (1 + vatRate)

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
			Subtitle: "如果 %.2f 是最终价格，则复制税前金额",
			Arg:      fmt.Sprintf("%.2f", amountWithoutVAT),
			IconPath: "icon.png",
		},
	}

	alfred.AddToWorkflow(wf, results)
}
