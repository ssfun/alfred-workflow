// calculate-anything-go/pkg/parser/types.go
package parser

// QueryType 定义了查询的类型
type QueryType int

const (
	UnknownQuery QueryType = iota
	CurrencyQuery
	CryptoQuery
	UnitQuery
	DataStorageQuery
	PercentageQuery
	PxEmRemQuery
	TimeQuery
	VATQuery
)

// ParsedQuery 是解析自然语言查询后的结果
type ParsedQuery struct {
	Type        QueryType // 查询类型
	Input       string    // 原始输入
	Amount      float64   // 数值
	From        string    // 源单位/货币
	To          string    // 目标单位/货币
	Action      string    // 附加动作 (e.g., "+", "-")
	Percent     float64   // 百分比值
	BaseValue   float64   // 百分比计算的基础值
}
