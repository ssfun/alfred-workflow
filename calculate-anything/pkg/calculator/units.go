// calculate-anything-go/pkg/calculators/units.go
package calculators

import (
    "calculate-anything-go/pkg/alfred"
    "calculate-anything-go/pkg/parser"
    "fmt"
    "github.com/deanishe/awgo"
    "strings"
)

// Unit represents a single unit of measurement
type Unit struct {
    Name    string
    Type    string  // "length", "mass", etc.
    ToSI    float64 // Factor to convert this unit to its SI base unit
}

var unitMap = map[string]Unit{
    // Length (SI base: meter 'm')
    "m":  {Name: "Meter", Type: "length", ToSI: 1.0},
    "km": {Name: "Kilometer", Type: "length", ToSI: 1000.0},
    "cm": {Name: "Centimeter", Type: "length", ToSI: 0.01},
    "mm": {Name: "Milimeter", Type: "length", ToSI: 0.001},
    "in": {Name: "Inch", Type: "length", ToSI: 0.0254},
    "ft": {Name: "Foot", Type: "length", ToSI: 0.3048},
    "yd": {Name: "Yard", Type: "length", ToSI: 0.9144},
    "mi": {Name: "Mile", Type: "length", ToSI: 1609.34},

    // Mass (SI base: kilogram 'kg')
    "kg": {Name: "Kilogram", Type: "mass", ToSI: 1.0},
    "g":  {Name: "Gram", Type: "mass", ToSI: 0.001},
    "mg": {Name: "Miligram", Type: "mass", ToSI: 1e-6},
    "lb": {Name: "Pound", Type: "mass", ToSI: 0.453592},
    "oz": {Name: "Ounce", Type: "mass", ToSI: 0.0283495},
    
    // Time (SI base: second 's')
    "s":      {Name: "Second", Type: "time", ToSI: 1.0},
    "min":    {Name: "Minute", Type: "time", ToSI: 60.0},
    "hr":     {Name: "Hour", Type: "time", ToSI: 3600.0},
    "day":    {Name: "Day", Type: "time", ToSI: 86400.0},
    "week":   {Name: "Week", Type: "time", ToSI: 604800.0},
    "month":  {Name: "Month", Type: "time", ToSI: 2.628e+6}, // approx
    "year":   {Name: "Year", Type: "time", ToSI: 3.154e+7},  // approx
}

func HandleUnits(wf *aw.Workflow, p *parser.ParsedQuery) {
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

    if fromUnit.Type != toUnit.Type {
        alfred.ShowError(wf, fmt.Errorf("无法在不同类型单位间转换: %s -> %s", fromUnit.Type, toUnit.Type))
        return
    }

    // 转换步骤: Amount -> SI -> Target
    valueInSI := p.Amount * fromUnit.ToSI
    resultValue := valueInSI / toUnit.ToSI
	resultString := strconv.FormatFloat(resultValue, 'f', -1, 64)


    title := fmt.Sprintf("%s %s = %s %s", formatNumber(p.Amount), p.From, resultString, p.To)
    subtitle := fmt.Sprintf("复制 '%s' 到剪贴板", resultString)

    alfred.AddToWorkflow(wf, []alfred.Result{
        {
            Title:    title,
            Subtitle: subtitle,
            Arg:      resultString,
        },
    })
}
