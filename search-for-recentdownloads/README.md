# Search for Recent Downloads Files

这是一个用 Go 语言编写的 **Alfred Workflow 文件搜索工具**，支持中文/拼音搜索、多音字处理、文件类型过滤、排序模式自定义。

---

## ✨ 功能特性

- 🔎 **中文搜索**：直接输入文件名中的汉字即可匹配  
- 🈶 **拼音搜索**：支持全拼搜索（`zhongqi`）、首字母搜索（`zq`）、模糊匹配  
- 🎵 **多音字支持**：通过 `polyphonic.json` 或内置表解决 `重/行/长` 等拼音歧义  
- 📂 **文件类型过滤**（支持前后位置）：
  - `.dir 项目` → 搜索目录
  - `项目 .dir` → 同上
  - `.file 报告` / `报告 .file` → 搜索文件
  - `.pdf 报告` / `报告 .pdf` → 搜索 PDF 文件
- 🔨 **排序模式**（通过 Alfred 配置环境变量）：
  - 修改时间：`mod_time_desc` / `mod_time_asc`
  - 创建时间：`add_time_desc` / `add_time_asc`
  - 文件名排序：`filename_asc` / `filename_desc`
- 🕒 **时间精确**：支持 **创建时间 (Birthtime)** 作为排序依据（macOS 专用）
- 📏 **文件大小换算**：显示为 `123 KB / 2.5 MB / 1.3 GB`
- 👁‍🗨 **隐藏文件自动忽略**

---

## ⚙️ 安装 & 使用

### 1. 克隆并编译

```bash
git clone https://github.com/yourname/alfred-search-go.git
cd alfred-search-go
go build -o search main.go
```

将生成的 `search` 可执行文件放到 Alfred Workflow 脚本动作中。

---

### 2. Alfred Workflow 配置

在 Workflow Script Filter 中使用：

```bash
./search "{query}"
```

---

### 3. 设置环境变量（用于排序模式）

在 Workflow 的 **Environment Variables** 添加以下配置：

| 变量名               | 值    | 说明                                    |
|----------------------|-------|---------------------------------------|
| `mod_time_desc`      | `rdn` | 按修改时间排序，最新在前                |
| `mod_time_asc`       | `rod` | 按修改时间排序，最旧在前                |
| `add_time_desc`      | `cdn` | 按创建时间排序，最新在前                |
| `add_time_asc`       | `cod` | 按创建时间排序，最旧在前                |
| `filename_asc`       | `az`  | 文件名 A→Z 排序                        |
| `filename_desc`      | `za`  | 文件名 Z→A 排序                        |
| `alfred_workflow_keyword` | `rdn` | 设置默认排序模式（对应上面某一模式）  |

> ⚡️ 注意：`alfred_workflow_keyword` 的值应该等于你希望默认模式的变量值，比如：  
> - 设置为 `rdn` → 默认 `mod_time_desc`  
> - 设置为 `za` → 默认 `filename_desc`

---

### 4. 搜索示例

假设目录中有以下文件：

- `报告.pdf`
- `项目计划`（文件夹）
- `test.txt`

搜索命令及效果：

| 输入            | 结果                   |
|-----------------|------------------------|
| `报告`          | 匹配 `报告.pdf`        |
| `.pdf 报告`     | 匹配 `报告.pdf`        |
| `报告 .pdf`     | 同上                   |
| `.dir 项目`     | 匹配目录 `项目计划`    |
| `项目 .dir`     | 同上                   |
| `.file test`    | 匹配 `test.txt`        |
| `test .file`    | 匹配 `test.txt`        |
| `zhbg`          | 匹配拼音首字母的文件名 |
| `baogao`        | 匹配拼音全拼的文件名   |

---

## 🔧 polyphonic.json (可选)

用于自定义多音字拼音转换规则。格式：

```json
{
  "重": ["chong", "zhong"],
  "长": ["chang", "zhang"],
  "行": ["hang", "xing"]
}
```

程序会优先读取本地 `polyphonic.json`，若不存在则使用默认内置表。

---

👉 这个 README 是否需要我再给你一个 **中文简化版（只包含用户使用说明，不含实现细节）**，方便直接发到 GitHub 项目首页用？
