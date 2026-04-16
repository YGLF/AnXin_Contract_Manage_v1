# 安信合同管理系统

基于 Go + Gin + MySQL（后端）和 Vue3 + Element Plus（前端）的智能合同管理系统，支持用户级别细粒度权限控制、数据完整性验证和 SHA-256 密码杂凑。

## 目录

- [功能模块](#功能模块)
- [技术栈](#技术栈)
- [快速部署](#快速部署)
  - [Docker 一键部署（推荐）](#docker-一键部署推荐)
  - [手动部署](#手动部署)
- [项目结构](#项目结构)
- [权限系统](#权限系统)
- [API 接口](#api-接口)
- [环境变量](#环境变量)
- [加密服务](#加密服务)
- [常见问题](#常见问题)

## 功能模块

| 模块 | 说明 |
|------|------|
| 用户权限管理 | 用户注册、登录、角色管理（超级管理员、经理、销售、审计管理员）、用户级别细粒度权限控制 |
| 客户/供应商管理 | 客户信息增删改查、客户分类、信用等级 |
| 合同管理 | 合同信息管理、合同分类管理、合同状态跟踪 |
| 合同执行跟踪 | 进度跟踪、付款记录、执行阶段管理 |
| 审批流程 | 合同审批、三级审批（销售总监→技术总监→财务总监）、审批记录查询 |
| 状态变更审批 | 关键状态变更（归档、终止、执行中、待付款）需管理员审批 |
| 合同生命周期 | 完整的合同状态变更历史记录 |
| 合同归档 | 已完成合同归档管理、到期自动归档、定时任务通知 |
| 到期提醒 | 合同到期提醒、续期管理、提醒通知、过期强提醒 |
| 统计报表 | 数据统计分析、图表展示 |
| 文档管理 | 合同文件上传、版本管理 |
| 加密服务 | 预留与服务器密码机（HSM）对接接口、支持 SM4/AES 加密 |
| 数据安全 | SHA-256 密码杂凑验证、30分钟无操作自动退出 |

## 技术栈

### 后端
| 技术 | 说明 |
|------|------|
| Go 1.21+ | 编程语言 |
| Gin | Web 框架 |
| GORM | ORM 库 |
| MySQL 8.0 | 数据库 |
| JWT | 用户认证 |
| bcrypt | 密码加密 |
| SHA-256 | 密码杂凑验证 |

### 前端
| 技术 | 说明 |
|------|------|
| Vue 3 | 前端框架 |
| Vite | 构建工具 |
| Element Plus | UI 组件库 |
| Pinia | 状态管理 |
| Vue Router | 路由管理 |
| Axios | HTTP 客户端 |
| ECharts | 数据可视化 |

## 快速部署

### Docker 一键部署（推荐）

适用于 CentOS 7 生产环境，自动安装 Docker、配置环境、启动服务。

#### 前置要求

- CentOS 7.x
- Root 权限
- 可访问互联网

#### 部署步骤

```bash
# 1. 上传项目到服务器 /opt/ 目录

# 2. 进入 docker 目录
cd /opt/AnXin_Contract_Manage_encrytp_v1.1/docker

# 3. 添加执行权限
chmod +x deploy.sh

# 4. 执行一键部署
./deploy.sh
```

部署过程会自动完成：
- ✓ 安装 Docker 24.x
- ✓ 安装 Docker Compose
- ✓ 配置镜像加速器（国内源）
- ✓ 生成安全随机密码
- ✓ 构建并启动 MySQL + Go后端 + Nginx前端
- ✓ 配置定时备份（每天凌晨3点）

#### 部署完成后

| 服务 | 地址 |
|------|------|
| 前端 | http://服务器IP/ |
| 后端 API | http://服务器IP:8000 |
| 数据库 | localhost:3306 |

| 账号 | 用户名 | 密码 |
|------|--------|------|
| 超级管理员 | admin | （部署时生成的随机密码） |
| 审计管理员 | auditadmin | （部署时生成的随机密码） |

#### 常用命令

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 启动服务
docker-compose up -d

# 手动备份数据库
./backup.sh
```

### 手动部署

#### 前置要求

- Go 1.21+
- MySQL 8.0+
- Node.js 16+
- Git

#### 后端部署

```bash
# 克隆项目
git clone https://github.com/YGLF/AnXin_Contract_Manage_v1.git
cd AnXin_Contract_Manage

# 安装依赖
go mod download

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，修改数据库配置

# 创建数据库
mysql -u root -p -e "CREATE DATABASE contract_manage CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 启动服务
go run main.go
```

后端默认运行在 http://localhost:8000

#### 前端部署

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

### Docker Compose 部署

```bash
# 复制环境变量配置
cp .env.example .env

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

## 项目结构

```
AnXin_Contract_Manage/
├── config/              # 配置模块
├── handlers/            # HTTP 处理器
│   ├── auth.go          # 认证
│   ├── contract.go      # 合同管理
│   ├── customer.go      # 客户管理
│   ├── approval.go      # 审批管理
│   ├── audit.go         # 审计日志
│   ├── workflow.go      # 工作流
│   └── crypto.go        # 加密服务
├── middleware/          # 中间件
│   ├── auth.go          # JWT 认证
│   ├── security.go      # 安全中间件
│   └── validator.go     # 输入验证
├── models/              # 数据模型
├── services/            # 业务逻辑层
├── routes/              # 路由配置
├── crypto/              # 加密服务模块
├── migrations/          # 数据库迁移
├── scripts/             # 测试脚本
├── docs/                # 开发文档
├── frontend/            # 前端项目
│   └── src/
│       ├── api/         # API 接口
│       ├── router/      # 路由配置
│       ├── store/       # 状态管理
│       ├── utils/       # 工具函数
│       └── views/       # 页面组件
├── docker/              # 部署脚本
│   ├── deploy.sh        # 一键部署脚本
│   ├── .env.production  # 生产环境配置
│   └── 部署说明.md      # 部署文档
├── main.go              # 后端入口
├── docker-compose.yml   # Docker 配置
└── Dockerfile           # 后端构建文件
```

## 权限系统

### 角色说明

| 角色 | 标识 | 权限 |
|------|------|------|
| 超级管理员 | admin | 所有权限 |
| 经理 | manager | 仪表盘、合同管理、客户管理、审批 |
| 销售 | user | 仪表盘、查看/创建合同、查看/创建客户 |
| 审计管理员 | audit_admin | 仪表盘、查看审计、查看合同/客户/审批 |

### 权限控制模式

采用 **角色权限 + 用户自定义权限** 双重控制：
- 角色权限：定义每个角色默认拥有的权限
- 用户自定义权限：基于角色权限追加额外权限（追加模式）
- 最终权限 = 角色权限 ∪ 用户自定义权限

## API 接口

### 公共端点（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| GET | / | 服务信息 |
| GET | /health | 健康检查 |

### 认证

除登录注册外，所有 API 需要 JWT Token 认证：

```bash
# 登录获取 Token
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin@123456", "password_hash": "<SHA-256>"}'

# 使用 Token 请求
curl -X GET http://localhost:8000/api/contracts \
  -H "Authorization: Bearer <token>"
```

### 完整 API 接口列表

#### 公共端点（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| GET | / | 服务信息 |
| GET | /health | 健康检查 |

#### 认证管理 (/api/auth)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/refresh | 刷新Token |
| GET | /api/auth/me | 获取当前用户信息 |
| GET | /api/auth/users | 用户列表（需管理权限） |
| GET | /api/auth/users/:user_id | 用户详情 |
| PUT | /api/auth/users/:user_id | 更新用户 |
| DELETE | /api/auth/users/:user_id | 删除用户 |
| POST | /api/auth/users/:user_id/reset-password | 重置密码 |
| POST | /api/auth/users/:user_id/unlock | 解锁用户 |
| GET | /api/auth/users/:user_id/lock-status | 锁定状态 |

#### 客户管理 (/api/customers)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/customers | 客户列表 |
| GET | /api/customers/:customer_id | 客户详情 |
| POST | /api/customers | 创建客户 |
| PUT | /api/customers/:customer_id | 更新客户 |
| DELETE | /api/customers/:customer_id | 删除客户 |

#### 合同类型管理 (/api/contract-types)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contract-types | 类型列表 |
| POST | /api/contract-types | 创建类型 |
| PUT | /api/contract-types/:type_id | 更新类型 |
| DELETE | /api/contract-types/:type_id | 删除类型 |

#### 合同管理 (/api/contracts)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts | 合同列表 |
| GET | /api/contracts/:contract_id | 合同详情 |
| POST | /api/contracts | 创建合同 |
| PUT | /api/contracts/:contract_id | 更新合同 |
| PUT | /api/contracts/:contract_id/status | 直接更新状态 |
| POST | /api/contracts/:contract_id/status-change | 申请状态变更 |
| GET | /api/contracts/:contract_id/status-change | 状态变更记录 |
| POST | /api/contracts/:contract_id/archive | 归档合同 |
| DELETE | /api/contracts/:contract_id | 删除合同 |
| GET | /api/contracts/:contract_id/lifecycle | 生命周期记录 |

#### 合同执行 (/api/executions)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/executions | 执行记录列表 |
| POST | /api/contracts/:contract_id/executions | 创建执行记录 |
| DELETE | /api/executions/:execution_id | 删除执行记录 |

#### 合同文档 (/api/documents)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/documents | 文档列表 |
| POST | /api/contracts/:contract_id/documents | 上传文档 |
| GET | /api/documents/:document_id/preview | 预览文档 |
| DELETE | /api/documents/:document_id | 删除文档 |

#### 审批管理 (/api/approvals)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/approvals | 审批记录 |
| POST | /api/contracts/:contract_id/approvals | 创建审批 |
| PUT | /api/approvals/:approval_id | 更新审批 |
| POST | /api/approvals/:approval_id/rollback | 回滚审批 |
| GET | /api/approvals/:approval_id/status | 审批状态 |
| POST | /api/approvals/process-expired | 处理过期审批 |
| GET | /api/pending-approvals | 待审批列表 |

#### 工作流 (/api/workflow)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/workflow/create | 创建工作流 |
| GET | /api/workflow/pending | 我的待审批 |
| GET | /api/workflow/:contract_id/status | 工作流状态 |
| GET | /api/workflow/:contract_id | 工作流详情 |
| POST | /api/workflow/:contract_id/remind | 催办 |
| POST | /api/workflow/approve | 审批通过 |
| POST | /api/workflow/reject | 审批拒绝 |

#### 状态变更审批

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/pending-status-changes | 待审批状态变更 |
| POST | /api/status-change-requests/:request_id/approve | 通过状态变更 |
| POST | /api/status-change-requests/:request_id/reject | 拒绝状态变更 |

#### 通知管理 (/api/notifications)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/notifications | 通知列表 |
| PUT | /api/notifications/:id/read | 标记已读 |
| DELETE | /api/notifications/:id | 删除通知 |
| DELETE | /api/notifications/all | 清空通知 |
| GET | /api/notifications/unread-count | 未读数量 |

#### 提醒管理 (/api/reminders)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/reminders | 合同提醒 |
| POST | /api/contracts/:contract_id/reminders | 创建提醒 |
| POST | /api/reminders/:reminder_id/send | 发送提醒 |

#### 统计与提醒

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/expiring-contracts | 即将到期合同 |
| GET | /api/statistics | 统计数据 |
| GET | /api/notifications/count | 待办数量 |

#### 审计日志 (/api/audit-logs)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/audit-logs | 审计日志 |
| DELETE | /api/audit-logs/:id | 删除单条 |
| POST | /api/audit-logs/batch-delete | 批量删除 |
| GET | /api/audit-logs/export | 导出日志 |

#### 加密服务 (/api/crypto)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/crypto/config-hsm | 配置HSM（管理员） |
| POST | /api/crypto/config-sm4 | 配置SM4（管理员） |
| POST | /api/crypto/config-aes | 配置AES（管理员） |
| POST | /api/crypto/generate-key | 生成密钥（管理员） |
| GET | /api/crypto/status | 加密服务状态 |
| POST | /api/crypto/encrypt | 加密数据 |
| POST | /api/crypto/decrypt | 解密数据 |

## 环境变量

| 变量名 | 说明 | 必填 |
|--------|------|------|
| MYSQL_HOST | 数据库地址 | 是 |
| MYSQL_PORT | 数据库端口 | 是 |
| MYSQL_USER | 数据库用户名 | 是 |
| MYSQL_PASSWORD | 数据库密码 | 是 |
| MYSQL_DATABASE | 数据库名称 | 是 |
| SECRET_KEY | JWT 签名密钥（至少32位） | 是 |
| ADMIN_USERNAME | 超级管理员用户名 | 否 |
| ADMIN_PASSWORD | 超级管理员密码 | 否 |
| AUDIT_ADMIN_USERNAME | 审计管理员用户名 | 否 |
| AUDIT_ADMIN_PASSWORD | 审计管理员密码 | 否 |
| HSM_ENABLED | 启用 HSM 密码机 | 否 |
| SM4_ENABLED | 启用 SM4 加密 | 否 |
| AES_ENABLED | 启用 AES 加密 | 否 |

生成随机 SECRET_KEY：
```bash
openssl rand -base64 32
```

## 加密服务

系统预留与服务器密码机（HSM）对接的加密服务接口。

### 支持的加密算法

- **HSM**：对接硬件安全模块/服务器密码机
- **SM4**：国密对称加密算法
- **AES**：高级加密标准

### API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/crypto/encrypt | POST | 加密数据 |
| /api/crypto/decrypt | POST | 解密数据 |
| /api/crypto/status | GET | 查看加密服务状态 |

## 常见问题

### 1. 数据库连接失败

检查：
- MySQL 服务是否启动
- `.env` 中的数据库配置是否正确
- 数据库用户是否有权限访问数据库

### 2. 前端无法访问后端

检查：
- 后端服务是否正常运行（http://localhost:8000/health）
- 防火墙是否允许对应端口

### 3. Token 过期

Token 默认 30 分钟过期，过期后需要重新登录。

### 4. 数据备份

```bash
# Docker 环境备份
docker exec contract_mysql mysqldump -u contract_user -p contract_manage > backup.sql

# 使用一键部署的备份脚本
./backup.sh
```

## 相关文档

- [部署说明](./docker/部署说明.md) - Docker 一键部署详细说明
- [开发文档](./docs/) - 完整的技术和运维文档

## 许可证

MIT License