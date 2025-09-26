// calculate-anything/pkg/parser/types.go
package parser

// QueryType 是一个枚举类型，用于定义解析器识别出的查询类型。
type QueryType int

// 定义所有可能的查询类型常量
const (
	UnknownQuery     QueryType = iota // 未知或无法解析的查询
	CurrencyQuery                     // 货币转换查询
	CryptoQuery                       // 加密货币转换查询
	UnitQuery                         // 物理单位转换查询
	DataStorageQuery                  // 数据存储单位转换查询
	PercentageQuery                   // 百分比计算查询
	PxEmRemQuery                      // Web 开发单位转换查询
	TimeQuery                         // 时间计算查询
	VATQuery                          // 增值税计算查询
)

// ParsedQuery 是解析自然语言查询后的结构化结果。
// 它是解析器和计算器之间传递数据的核心数据结构。
type ParsedQuery struct {
	Type      QueryType // 查询的类型
	Input     string    // 用户输入的原始查询字符串
	Amount    float64   // 查询中的主要数值 (e.g., 100 in "100 usd to eur")
	From      string    // 源单位/货币 (e.g., "usd")
	To        string    // 目标单位/货币 (e.g., "eur")
	Action    string    // 附加的动作，主要用于百分比计算 (e.g., "+", "-", "of")
	Percent   float64   // 百分比计算中的百分比值 (e.g., 15 in "120 + 15%")
	BaseValue float64   // 百分比计算中的基础值 (e.g., 120 in "120 + 15%")
}
