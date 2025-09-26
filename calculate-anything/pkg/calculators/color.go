// calculate-anything/pkg/calculators/color.go
package calculators

import (
	"calculate-anything/pkg/alfred"
	"fmt"
	aw "github.com/deanishe/awgo"
	"math"
	"strconv"
	"strings"
)

// HandleColor 解析颜色代码（HEX, RGB）并提供不同格式的转换结果。
func HandleColor(wf *aw.Workflow, query string) {
	query = strings.TrimSpace(query)
	var r, g, b uint8 // 使用 uint8 (0-255) 来存储颜色分量

	// 尝试解析 HEX 格式, e.g., "#FF6347" or "#F63"
	if strings.HasPrefix(query, "#") {
		hex := strings.TrimPrefix(query, "#")
		// 将三位简写格式扩展为六位
		if len(hex) == 3 {
			hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
		}
		// 解析六位 HEX
		if len(hex) == 6 {
			val, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return // 无效的 HEX 格式，静默失败
			}
			r = uint8(val >> 16)
			g = uint8(val >> 8)
			b = uint8(val)
		} else {
			return
		}
	} else {
		// 尝试解析 RGB 格式, e.g., "rgb(255, 99, 71)"
		var rInt, gInt, bInt int
		_, err := fmt.Sscanf(strings.ToLower(query), "rgb(%d,%d,%d)", &rInt, &gInt, &bInt)
		if err != nil {
			return // 不是有效的 RGB 格式
		}
		// 验证 RGB 值范围
		if rInt < 0 || rInt > 255 || gInt < 0 || gInt > 255 || bInt < 0 || bInt > 255 {
			return
		}
		r, g, b = uint8(rInt), uint8(gInt), uint8(bInt)
	}

	// 生成不同格式的结果
	hexValue := fmt.Sprintf("#%02X%02X%02X", r, g, b)
	rgbValue := fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	hslValue := toHSL(r, g, b)

	results := []alfred.Result{
		{
			Title:    hexValue,
			Subtitle: "复制 HEX 值",
			Arg:      hexValue,
		},
		{
			Title:    rgbValue,
			Subtitle: "复制 RGB 值",
			Arg:      rgbValue,
		},
		{
			Title:    hslValue,
			Subtitle: "复制 HSL 值",
			Arg:      hslValue,
		},
	}

	alfred.AddToWorkflow(wf, results)
}

// toHSL 将 RGB 转换为 HSL 字符串 (标准的转换算法)
func toHSL(r, g, b uint8) string {
	R := float64(r) / 255
	G := float64(g) / 255
	B := float64(b) / 255

	max := math.Max(math.Max(R, G), B)
	min := math.Min(math.Min(R, G), B)

	h, s, l := 0.0, 0.0, (max+min)/2

	if max != min {
		d := max - min
		if l > 0.5 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}
		switch max {
		case R:
			h = (G - B) / d
			if G < B {
				h += 6
			}
		case G:
			h = (B-R)/d + 2
		case B:
			h = (R-G)/d + 4
		}
		h /= 6
	}

	return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", h*360, s*100, l*100)
}
