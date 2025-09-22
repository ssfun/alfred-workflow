package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/colornames"
)

type Emoji struct {
	Char     string `json:"char"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Item struct {
	Title    string            `json:"title"`
	Subtitle string            `json:"subtitle"`
	Arg      string            `json:"arg"`
	Icon     map[string]string `json:"icon"`
}

// 渲染 Emoji 为 PNG 图标
func renderEmojiPNG(char string, face font.Face, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(colornames.Black),
		Face: face,
		Dot:  fixed.P(30, 200), // 调整数值与居中效果
	}
	d.DrawString(char)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	baseDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	iconDir := filepath.Join(baseDir, "icons")
	os.MkdirAll(iconDir, 0755)

	// 载入 emoji.json
	data, err := ioutil.ReadFile(filepath.Join(baseDir, "emoji.json"))
	if err != nil {
		fmt.Printf(`{"items":[{"title":"错误","subtitle":"无法读取 emoji.json","valid":false}]}`)
		return
	}

	var emojis []Emoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		fmt.Printf(`{"items":[{"title":"错误","subtitle":"emoji.json 解析失败","valid":false}]}`)
		return
	}

	// 搜索关键词
	query := ""
	if len(os.Args) > 1 {
		query = strings.ToLower(strings.Join(os.Args[1:], " "))
	}

	// 加载 Apple Color Emoji 字体
	fontBytes, err := ioutil.ReadFile("/System/Library/Fonts/Apple Color Emoji.ttc")
	if err != nil {
		fmt.Printf(`{"items":[{"title":"错误","subtitle":"无法读取系统字体","valid":false}]}`)
		return
	}
	tt, _ := opentype.Parse(fontBytes)
	face, _ := opentype.NewFace(tt, &opentype.FaceOptions{Size: 200, DPI: 72})

	// 收集结果
	items := []Item{}
	for _, e := range emojis {
		if query != "" &&
			!strings.Contains(strings.ToLower(e.Name), query) &&
			!strings.Contains(strings.ToLower(e.Category), query) &&
			!strings.Contains(strings.ToLower(e.Char), query) {
			continue
		}

		// 缓存 PNG
		code := fmt.Sprintf("%x", []rune(e.Char)[0]) // e.g. "1f600"
		iconPath := filepath.Join(iconDir, code+".png")
		if _, err := os.Stat(iconPath); os.IsNotExist(err) {
			renderEmojiPNG(e.Char, face, iconPath)
		}

		items = append(items, Item{
			Title:    e.Char,      // 在 Grid 底部一行小字
			Subtitle: e.Name,      // 辅助说明
			Arg:      e.Char,      // 传给后续节点
			Icon:     map[string]string{"path": iconPath}, // Grid 大图标
		})
	}

	if len(items) == 0 {
		items = append(items, Item{
			Title:    "未找到 Emoji",
			Subtitle: query,
			Arg:      "",
		})
	}

	output, _ := json.Marshal(map[string][]Item{"items": items})
	fmt.Println(string(output))
}
