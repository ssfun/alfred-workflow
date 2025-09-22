# Search for Github Alfred Workflow

一个基于 **Go** 编写的轻量级 Alfred Workflow，用于管理并快速搜索 GitHub 账户数据（Stars、Repos、Gists 等） 🚀。  
编译后直接运行，支持缓存、本地模糊搜索与缓存管理。  

## ✨ 功能特点

- 📂 **列出与搜索 GitHub Stars / Repos / Gists**
- 🔍 **全局搜索 GitHub 仓库**（使用 Search API，本地缓存加速）
- ⚡ **缓存支持**（SQLite 存储，支持刷新、清除、查看统计信息）
- 🧹 **缓存管理**（清除 Stars / Repos / Gists 缓存，或全部清除）
- 📑 **多种操作快捷键**
  - 回车：打开 GitHub 仓库页面
  - ⌘ Command + Enter：拷贝 Clone URL
  - ⌥ Option + Enter：拷贝 Repo URL

---

## 📥 安装

1. 下载最新的 **`GithubControl.alfredworkflow`** 文件。
2. 双击导入 Alfred（需要 Alfred Powerpack 支持 Workflow）。
3. 在 Alfred Preferences → Workflows → Github Control → 配置下列环境变量。

---

## ⚙️ 环境变量设置

- `GITHUB_USER`  
  GitHub 用户名（用于拉取公开数据时必填）

- `GITHUB_TOKEN`  
  GitHub API Token（可选，但推荐设置，否则 API 有速率限制；设置后能访问私有仓库）

- `MAX_REPOS`  
  个人 repos 结果最大输出条目（可选）

- `MAX_STARS`  
  个人 stars 结果最大输出条目（可选）

- `MAX_GISTS`  
  个人 gist 结果最大输出条目（可选）

- `GITHUB_CACHE_DIR`  
  缓存存放路径（可选）。若未定义，则默认：
  ```
  ~/Library/Caches/com.runningwithcrayons.Alfred/<workflow_bundle_id>
  ```

---

## 🔑 使用方式

调用关键字（可在 Workflow 配置中自定义关键字）：  

- `gs` → 查看 **Stars**
- `gm` → 查看 **我的 Repos**
- `gist` → 查看 **我的 Gists**
- `gh <keyword>` → **搜索 GitHub Repo**
- `gc` → 管理缓存（清除 Stars / Repos / Gists 缓存，或打开缓存目录）

### 缓存控制示例

- `清除所有缓存` → 删除全部缓存
- `清除 Stars 缓存` → 删除 Stars 缓存
- `📂 打开缓存目录` → Finder 打开缓存目录（`GITHUB_CACHE_DIR` 或默认路径）

---

## ⚡ 开发构建

### 本地构建

```bash
cd github-control-workflow
go mod tidy
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -buildvcs=false -o githubwf
```

> 默认编译出适配 macOS arm64 架构的二进制。  
> 如果需要 Intel 版，替换：`GOARCH=amd64`  

二进制体积可通过 `strip` 或 `upx` 进一步压缩。

---

## 📜 许可证

MIT License，欢迎自由使用与修改。  
