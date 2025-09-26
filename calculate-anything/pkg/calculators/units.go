// calculate-anything/pkg/calculators/units.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"calculate-anything/pkg/parser"
	"fmt"
	"strconv"
	"strings"

	"github.com/deanishe/awgo"
)

// Unit 定义了一个物理单位及其换算到国际标准单位（SI）的规则。
type Unit struct {
	Name   string
	Type   string                // 单位类型, e.g., "length", "mass"
	ToSI   float64               // 乘以该因子可转换为 SI 单位
	FromSI func(float64) float64 // 从 SI 单位转换回来的函数（主要用于温度这类非线性转换）
}

// unitMap 包含了所有支持的物理单位，数据来源于 README.md 文件。
var unitMap = map[string]Unit{
	// --- 长度 (SI: 米 'm') ---
	"m":   {Name: "Meter", Type: "length", ToSI: 1.0},
	"km":  {Name: "Kilometer", Type: "length", ToSI: 1000.0},
	"dm":  {Name: "Decimeter", Type: "length", ToSI: 0.1},
	"cm":  {Name: "Centimeter", Type: "length", ToSI: 0.01},
	"mm":  {Name: "Milimeter", Type: "length", ToSI: 0.001},
	"μm":  {Name: "Micrometer", Type: "length", ToSI: 1e-6},
	"nm":  {Name: "Nanometer", Type: "length", ToSI: 1e-9},
	"pm":  {Name: "Picometer", Type: "length", ToSI: 1e-12},
	"in":  {Name: "Inch", Type: "length", ToSI: 0.0254},
	"ft":  {Name: "Foot", Type: "length", ToSI: 0.3048},
	"yd":  {Name: "Yard", Type: "length", ToSI: 0.9144},
	"mi":  {Name: "Mile", Type: "length", ToSI: 1609.34},
	"nmi": {Name: "Nautical Mile", Type: "length", ToSI: 1852.0},
	"h":   {Name: "Hand", Type: "length", ToSI: 0.1016},
	"ly":  {Name: "Lightyear", Type: "length", ToSI: 9.461e+15},
	"au":  {Name: "Astronomical Unit", Type: "length", ToSI: 1.496e+11},
	"pc":  {Name: "Parsec", Type: "length", ToSI: 3.086e+16},

	// --- 面积 (SI: 平方米 'm2') ---
	"m2":  {Name: "Square Meter", Type: "area", ToSI: 1.0},
	"km2": {Name: "Square Kilometer", Type: "area", ToSI: 1e6},
	"cm2": {Name: "Square Centimeter", Type: "area", ToSI: 1e-4},
	"mm2": {Name: "Square Milimeter", Type: "area", ToSI: 1e-6},
	"ft2": {Name: "Square Foot", Type: "area", ToSI: 0.092903},
	"mi2": {Name: "Square Mile", Type: "area", ToSI: 2.59e+6},
	"ha":  {Name: "Hectare", Type: "area", ToSI: 10000},

	// --- 体积 (SI: 立方米 'm3') ---
	"l":     {Name: "Litre", Type: "volume", ToSI: 0.001},
	"ml":    {Name: "Mililitre", Type: "volume", ToSI: 1e-6},
	"m3":    {Name: "Cubic Meter", Type: "volume", ToSI: 1.0},
	"kl":    {Name: "Kilolitre", Type: "volume", ToSI: 1.0},
	"hl":    {Name: "Hectolitre", Type: "volume", ToSI: 0.1},
	"qt":    {Name: "Quart", Type: "volume", ToSI: 0.000946353},
	"pt":    {Name: "Pint (US)", Type: "volume", ToSI: 0.000473176},
	"ukpt":  {Name: "Pint (UK)", Type: "volume", ToSI: 0.000568261},
	"gal":   {Name: "Gallon (US)", Type: "volume", ToSI: 0.00378541},
	"ukgal": {Name: "Gallon (UK)", Type: "volume", ToSI: 0.00454609},
	"floz":  {Name: "Fluid ounce", Type: "volume", ToSI: 2.95735e-5},

	// --- 重量 (SI: 千克 'kg') ---
	"kg":  {Name: "Kilogram", Type: "mass", ToSI: 1.0},
	"g":   {Name: "Gram", Type: "mass", ToSI: 0.001},
	"mg":  {Name: "Miligram", Type: "mass", ToSI: 1e-6},
	"n":   {Name: "Newton", Type: "mass", ToSI: 0.10197}, // 在地球重力下
	"st":  {Name: "Stone", Type: "mass", ToSI: 6.35029},
	"lb":  {Name: "Pound", Type: "mass", ToSI: 0.453592},
	"oz":  {Name: "Ounce", Type: "mass", ToSI: 0.0283495},
	"t":   {Name: "Metric Tonne", Type: "mass", ToSI: 1000.0},
	"ukt": {Name: "UK Long Ton", Type: "mass", ToSI: 1016.05},
	"ust": {Name: "US Short Ton", Type: "mass", ToSI: 907.185},

	// --- 速度 (SI: 米每秒 'mps') ---
	"mps": {Name: "Meters Per Second", Type: "speed", ToSI: 1.0},
	"kph": {Name: "Kilometers Per Hour", Type: "speed", ToSI: 1.0 / 3.6},
	"mph": {Name: "Miles Per Hour", Type: "speed", ToSI: 0.44704},
	"fps": {Name: "Feet Per Second", Type: "speed", ToSI: 0.3048},

	// --- 旋转 (SI: 弧度 'rad') ---
	"deg": {Name: "Degrees", Type: "rotation", ToSI: 0.0174533},
	"rad": {Name: "Radian", Type: "rotation", ToSI: 1.0},

	// --- 温度 (SI: 开尔文 'k') ---
	"k": {Name: "Kelvin", Type: "temperature", FromSI: func(k float64) float64 { return k }},
	"c": {Name: "Centigrade", Type: "temperature", FromSI: func(k float64) float64 { return k - 273.15 }},
	"f": {Name: "Fahrenheit", Type: "temperature", FromSI: func(k float64) float64 { return (k-273.15)*9/5 + 32 }},

	// --- 压力 (SI: 帕斯卡 'pa') ---
	"pa":   {Name: "Pascal", Type: "pressure", ToSI: 1.0},
	"kpa":  {Name: "Kilopascal", Type: "pressure", ToSI: 1000.0},
	"mpa":  {Name: "Megapascal", Type: "pressure", ToSI: 1e6},
	"bar":  {Name: "Bar", Type: "pressure", ToSI: 100000.0},
	"mbar": {Name: "Milibar", Type: "pressure", ToSI: 100.0},
	"psi":  {Name: "Pound-force Per Square Inch", Type: "pressure", ToSI: 6894.76},

	// --- 时间 (SI: 秒 's') ---
	"s":    {Name: "Second", Type: "time", ToSI: 1.0},
	"year": {Name: "Year", Type: "time", ToSI: 3.154e+7},
	"month":{Name: "Month", Type: "time", ToSI: 2.628e+6},
	"week": {Name: "Week", Type: "time", ToSI: 604800.0},
	"day":  {Name: "Day", Type: "time", ToSI: 86400.0},
	"hr":   {Name: "Hour", Type: "time", ToSI: 3600.0},
	"min":  {Name: "Minute", Type: "time", ToSI: 60.0},
	"ms":   {Name: "Milisecond", Type: "time", ToSI: 0.001},
	"μs":   {Name: "Microsecond", Type: "time", ToSI: 1e-6},
	"ns":   {Name: "Nanosecond", Type: "time", ToSI: 1e-9},

	// --- 能量/功率 (SI: 焦耳 'j') ---
	"j":    {Name: "Joule", Type: "energy", ToSI: 1.0},
	"kj":   {Name: "Kilojoule", Type: "energy", ToSI: 1000.0},
	"mj":   {Name: "Megajoule", Type: "energy", ToSI: 1e6},
	"cal":  {Name: "Calorie", Type: "energy", ToSI: 4.184},
	// "nm" 与长度单位冲突，需要在解析器层面做更智能的区分，此处暂时注释
	// "nm":   {Name: "Newton Meter", Type: "energy", ToSI: 1.0},
	"ftlb": {Name: "Foot Pound", Type: "energy", ToSI: 1.35582},
	"whr":  {Name: "Watt Hour", Type: "energy", ToSI: 3600.0},
	"kwhr": {Name: "Kilowatt Hour", Type: "energy", ToSI: 3.6e+6},
	"mwhr": {Name: "Megawatt Hour", Type: "energy", ToSI: 3.6e+9},
	"mev":  {Name: "Mega Electron Volt", Type: "energy", ToSI: 1.6022e-13},
}

