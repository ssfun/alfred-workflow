// calculate-anything/pkg/calculators/datastorage.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/config"
	"calculate-anything/pkg/parser"
	"fmt"
	"math"
	"strings"

	"github.com/deanishe/awgo"
)

// storageUnit 定义了一个数据存储单位及其与“字节(Byte)”的换算因子
type storageUnit struct {
	Name   string
	Factor float64 // 相对于“字节”的换算因子
}

// 十进制单位 (IEC 标准, 1 KB = 1000 Bytes)
var decimalUnits = map[string]storageUnit{
	"B":   {Name: "Byte", Factor: 1},
	"KB":  {Name: "Kilobyte", Factor: math.Pow(1000, 1)},
	"MB":  {Name: "Megabyte", Factor: math.Pow(1000, 2)},
	"GB":  {Name: "Gigabyte", Factor: math.Pow(1000, 3)},
	"TB":  {Name: "Terabyte", Factor: math.Pow(1000, 4)},
	"PB":  {Name: "Petabyte", Factor: math.Pow(1000, 5)},
	"EB":  {Name: "Exabyte", Factor: math.Pow(1000, 6)},
	"ZB":  {Name: "Zettabyte", Factor: math.Pow(1000, 7)},
	"YB":  {Name: "Yottabyte", Factor: math.Pow(1000, 8)},
	"BIT": {Name: "Bit", Factor: 1.0 / 8.0},
}

// 二进制单位 (JEDEC/传统标准, 1 KiB = 1024 Bytes)
var binaryUnits = map[string]storageUnit{
	"B":   {Name: "Byte", Factor: 1},
	"KIB": {Name: "Kibibyte", Factor: math.Pow(1024, 1)},
	"MIB": {Name: "Mebibyte", Factor: math.Pow(1024, 2)},
	"GIB": {Name: "Gibibyte", Factor: math.Pow(1024, 3)},
	"TIB": {Name: "Tebibyte", Factor: math.Pow(1024, 4)},
	"PIB": {Name: "Pebibyte", Factor: math.Pow(1024, 5)},
	"EIB": {Name: "Exbibyte", Factor: math.Pow(1024, 6)},
	"ZIB": {Name: "Zebibyte", Factor: math.Pow(1024, 7)},
	"YIB": {Name: "Yobibyte", Factor: math.Pow(1024, 8)},
	"BIT": {Name: "Bit", Factor: 1.0 / 8.0},
}

// IsDataStorageUnit 检查一个单位字符串是否是已知的数据存储单位。
func IsDataStorageUnit(unit string) bool {
	u := strings.ToUpper(unit)
	_, isDecimal := decimalUnits[u]
	_, isBinary := binaryUnits[u]
	return isDecimal || isBinary
}

// HandleDataStorage 处理数据存储单位的转换。
func HandleDataStorage(wf *aw.Workflow, cfg *config.AppConfig, p *parser.ParsedQuery) {
	from := strings.ToUpper(p.From)
	to := strings.ToUpper(p.To)

	var fromUnit, toUnit storageUnit
	var okFrom, okTo bool

	// 判断本次转换应该使用二进制还是十进制
	// 满足以下任一条件即使用二进制：
	// 1. 用户在配置中强制开启二进制模式
	// 2. 查询的单位中包含二进制单位 (如 KiB, MiB)
	useBinary := cfg.DataStorageForceBinary || isBinaryUnit(from) || isBinaryUnit(to)

	var activeUnitMap map[string]storageUnit
	if useBinary {
		// 在二进制模式下，我们让 KB, MB, GB 也按 1024 计算以符合传统用法
		activeUnitMap = make(map[string]storageUnit)
		for k, v := range binaryUnits { activeUnitMap[k] = v }
		activeUnitMap["KB"] = storageUnit{Name: "Kilobyte (binary)", Factor: math.Pow(1024, 1)}
		activeUnitMap["MB"] = storageUnit{Name: "Megabyte (binary)", Factor: math.Pow(1024, 2)}
		activeUnitMap["GB"] = storageUnit{Name: "Gigabyte (binary)", Factor: math.Pow(1024, 3)}
		activeUnitMap["TB"] = storageUnit{Name: "Terabyte (binary)", Factor: math.Pow(1024, 4)}
		activeUnitMap["PB"] = storageUnit{Name: "Petabyte (binary)", Factor: math.Pow(1024, 5)}
	} else {
		activeUnitMap = decimalUnits
	}

	fromUnit, okFrom = activeUnitMap[from]
	toUnit, okTo = activeUnitMap[to]

	if !okFrom {
		alfred.ShowError(wf, fmt.Errorf("未知的数据存储单位: %s", p.From))
		return
	}
	if !okTo {
		alfred.ShowError(wf, fmt.Errorf("未知的数据存储单位: %s", p.To))
		return
	}

	// 转换逻辑: Amount -> Bytes -> Target
	valueInBytes := p.Amount * fromUnit.Factor
	resultValue := valueInBytes / toUnit.Factor
	resultString := fmt.Sprintf("%g", resultValue)

	title := fmt.Sprintf("%g %s = %s %s", p.Amount, p.From, resultString, p.To)
	subtitle := fmt.Sprintf("复制 '%s'", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{Title: title, Subtitle: subtitle, Arg: resultString},
	})
}

// isBinaryUnit 检查一个单位是否是标准的二进制单位（以 'iB' 结尾）。
func isBinaryUnit(unit string) bool {
	return strings.HasSuffix(strings.ToUpper(unit), "IB")
}
