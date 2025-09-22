# README.md

# Alfred Emoji Workflow

一个基于 **Go** 编写的高性能 Emoji 搜索与管理 Alfred Workflow。  
支持 **分类搜索、Grid 大图标展示、最近使用记录、资源在线更新**，完全依赖 **单一二进制**，无第三方依赖。  

---

## ✨ 功能特性

- 🔍 **Emoji 搜索**：支持关键字，结果中自动跳过缺图标 Emoji  
- 🖼️ **大图标 Grid 展示**：利用 Noto Emoji PNG，本地缓存资源，确保显示完整美观  
- 🕒 **最近使用记录**：自动保存最近 8 个使用过的 Emoji，下次调用时置顶展示  
- 🔧 **辅助工具**（utils 模式）：  
  1. ⬇️ 更新 `emoji.json` 数据文件（来自 [amio/emoji.json](https://github.com/amio/emoji.json)）  
  2. ⬇️ 下载 Emoji 图标资源（补缺 / 覆盖）（来自 [Google Noto Emoji](https://github.com/googlefonts/noto-emoji)）  
  3. 🗑️ 清除最近使用记录  

---

## 📂 Workflow 目录结构

```
workflow/
 ├── main.go         # 查询和 recent 更新
 ├── utils.go        # 工具模式 (更新 JSON、下载图标、清除 recent)
 ├── emoji           # go build 输出的二进制
 ├── emoji.json      # emoji 数据文件
 ├── icons/          # PNG 图标存放目录 (Noto Emoji)
 └── recent.json     # 最近使用记录
```

---

## ⚡ 安装

1. 克隆仓库并进入目录
   ```bash
   git clone https://github.com/yourname/alfred-emoji-go.git
   cd alfred-emoji-go/workflow
   ```

2. 编译 Workflow 二进制
   ```bash
   go build -o emoji .
   ```

3. 更新数据文件 & 资源
   ```bash
   ./emoji utils update-json           # 更新 emoji.json
   ./emoji utils download-skip         # 补齐缺失 PNG
   # 或
   ./emoji utils download-overwrite    # 全量覆盖下载
   ```

4. 在 Alfred 中创建 Workflow，将 `emoji` 二进制添加为 Script Filter。

---

## 🚀 使用方法

### 1. 搜索 Emoji

- 打开 Alfred，输入：

  ```
  emoji smile
  ```

- 会搜索所有与 `smile` 相关的 Emoji，并显示大图标。

### 2. 最近使用

- 输入 `emoji`（无参数），顶部会展示最近使用的 Emoji（最多 8 个）。  
- 选中一个 Emoji，会：
  1. 自动复制到剪贴板
  2. 自动写入 `recent.json`，下次优先展示  

### 3. 工具菜单（utils）

- 打开 Alfred，输入：

  ```
  emojiutils
  ```

- 会出现 4 个选项：
  1. **更新 emoji.json** （Subtitle 显示文件最近更新时间）  
  2. **更新 Emoji 资源（增补缺失）**  
  3. **更新 Emoji 资源（覆盖全部）**  
  4. **清除最近使用记录**  

- 选择后自动触发对应操作。

---

## 🔧 开发说明

- 项目完全用 **Go** 编写，依赖标准库，无需 Python/Node.js。  
- 数据： [amio/emoji.json](https://github.com/amio/emoji.json)  
- 图标： [Noto Emoji PNG (128px)](https://github.com/googlefonts/noto-emoji/tree/main/png/128)  
- 本地 icon 命名规则：  
  ```
  1f469-1f3fc-200d-1f9b2.png
  ```
  （和 `emoji.json` 的 Codes 一致）

---

## 📝 License

MIT
