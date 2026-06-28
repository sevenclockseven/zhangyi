# 易记 (YiJi)

面向个人代账会计的轻量级代理记账系统。

## ✨ 特性

- 🖥️ **单机运行** — 双击即用，零配置，无需安装数据库
- 📊 **多账套管理** — 每个代理客户独立账套，互不干扰
- 🏭 **行业适配** — 7个行业科目模板（基础/制造/零售/服务/建筑/运输/农业）
- 📝 **凭证管理** — 录入、审核、记账、反审核、反记账、作废、恢复、批量操作
- 📒 **账簿查询** — 科目余额表、总账、日记账、多栏账
- 📈 **标准报表** — 资产负债表、利润表、现金流量表、自定义报表引擎
- 🔧 **辅助核算** — 7个维度 + 现金流量项目，支持扩展字段、批量导入导出
- 👤 **用户管理** — JWT认证、登录/注册、密码修改、角色权限
- 🔐 **账套权限** — 管理员分配用户可访问的账套及读写权限
- 📋 **操作日志** — 自动记录非GET请求，按模块/操作/用户筛选
- 💾 **自动备份** — 定时备份(SQLite/PG)、手动备份、一键恢复
- 🗄️ **数据库可切换** — SQLite(默认零配置) / PostgreSQL(生产环境)
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
      - zhangyi-data:/app/data
      - zhangyi-backups:/app/backups
    environment:
      - TZ=Asia/Shanghai
      # 数据库切换（默认SQLite，取消注释启用PostgreSQL）
      # - DB_DRIVER=postgres
      # - DB_DSN=host=postgres user=zhangyi password=*** dbname=zhangyi port=5432 sslmode=disable
      # 备份配置
      # - BACKUP_SCHEDULE=24h  # disabled/hourly/6h/24h
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  zhangyi-data:
  zhangyi-backups:
```

### Windows 本地运行

#### 默认 SQLite（零配置，推荐）

直接双击 `zhangyi.exe` 即可，数据自动保存在 `data/zhangyi.db`。

#### 切换到 PostgreSQL

**方式一：CMD 命令行**

```cmd
set DB_DRIVER=postgres
set DB_DSN=host=localhost user=zhangyi password=你的密码 dbname=zhangyi port=5432 sslmode=disable
zhangyi.exe
```

**方式二：PowerShell**

```powershell
$env:DB_DRIVER="postgres"
$env:DB_DSN="host=localhost user=zhangyi password=你的密码 dbname=zhangyi port=5432 sslmode=disable"
.\zhangyi.exe
```

**方式三：写 bat 脚本（一劳永逸）**

创建 `start.bat` 文件，内容如下：

```bat
@echo off
set DB_DRIVER=postgres
set DB_DSN=host=localhost user=zhangyi password=你的密码 dbname=zhangyi port=5432 sslmode=disable
zhangyi.exe
```

以后双击 `start.bat` 即可启动。

> ⚠️ 切换数据库前请确保 PostgreSQL 已安装并运行，且已创建 `zhangyi` 数据库。

### Mac/Linux 本地运行

```bash
# 默认SQLite
chmod +x zhangyi
./zhangyi

# 切换PostgreSQL
DB_DRIVER=postgres DB_DSN="host=localhost user=zhangyi password=*** dbname=zhangyi port=5432" ./zhangyi
```

## 🔧 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 服务端口 |
| DB_DRIVER | sqlite | 数据库驱动：sqlite / postgres |
| DB_DSN | data/zhangyi.db | 数据库连接串（SQLite为文件路径，PG为连接串） |
| BACKUP_SCHEDULE | 24h | 备份周期：disabled / hourly / 6h / 24h |
| BACKUP_DIR | backups | 备份文件目录 |

## 🔐 默认账号

- 管理员：admin / admin123

## 📋 版本

当前版本：v0.8.4

## 📝 版本记录

| 版本 | 日期 | 内容 |
|------|------|------|
| v0.8.4 | 2026-06-28 | 科目余额表树形化+可折叠+一级科目颜色标识+父节点金额自动汇总；账簿查询拆分为总账/现金日记账/银行日记账；修复辅助核算扩展字段 |
| v0.8.3 | 2026-06-28 | 修复辅助核算扩展字段丢失问题 |
| v0.8.2 | 2026-06-28 | 修复账套权限过滤和备份恢复功能 |
| v0.8.1 | 2026-06-28 | 修复登录后用户信息刷新+重置密码弹窗优化 |
| v0.8.0 | 2026-06-28 | Phase3 完成：数据库可切换(SQLite/PG)+自动备份+操作日志+账套权限+前端Settings完善 |
| v0.7.5 | 2026-06-28 | 科目余额表树形化+可折叠+一级科目颜色标识+父节点金额自动汇总 |
| v0.7.4 | 2026-06-26 | 期末处理逻辑修正(结转简化+反结账自动删除+事务一致性) |
| v0.7.0 | 2026-06-26 | 拆分handlers.go+前端API层+Element Plus按需引入 |
| v0.6.0 | 2026-06-26 | 模板体系重构：行业×纳税人×准则科目模板 |
| v0.5.0 | 2026-06-25 | Phase2 完成：自定义报表引擎+凭证模板+数据导入导出 |
| v0.3.0 | 2026-06-25 | 现金流量表+凭证作废+日记账+多栏账+移动端适配 |
| v0.2.0 | 2026-06-25 | 用户管理(JWT)+登录页+期末处理 |
| v0.1.0 | 2026-06-25 | 初始版本：账套/科目/凭证/报表/辅助核算 |