// HandleUnits 处理物理单位的转换。
func HandleUnits(wf *aw.Workflow, p *parser.ParsedQuery) {
	// 将单位符号转为小写以匹配 unitMap
	fromUnit, okFrom := unitMap[strings.ToLower(p.From)]
	toUnit, okTo := unitMap[strings.ToLower(p.To)]

	if !okFrom {
		alfred.ShowError(wf, fmt.Errorf("未知的源单位: %s", p.From))
		return
	}
	if !okTo {
		alfred.ShowError(wf, fmt.Errorf("未知的目标单位: %s", p.To))
		return
	}

	// 确保两个单位属于同一类型（例如，不能将长度转换为质量）
	if fromUnit.Type != toUnit.Type {
		alfred.ShowError(wf, fmt.Errorf("无法在不同类型单位间转换: %s -> %s", fromUnit.Type, toUnit.Type))
		return
	}

	var resultValue float64
	// 温度是特例，它的转换不是简单的乘法，需要特殊处理
	if fromUnit.Type == "temperature" {
		var valueInKelvin float64
		// 步骤 1: 将输入温度统一转换为开尔文（Kelvin）
		switch fromUnit.Name {
		case "Kelvin":
			valueInKelvin = p.Amount
		case "Centigrade":
			valueInKelvin = p.Amount + 273.15
		case "Fahrenheit":
			valueInKelvin = (p.Amount-32)*5/9 + 273.15
		}
		// 步骤 2: 从开尔文转换为目标温度单位
		resultValue = toUnit.FromSI(valueInKelvin)
	} else {
		// 对于其他所有线性单位，使用标准转换流程: Amount -> SI -> Target
		valueInSI := p.Amount * fromUnit.ToSI
		resultValue = valueInSI / toUnit.ToSI
	}

	resultString := strconv.FormatFloat(resultValue, 'f', -1, 64)

	title := fmt.Sprintf("%g %s = %s %s", p.Amount, p.From, resultString, p.To)
	subtitle := fmt.Sprintf("复制 '%s'", resultString)

	alfred.AddToWorkflow(wf, []alfred.Result{
		{
			Title:    title,
			Subtitle: subtitle,
			Arg:      resultString,
		},
	})
}
