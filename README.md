# 账易 (ZhangYi)

面向个人代账会计的轻量级代理记账系统。

## ✨ 特性

- 🖥️ **单机运行** — 双击即用，零配置，无需安装数据库
- 📊 **多账套管理** — 每个代理客户独立账套，互不干扰
- 🏭 **行业适配** — 内置多行业会计科目模板（制造业/零售业/服务业等）
- 📝 **凭证管理** — 录入、审核、记账全流程
- 📈 **标准报表** — 资产负债表、利润表、现金流量表
- 🔧 **自定义报表** — 取数公式引擎，灵活出表
- 🔄 **科目可更新** — 会计准则变更时推送更新包
- 🚀 **可扩展** — 架构预留多人协作能力

## 📦 下载

从 [Releases](https://github.com/sevenclockseven/zhangyi/releases) 页面下载对应平台的可执行文件：

| 平台 | 文件 |
|------|------|
| Windows | `zhangyi-windows-amd64.exe` |
| Linux x64 | `zhangyi-linux-amd64` |
| Linux ARM64 | `zhangyi-linux-arm64` |
| macOS Intel | `zhangyi-macos-amd64` |
| macOS Apple Silicon | `zhangyi-macos-arm64` |

## 🚀 快速开始

```bash
# Windows
zhangyi-windows-amd64.exe

# Linux/macOS
chmod +zhangyi-linux-amd64
./zhangyi-linux-amd64
```

启动后访问 http://localhost:8080

## 🛠️ 开发

### 环境要求

- Go 1.21+
- Node.js 20+
- GCC (CGO for SQLite)

### 本地开发

```bash
# 安装 Go 依赖
go mod tidy

# 启动后端（热重载可选 air）
go run cmd/server/main.go

# 启动前端开发服务器
cd web
npm install
npm run dev
```

### 构建

```bash
# 构建前端
cd web && npm run build && cd ..

# 构建可执行文件
go build -o zhangyi cmd/server/main.go
```

### 发布

```bash
# 打 tag 触发 GitHub Actions 自动构建
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions 会自动构建 5 个平台的可执行文件并创建 Release。

## 📁 项目结构

```
zhangyi/
├── cmd/server/          # 程序入口
├── internal/
│   ├── api/             # HTTP API 处理器
│   ├── models/          # 数据模型
│   ├── services/        # 业务逻辑
│   └── middleware/       # 中间件
├── web/                 # Vue3 前端
├── templates/           # 会计科目模板 (JSON)
├── reports/             # 报表模板 (JSON)
├── .github/workflows/   # GitHub Actions
└── data/                # SQLite 数据库 (运行时生成)
```

## 📋 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go + Gin + GORM |
| 数据库 | SQLite3 |
| 前端 | Vue3 + Element Plus |
| CI/CD | GitHub Actions |

## 📄 许可证

MIT License
