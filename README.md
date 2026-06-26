# 账易 (ZhangYi)

面向个人代账会计的轻量级代理记账系统。

## ✨ 特性

- 🖥️ **单机运行** — 双击即用，零配置，无需安装数据库
- 📊 **多账套管理** — 每个代理客户独立账套，互不干扰
- 🏭 **行业适配** — 7个行业科目模板（基础/制造/零售/服务/建筑/运输/农业）
- 📝 **凭证管理** — 录入、审核、记账、反审核、反记账、作废、恢复、批量操作、凭证号/摘要搜索全流程
- 📒 **账簿查询** — 科目余额表、总账、日记账、多栏账
- 📈 **标准报表** — 资产负债表、利润表、现金流量表
- 🔧 **辅助核算** — 7个维度，支持扩展字段、批量导入导出
- 👤 **用户管理** — JWT认证、登录/注册、密码修改
- 📱 **移动端适配** — 响应式布局，手机可用
- 🚀 **Docker部署** — Docker Compose 一键部署

## 📦 部署

### Docker Compose（推荐）

```yaml
services:
  zhangyi:
    image: ghcr.io/sevenclockseven/zhangyi:latest
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
```

```bash
docker compose up -d
```

访问 http://localhost:8080 ，首次运行自动创建管理员账号

### 源码编译

```bash
git clone https://github.com/sevenclockseven/zhangyi.git
cd zhangyi

# 前端
cd web && npm install --legacy-peer-deps && npm run build && cd ..

# 后端
GOPROXY=https://goproxy.cn,direct go build -o zhangyi .

# 运行
./zhangyi
```

## 🏗️ 功能清单

| 模块 | 功能 | 状态 |
|------|------|------|
| 账套管理 | CRUD、行业模板自动加载 | ✅ |
| 科目管理 | 树形展示、搜索、增删改、停用、同步模板 | ✅ |
| 凭证管理 | 录入、审核、记账、反审核、反记账、作废、批量 | ✅ |
| 账簿查询 | 科目余额表、总账、日记账、多栏账 | ✅ |
| 报表中心 | 资产负债表、利润表(税务格式)、现金流量表、费用统计、总账报表、应收/应付帐龄分析 | ✅ |
| 辅助核算 | 7维度CRUD、扩展字段、CSV导入导出、批量删除 |
| 凭证模板 | 模板CRUD、从模板加载凭证、模板管理页面 | ✅ |
| 期末处理 | 损益结转、期末结账/反结账（含独立页面） | ✅ |
| 期初余额 | ✅ | 期初余额录入、试算平衡校验、自动汇总 |
| 用户管理 | JWT认证、登录/注册、密码修改、管理员 | ✅ |
| 移动端 | 响应式布局、侧边栏抽屉、表格横向滚动 | ✅ |

## 🛠️ 开发

### 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.21+ + Gin + GORM |
| 数据库 | SQLite3（纯Go驱动，无CGO依赖） |
| 前端 | Vue3 + Element Plus + Vite |
| 容器 | Docker + Docker Compose |
| CI/CD | GitHub Actions → ghcr.io |

### 项目结构

```
zhangyi/
├── main.go                  # 程序入口
├── internal/
│   ├── api/                 # HTTP API
│   │   ├── routes.go        # 路由注册
│   │   ├── handlers.go      # 账套/科目/凭证/报表/辅助核算
│   │   ├── auth_handlers.go # 用户认证
│   │   └── closing_handlers.go # 期末处理
│   ├── models/              # 数据模型
│   ├── services/            # 业务逻辑
│   └── middleware/           # JWT认证中间件
├── web/src/views/           # Vue页面组件
├── templates/               # 会计科目模板 (JSON)
├── Dockerfile               # 多阶段构建
└── docker-compose.yml       # 部署配置
```

### API 列表

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | POST /api/auth/login | 登录 |
| 认证 | POST /api/auth/register | 注册 |
| 账套 | GET/POST/PUT/DELETE /api/books | 账套CRUD |
| 科目 | GET/POST/PUT/DELETE /api/books/:id/accounts | 科目CRUD |
| 凭证 | GET/POST/PUT/DELETE /api/books/:id/vouchers | 凭证CRUD |
| 凭证 | POST .../review,unreview,post,void,restore,unpost | 审核/记账/作废/恢复 |
| 凭证 | POST .../batch-review,batch-post | 批量审核/记账 |
| 账簿 | GET /api/books/:id/ledger/general | 总账 |
| 账簿 | GET /api/books/:id/ledger/journal | 日记账 |
| 账簿 | GET /api/books/:id/ledger/multi-column | 多栏账 |
| 报表 | GET /api/books/:id/reports/balance-sheet | 资产负债表 |
| 报表 | GET /api/books/:id/reports/income-statement | 利润表 |
| 报表 | GET /api/books/:id/reports/cash-flow | 现金流量表 |
| 报表 | GET /api/books/:id/reports/account-balance | 科目余额表 |
| 辅助 | GET/POST/PUT/DELETE /api/books/:id/aux/:type | 辅助核算CRUD |
| 辅助 | GET .../export | CSV导出 |
| 辅助 | POST .../import | CSV导入 |
| 辅助 | POST .../batch-delete | 批量删除 |
| 期初 | GET/POST /api/books/:id/opening-balances | 期初余额查询/保存 |
| 期末 | POST /api/books/:id/closing/auto-transfer | 损益结转 |
| 期初 | GET/POST /api/books/:id/opening-balances | 期初余额查询/保存 |
| 期末 | POST /api/books/:id/closing/close | 期末结账 |
| 期初 | GET/POST /api/books/:id/opening-balances | 期初余额查询/保存 |
| 期末 | POST /api/books/:id/closing/unclose | 反结账 |

## 📋 版本历史

| 版本 | 日期 | 内容 |
|------|------|------|
| v0.1.0 | 2026-06-25 | 初始版本：账套/科目/凭证/报表/辅助核算/期末处理 |
| v0.2.0 | 2026-06-25 | 用户管理(JWT)、登录页、期末处理、Logo |
| v0.3.0 | 2026-06-25 | 现金流量表、凭证作废、日记账、多栏账、移动端适配 |
| v0.3.1 | 2026-06-25 | 辅助核算扩展字段、CSV批量导入导出、凭证号/摘要搜索 |
| v0.5.0 | 2026-06-25 | 自定义报表引擎、凭证模板、数据导入导出、科目在线更新 |
| v0.5.1 | 2026-06-26 | 修复工作台系统信息版本号不同步 |
| v0.5.2 | 2026-06-26 | 修复报表导出（CSV/XLSX/打印）、自定义报表编辑、去掉PDF选项 |
| v0.5.3 | 2026-06-26 | 自定义报表导出修复、打印预览优化、使用说明完善 |

## 📄 许可证

MIT License
