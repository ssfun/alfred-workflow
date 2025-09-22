package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/mozillazg/go-pinyin"
)

var a = pinyin.NewArgs()

// 多音字字典
var polyphonic = map[rune][]string{}

// 加载外部/默认字典
func loadPolyphonicDict(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		polyphonic = map[rune][]string{
			'行': {"hang", "xing"},
	        '长': {"chang", "zhang"},
	        '重': {"chong", "zhong"},
	        '乐': {"le", "yue"},
	        '处': {"chu", "cu"},
	        '还': {"hai", "huan"},
	        '藏': {"cang", "zang"},
	        '假': {"jia", "jie"},
	        '召': {"zhao", "shao"},
		}
		return
	}
	tmp := make(map[string][]string)
	if err := json.Unmarshal(data, &tmp); err == nil {
		for k, v := range tmp {
			runes := []rune(k)
			if len(runes) > 0 {
				polyphonic[runes[0]] = v
			}
		}
	}
}

// 拼音缓存
type PinyinCache struct {
	mu    sync.RWMutex
	cache map[string][2]string
}

func NewPinyinCache() *PinyinCache {
	return &PinyinCache{cache: make(map[string][2]string)}
}

func (pc *PinyinCache) Get(name string) (string, string) {
	pc.mu.RLock()
	if val, ok := pc.cache[name]; ok {
		pc.mu.RUnlock()
		return val[0], val[1]
	}
	pc.mu.RUnlock()

	var fullParts []string
	var initialParts []string

	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			// 多音字优先
			if alts, ok := polyphonic[r]; ok && len(alts) > 0 {
				choose := alts[0]
				fullParts = append(fullParts, choose)
				initialParts = append(initialParts, string(choose[0]))
			} else {
				py := pinyin.LazyPinyin(string(r), a)
				if len(py) > 0 {
					fullParts = append(fullParts, py[0])
					initialParts = append(initialParts, string(py[0][0]))
				}
			}
		}
	}
	full := strings.Join(fullParts, "")
	initials := strings.Join(initialParts, "")

	pc.mu.Lock()
	pc.cache[name] = [2]string{full, initials}
	pc.mu.Unlock()
	return full, initials
}
