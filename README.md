# 账易 (ZhangYi)

面向个人代账会计的轻量级代理记账系统。

## ✨ 特性

- 🖥️ **单机运行** — 双击即用，零配置，无需安装数据库
- 📊 **多账套管理** — 每个代理客户独立账套，互不干扰
- 🏭 **行业适配** — 7个行业科目模板（基础/制造/零售/服务/建筑/运输/农业）
- 📝 **凭证管理** — 录入、审核、记账、批量操作全流程
- 📈 **标准报表** — 资产负债表、利润表、科目余额表
- 🔧 **辅助核算** — 客户/供应商/部门/项目/员工/仓库/银行账号
- 🔄 **科目可更新** — 会计准则变更时推送更新包
- 🚀 **Docker部署** — Docker Compose 一键部署

## 📦 下载

### Docker 部署（推荐）

```bash
# 拉取镜像
docker pull ghcr.io/sevenclockseven/zhangyi:0.1

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  zhangyi:
    image: ghcr.io/sevenclockseven/zhangyi:0.1
    container_name: zhangyi
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./backups:/app/backups
    environment:
      - TZ=Asia/Shanghai
      - PORT=8080
    networks:
      - zhangyi-net

networks:
  zhangyi-net:
    ipam:
      config:
        - subnet: 192.168.88.0/24
EOF

# 启动
docker compose up -d
```

访问 http://localhost:8080

### 二进制下载

从 [Releases](https://github.com/sevenclockseven/zhangyi/releases) 页面下载：

| 平台 | 文件 |
|------|------|
| Windows | `zhangyi-windows-amd64.exe` |
| Linux x64 | `zhangyi-linux-amd64` |
| Linux ARM64 | `zhangyi-linux-arm64` |
| macOS Intel | `zhangyi-macos-amd64` |
| macOS Apple Silicon | `zhangyi-macos-arm64` |

```bash
# 运行
./zhangyi-linux-amd64
```

## 🏗️ 功能清单

| 模块 | 状态 | 说明 |
|------|------|------|
| 账套管理 | ✅ | CRUD、行业模板自动加载 |
| 科目管理 | ✅ | 树形展示、搜索、增删改、停用 |
| 凭证管理 | ✅ | 录入、审核、记账、批量操作 |
| 账簿查询 | ✅ | 科目余额表、总账 |
| 报表中心 | ✅ | 资产负债表、利润表 |
| 辅助核算 | ✅ | 7个维度（客户/供应商/部门/项目/员工/仓库/银行） |
| 期末处理 | 🔜 | 损益结转、折旧、结账 |
| 报表导出 | 🔜 | Excel/PDF/打印 |

## 🛠️ 开发

### 环境要求

- Go 1.21+
- Node.js 20+
- Docker（部署用）

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/sevenclockseven/zhangyi.git
cd zhangyi

# 安装前端依赖并构建
cd web && npm install --legacy-peer-deps && npm run build && cd ..

# 运行后端
go run main.go
```

### 发布流程

```bash
# 打 tag 触发 GitHub Actions
git tag v0.2.0
git push origin v0.2.0

# Actions 自动：
# 1. 构建前端
# 2. 编译5平台二进制 → 创建 Release
# 3. 构建Docker镜像 → 推到 ghcr.io

# 部署
docker compose pull && docker compose up -d
```

## 📁 项目结构

```
zhangyi/
├── main.go                  # 程序入口
├── internal/
│   ├── api/                 # HTTP API
│   │   ├── routes.go        # 路由注册
│   │   └── handlers.go      # 请求处理
│   ├── models/              # 数据模型
│   └── services/            # 业务逻辑
├── web/                     # Vue3 前端
│   ├── src/views/           # 页面组件
│   ├── src/router/          # 路由
│   └── package.json
├── templates/               # 会计科目模板 (JSON)
├── docs/                    # 文档
├── .github/workflows/       # GitHub Actions
├── Dockerfile               # 多阶段构建
└── docker-compose.yml       # 部署配置
```

## 📋 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go + Gin + GORM |
| 数据库 | SQLite3（纯Go驱动，无CGO依赖） |
| 前端 | Vue3 + Element Plus + Vite |
| 容器 | Docker + Docker Compose |
| CI/CD | GitHub Actions → ghcr.io |

## 📄 许可证

MIT License
