// calculate-anything/pkg/parser/types.go
package parser

// QueryType defines the type of query detected by the parser.
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

// ParsedQuery is the structured result after parsing a natural language query.
type ParsedQuery struct {
	Type      QueryType // The type of query
	Input     string    // The original input string
	Amount    float64   // The primary numerical value
	From      string    // The source unit or currency
	To        string    // The target unit or currency
	Action    string    // An associated action (e.g., "+", "-")
	Percent   float64   // A percentage value
	BaseValue float64   // The base value for percentage calculations
}
