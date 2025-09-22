## 🔍 file-search

一个使用 **Go 编写** 的 **Alfred Workflow 文件搜索工具**，支持：
- **拼音搜索**：首字母、全拼（精确/前缀）  
- **多音字修复**：支持 `银行` → `yinhang`  
- **模糊容错**：轻微拼写错误也可匹配（低权重兜底）  
- **类型过滤**：支持 `dir` / `file` / `.pdf` 等过滤  
- **结果优化**：精准打分排序，保证高相关结果优先显示  

相比基于 `mdfind` 的方案，本工具更灵活、可控，适合复杂中文拼音检索场景。  

---

## 🚀 安装与构建

### 1. 构建二进制
```bash
git clone https://github.com/yourname/file-search.git
cd file-search
go build -o file-search
```

### 2. Alfred Workflow 配置
在 Alfred **Workflow** 中：
- 新建一个 **Script Filter**  
- **Keyword** 输入： `fs` （例如）  
- **Script** 设置为：  

```bash
./file-search "{query}"
```

---

## ⚙️ 配置说明

可通过 **环境变量** 配置搜索参数：  

| 环境变量        | 默认值                            | 描述 |
|----------------|----------------------------------|------|
| `SEARCH_DIRS`  | `~/Documents,~/Desktop,~/Downloads` | 搜索目录，逗号分隔 |
| `EXCLUDES`     | `.git,__pycache__,node_modules,.DS_Store` | 排除目录/文件 |
| `MAX_RESULTS`  | 100                              | 最多返回结果数 |
| `DEBUG`        | `0`                              | 调试模式（设为 `1` 会输出 Debug 到 stderr） |

示例配置：  
```bash
export SEARCH_DIRS="$HOME/Documents,$HOME/Projects"
export EXCLUDES=".git,tmp,node_modules"
```

---

## 🔎 使用方法

### 基础搜索
```bash
fs 银行信息
fs yhxx
fs yinhang
```

- `yhxx` → 命中 “银行信息.txt”（首字母缩写匹配，权重最高）  
- `yinhang` → 命中 “银行信息.txt”（全拼前缀匹配）  
- `yinhangxinxi` → 命中 “银行信息.txt”（全拼精确匹配）  

### 过滤器
```bash
fs yhxx dir      # 搜索目录类型
fs yhxx file     # 搜索文件
fs yhxx .pdf     # 搜索 PDF 文件
```

### 模糊场景
```bash
fs yinhagxinxi   # 少打一个字母，也能以低权重匹配“银行信息.txt”
```

---

## ⚡ 匹配权重逻辑

匹配优先级（由高到低）：  

1. **英文文件**  
   - 完全匹配：500  
   - 前缀匹配：450  
   - 包含匹配：400  

2. **中文拼音**  
   - 首字母完整匹配：380  
   - 全拼精确匹配：350  
   - 全拼前缀匹配：300  
   - 首字母子序列匹配：250  

3. **兜底策略**  
   - 多音字重试：180  
   - 拼音模糊（fuzzy）：80  

👉 这样确保输入缩写/拼音时，真正目标文件总能优先命中，杂项结果排在后面。  

---

## 🛠 调试模式

可开启 `DEBUG=1` 查看调试日志：

```bash
export DEBUG=1
./file-search yhxx
```

输出示例（stderr）：

```
DEBUG: 银行信息.txt → q: yhxx full: yinhangxinxi initials: yhxx score: 380
DEBUG: 四合一嵌入式.jpg → q: yhxx full: siheyiqianru... initials: shyqrsw... score: 0
```

---

## 📂 项目结构

```
file-search/
├── main.go       # 程序入口
├── config.go     # 配置管理
├── pinyin.go     # 拼音处理 + 多音字修复
├── match.go      # 匹配逻辑
├── search.go     # 文件扫描逻辑
└── alfred.go     # Alfred JSON 输出
```

---

## ❤️ 致谢

- [go-pinyin](https://github.com/mozillazg/go-pinyin)：汉字转拼音库  
