// calculate-anything-go/pkg/calculators/percentage.go
package calculators

import (
	"calculate-anything-go/pkg/parser"
	"fmt"
)

func HandlePercentage(p *parser.ParsedQuery) (string, error) {
	var result float64
	switch p.Action {
	case "+":
		result = p.BaseValue * (1 + p.Percent/100)
	case "-":
		result = p.BaseValue * (1 - p.Percent/100)
    case "of":
        result = (p.Percent / 100) * p.BaseValue
	default:
		return "", fmt.Errorf("未知的百分比操作: %s", p.Action)
	}
	return fmt.Sprintf("%.2f", result), nil
}
