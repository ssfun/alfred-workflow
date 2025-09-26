// calculate-anything-go/pkg/calculators/currency.go
package calculators

import (
	"calculate-anything-go/pkg/parser"
	"fmt"
)

func HandleCurrency(p *parser.ParsedQuery) (string, error) {
	// 在这里调用 API 客户端来获取汇率
	//
	// import "calculate-anything-go/pkg/api"
	// result, err := api.ConvertCurrency(config.APIKeyFixer, p.From, p.To, p.Amount)
	// if err != nil {
	// 	 return "", err
	// }
	//
	// return fmt.Sprintf("%.2f", result), nil

	// 作为演示，我们返回一个模拟结果
	return fmt.Sprintf("模拟结果：%.2f %s ≈ %.2f %s", p.Amount, p.From, p.Amount*1.1, p.To), nil
}
