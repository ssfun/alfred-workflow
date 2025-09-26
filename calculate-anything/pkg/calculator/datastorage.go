// calculate-anything-go/pkg/calculators/datastorage.go
package calculators

import (
	"calculate-anything-go/pkg/alfred"
	"calculate-anything-go/pkg/config"
	"calculate-anything-go/pkg/parser"
	"fmt"
	"github.com/deanishe/awgo"
	"math"
	"strings"
)

type storageUnit struct {
	Name   string
	Factor float64 // Factor relative to Byte
}

var (
	// IEC standard (decimal)
	decimalUnits = map[string]storageUnit{
		"b":   {Name: "Byte", Factor: 1},
		"kb":  {Name: "Kilobyte", Factor: math.Pow(1000, 1)},
		"mb":  {Name: "Megabyte", Factor: math.Pow(1000, 2)},
		"gb":  {Name: "Gigabyte", Factor: math.Pow(1000, 3)},
		"tb":  {Name: "Terabyte", Factor: math.Pow(1000, 4)},
		"pb":  {Name: "Petabyte", Factor: math.Pow(1000, 5)},
		"bit": {Name: "Bit", Factor: 1.0 / 8.0},
	}
	// JEDEC standard (binary)
	binaryUnits = map[string]storageUnit{
		"b":   {Name: "Byte", Factor: 1},
		"kib": {Name: "Kibibyte", Factor: math.Pow(1024, 1)},
		"mib": {Name: "Mebibyte", Factor: math.Pow(1024, 2)},
		"gib": {Name: "Gibibyte", Factor: math.Pow(1024, 3)},
		"tib": {Name: "Tebibyte", Factor: math.Pow(1024, 4)},
		"pib": {Name: "Pebibyte", Factor: math.Pow(1024, 5)},
		"bit": {Name: "Bit", Factor: 1.0 / 8.0},
	}
)

func HandleDataStorage(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	from := strings.ToLower(p.From)
	to := strings.ToLower(p.To)

	var fromUnit, toUnit storageUnit
	var okFrom, okTo bool

	// 根据查询的单位和配置决定使用哪个单位表
	useBinary := cfg.DataStorageForceBinary || isBinaryUnit(from) || isBinaryUnit(to)

	if useBinary {
		// 如果是二进制模式，KB/MB/GB 也要按 1024 计算
		binaryUnits["kb"] = storageUnit{Name: "Kilobyte (binary)", Factor: math.Pow(1024, 1)}
		binaryUnits["mb"] = storageUnit{Name: "Megabyte (binary)", Factor: math.Pow(1024, 2)}
		binaryUnits["gb"] = storageUnit{Name: "Gigabyte (binary)", Factor: math.Pow(1024, 3)}
		// ... etc
		fromUnit, okFrom = binaryUnits[from]
		toUnit, okTo = binaryUnits[to]
	} else {
		fromUnit, okFrom = decimalUnits[from]
		toUnit, okTo = decimalUnits[to]
	}

	if !okFrom {
		alfred.ShowError(wf, fmt.Errorf("未知的数据存储单位: %s", p.From))
		return
	}
	if !okTo {
		alfred.ShowError(wf, fmt.Errorf("未知的数据存储单位: %s", p.To))
		return
	}

	// 转换步骤: Amount -> Bytes -> Target
	valueInBytes := p.Amount * fromUnit.Factor
	resultValue := valueInBytes / toUnit.Factor
	resultString := fmt.Sprintf("%g", resultValue)

	title := fmt.Sprintf("%g %s = %s %s", p.Amount, strings.ToUpper(p.From), resultString, strings.ToUpper(p.To))
	subtitle := fmt.Sprintf("复制 '%s'", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{Title: title, Subtitle: subtitle, Arg: resultString},
	})
}

func isBinaryUnit(unit string) bool {
	return strings.HasSuffix(strings.ToLower(unit), "ib")
}
