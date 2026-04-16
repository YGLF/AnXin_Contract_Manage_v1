# 安信合同管理系统

基于 Go + Gin + MySQL（后端）和 Vue3 + Element Plus（前端）的智能合同管理系统，支持用户级别细粒度权限控制、数据完整性验证和 SHA-256 密码杂凑。

## 目录

- [功能模块](#功能模块)
- [技术栈](#技术栈)
- [快速开始](#快速开始)
  - [手动部署](#手动部署)
  - [Docker 部署](#docker-部署)
- [权限系统](#权限系统)
- [API 认证](#api-认证)
- [API 端点](#api-端点)
- [项目结构](#项目结构)
- [前端页面说明](#前端页面说明)
- [环境变量](#环境变量)
- [Docker 操作](#docker-操作)
- [数据备份与恢复](#数据备份与恢复)
- [安全说明](#安全说明)
- [常见问题](#常见问题)

## 功能模块

- **用户权限管理**：用户注册、登录、角色管理（超级管理员、经理、销售、审计管理员）、用户级别细粒度权限控制
- **客户/供应商管理**：客户信息增删改查、客户分类、信用等级
- **合同管理**：合同信息管理、合同分类管理、合同状态跟踪
- **合同执行跟踪**：进度跟踪、付款记录、执行阶段管理
- **审批流程**：合同审批、三级审批（销售总监→技术总监→财务总监）、审批记录查询
- **状态变更审批**：关键状态变更（归档、终止、执行中、待付款）需管理员审批
- **合同生命周期**：完整的合同状态变更历史记录
- **合同归档**：已完成合同归档管理、到期自动归档、定时任务通知
- **到期提醒**：合同到期提醒、续期管理、提醒通知、过期强提醒
- **统计报表**：数据统计分析、图表展示
- **文档管理**：合同文件上传、版本管理
- **合同类型管理**：合同类型分类管理
- **待办提示**：侧边栏菜单红点提示待办事项
- **数据完整性验证**：SHA-256 密码杂凑验证、用户数据完整性哈希校验
- **登录超时**：30分钟无操作自动退出、提前2分钟提醒
- **加密服务接口**：预留与服务器密码机（HSM）对接接口、支持SM4/AES加密

## 技术栈

### 后端
- Go 1.21+
- Gin Web Framework（高性能 HTTP 框架）
- GORM（ORM 库）
- MySQL 8.0
- JWT（用户认证）
- bcrypt（密码加密）
- SHA-256（密码杂凑验证）

### 前端
- Vue 3（渐进式前端框架）
- Vite（构建工具）
- Element Plus（UI 组件库）
- Pinia（状态管理）
- Vue Router（路由管理）
- Axios（HTTP 客户端）
- ECharts（数据可视化）

## 快速开始

### 手动部署

手动部署适合开发环境或不想使用 Docker 的用户。

#### 前置要求

- Go 1.21 或更高版本
- MySQL 8.0 或更高版本
- Node.js 16+ 和 npm
- Git

#### 后端部署步骤

1. **克隆项目**

```bash
git clone <repository-url>
cd AnXin_Contract_Manage
```

2. **安装 Go 依赖**

```bash
go mod download
```

3. **配置环境变量**

复制环境变量示例文件并修改配置：

```bash
cp .env.example .env
```

编辑 `.env` 文件，修改数据库连接信息：

```env
APP_NAME=合同管理系统
APP_VERSION=1.0.0

MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=contract_manage

SECRET_KEY=your-secret-key-change-in-production
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=30

UPLOAD_DIR=uploads

ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin@123456
ADMIN_EMAIL=admin@example.com
```

4. **创建数据库**

```bash
mysql -u root -p -e "CREATE DATABASE contract_manage CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

如果使用 Docker 安装 MySQL：

```bash
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=root123456 \
  -e MYSQL_DATABASE=contract_manage \
  -p 3306:3306 \
  mysql:8.0
```

5. **运行后端服务**

```bash
go run main.go
```

首次运行会自动创建数据库表。后端服务默认运行在 http://localhost:8000

#### 前端部署步骤

1. **进入前端目录**

```bash
cd frontend
```

2. **安装依赖**

```bash
npm install
```

3. **配置 API 地址（如需要）**

编辑 `vite.config.js` 配置后端 API 地址：

```javascript
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true
      }
    }
  }
})
```

4. **运行开发服务器**

```bash
npm run dev
```

前端服务默认运行在 http://localhost:3000

5. **构建生产版本**

```bash
npm run build
```

构建完成后，静态文件会生成在 `dist` 目录，可以部署到任意 Web 服务器。

### Docker 部署

Docker 部署适合快速启动或生产环境。

#### 前置要求

- Docker 20.10+
- Docker Compose 2.0+

#### 部署步骤

1. **进入项目目录**

```bash
cd AnXin_Contract_Manage
```

2. **配置环境变量**

```bash
cp .env.example .env
```

3. **启动服务**

```bash
docker-compose up -d
```

首次启动会：
- 创建 MySQL 容器并初始化数据库
- 构建并启动后端容器
- 构建并启动前端容器

4. **访问系统**

- 前端：http://localhost
- 后端 API：http://localhost:8000
- MySQL：localhost:3306

#### 查看服务状态

```bash
docker-compose ps
```

#### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看后端日志
docker-compose logs -f backend

# 查看前端日志
docker-compose logs -f frontend

# 查看数据库日志
docker-compose logs -f mysql
```

## 权限系统

### 概述

系统采用 **角色权限 + 用户自定义权限** 的双重权限控制模式：
- **角色权限**：定义每个角色默认拥有的权限
- **用户自定义权限**：基于角色权限追加额外权限（追加模式）
- **最终权限** = 角色权限 ∪ 用户自定义权限

### 权限标识说明

权限标识使用点号分隔（便于扩展），权限名称使用中文。

### 系统权限清单

| 权限标识 | 权限名称 | 分组 |
|---------|---------|------|
| `dashboard` | 仪表盘 | 系统 |
| `user.manage` | 用户管理 | 系统 |
| `audit.view` | 查看审计 | 系统 |
| `contract.read` | 查看合同 | 合同 |
| `contract.create` | 创建合同 | 合同 |
| `contract.edit` | 编辑合同 | 合同 |
| `contract.delete` | 删除合同 | 合同 |
| `customer.read` | 查看客户 | 客户 |
| `customer.create` | 创建客户 | 客户 |
| `customer.edit` | 编辑客户 | 客户 |
| `customer.delete` | 删除客户 | 客户 |
| `approval.process` | 审批处理 | 审批 |
| `approval.view` | 查看审批 | 审批 |

### 角色默认权限

| 角色 | 角色标识 | 默认权限 |
|------|---------|---------|
| 超级管理员 | `admin` | `all`（所有权限） |
| 经理 | `manager` | 仪表盘、查看/创建/编辑合同、查看/创建/编辑客户、审批处理、查看审批 |
| 销售 | `user` | 仪表盘、查看/创建合同、查看/创建客户 |
| 审计管理员 | `audit_admin` | 仪表盘、查看审计、查看合同、查看客户、查看审批 |

### 用户自定义权限配置

在用户管理中，可为用户追加额外权限：

```
基础角色: 经理 (继承权限: 仪表盘、查看合同、创建合同、编辑合同、客户权限、审批权限)

用户自定义权限 (追加到角色权限):
┌─ 系统权限 ────────────────────────┐
│ ☑ audit.view    查看审计         │
└─────────────────────────────────┘
┌─ 合同权限 ────────────────────────┐
│ ☑ contract.delete  删除合同      │
└─────────────────────────────────┘

最终生效权限 = 角色权限 + 自定义权限
```

### 权限检查流程

1. 用户登录时，系统返回用户完整权限列表
2. 路由守卫根据路由配置的权限检查用户是否有权访问
3. 侧边栏菜单根据用户权限动态显示/隐藏菜单项
4. 后端 API 中间件可添加权限验证

## API 认证

除 `/api/auth/register` 和 `/api/auth/login` 外，所有 API 需要 JWT 认证。

### 认证流程

1. **注册用户**（可选，已有用户可跳过）

```bash
curl -X POST http://localhost:8000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "password123",
    "full_name": "管理员",
    "role": "admin"
  }'
```

2. **登录获取 Token**

```bash
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin@123456",
    "password_hash": "<前端SHA-256杂凑值>"
  }'
```

响应：

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "user_info": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "full_name": "管理员",
    "role": "admin",
    "permissions": ["dashboard", "contract.read", "..."]
  }
}
```

3. **在请求中携带 Token**

```bash
curl -X GET http://localhost:8000/api/auth/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Token 过期时间

默认 Token 有效期为 30 分钟，可在环境变量中修改 `ACCESS_TOKEN_EXPIRE_MINUTES`。

### 默认账号

程序首次启动时会自动创建超级管理员和审计管理员账号。

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | admin@123456 | 超级管理员 |
| auditadmin | auditadmin@123456 | 审计管理员 |

如需修改账号信息，可在 `.env` 中配置：

```env
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin@123456
ADMIN_EMAIL=admin@example.com

AUDIT_ADMIN_USERNAME=auditadmin
AUDIT_ADMIN_PASSWORD=auditadmin@123456
AUDIT_ADMIN_EMAIL=audit@example.com
```

> 注意：如果账号已存在，则不会重复创建。

## API 端点

### 公共端点（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| GET | / | 服务信息 |
| GET | /health | 健康检查 |

### 需要认证的端点

#### 用户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/auth/users | 获取用户列表 |
| GET | /api/auth/users/:user_id | 获取用户详情 |
| PUT | /api/auth/users/:user_id | 更新用户信息 |
| DELETE | /api/auth/users/:user_id | 删除用户 |

请求示例：

```bash
# 获取用户列表
curl -X GET http://localhost:8000/api/auth/users?skip=0&limit=100 \
  -H "Authorization: Bearer <token>"

# 创建用户（带自定义权限）
curl -X POST http://localhost:8000/api/auth/register \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "sales1",
    "password": "password123",
    "full_name": "销售员1",
    "role": "user",
    "custom_permissions": "[\"contract.delete\"]"
  }'

# 更新用户
curl -X PUT http://localhost:8000/api/auth/users/1 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "新名字",
    "email": "newemail@example.com",
    "custom_permissions": "[\"contract.delete\",\"audit.view\"]"
  }'
```

#### 客户管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/customers | 获取客户列表 |
| GET | /api/customers/:customer_id | 获取客户详情 |
| POST | /api/customers | 创建客户 |
| PUT | /api/customers/:customer_id | 更新客户 |
| DELETE | /api/customers/:customer_id | 删除客户 |
| GET | /api/contract-types | 获取合同类型列表 |
| POST | /api/contract-types | 创建合同类型 |
| PUT | /api/contract-types/:type_id | 更新合同类型 |
| DELETE | /api/contract-types/:type_id | 删除合同类型 |

#### 合同管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts | 获取合同列表 |
| GET | /api/contracts/:contract_id | 获取合同详情 |
| POST | /api/contracts | 创建合同 |
| PUT | /api/contracts/:contract_id | 更新合同 |
| PUT | /api/contracts/:contract_id/status | 直接更新合同状态 |
| POST | /api/contracts/:contract_id/status-change | 申请状态变更（需要审批的状态） |
| GET | /api/contracts/:contract_id/status-change | 获取状态变更申请记录 |
| DELETE | /api/contracts/:contract_id | 删除合同 |
| GET | /api/contracts/:contract_id/lifecycle | 获取合同生命周期记录 |
| GET | /api/contracts/:contract_id/executions | 获取执行记录 |
| POST | /api/contracts/:contract_id/executions | 创建执行记录 |
| DELETE | /api/executions/:execution_id | 删除执行记录 |
| GET | /api/contracts/:contract_id/documents | 获取文档列表 |
| POST | /api/contracts/:contract_id/documents | 上传文档 |
| DELETE | /api/documents/:document_id | 删除文档 |

**合同状态说明：**
- `draft` - 草稿
- `pending` - 待审批
- `approved` - 已批准
- `active` - 已生效
- `in_progress` - 执行中
- `pending_pay` - 待付款
- `completed` - 已完成
- `terminated` - 已终止
- `archived` - 已归档

**需要审批的状态变更：**
- `archived` (归档)
- `terminated` (终止)
- `in_progress` (执行中)
- `pending_pay` (待付款)

#### 审批管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/approvals | 获取审批记录 |
| POST | /api/contracts/:contract_id/approvals | 创建审批记录 |
| PUT | /api/approvals/:approval_id | 更新审批状态 |
| GET | /api/pending-approvals | 获取待审批列表 |
| GET | /api/pending-status-changes | 获取待审批状态变更列表 |
| POST | /api/status-change-requests/:request_id/approve | 审批通过状态变更 |
| POST | /api/status-change-requests/:request_id/reject | 拒绝状态变更 |
| GET | /api/notifications/count | 获取待办事项数量 |

#### 提醒管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/contracts/:contract_id/reminders | 获取提醒列表 |
| POST | /api/contracts/:contract_id/reminders | 创建提醒 |
| POST | /api/reminders/:reminder_id/send | 发送提醒 |
| GET | /api/expiring-contracts | 获取即将到期合同 |
| GET | /api/statistics | 获取统计数据 |

## 项目结构

```
AnXin_Contract_Manage/
├── config/                    # 配置模块
│   └── config.go              # 环境配置加载
├── handlers/                  # HTTP 处理器
│   ├── auth.go                # 认证相关（登录、注册、用户管理）
│   ├── customer.go            # 客户管理
│   ├── contract.go            # 合同管理（含生命周期、状态变更）
│   ├── approval.go            # 审批与提醒
│   ├── audit.go               # 审计日志
│   ├── workflow.go            # 工作流审批
│   └── crypto.go              # 加密服务（HSM/SM4/AES）
├── middleware/                # 中间件
│   ├── auth.go                # JWT 认证中间件、权限检查中间件
│   ├── security.go            # 安全中间件
│   └── validator.go           # 输入验证中间件
├── models/                    # 数据模型
│   ├── models.go             # GORM 模型定义（含 User、Contract、Customer 等）
│   ├── permissions.go        # 权限常量定义、角色权限映射
│   └── workflow.go            # 工作流模型
├── services/                  # 业务逻辑层
│   ├── user_service.go        # 用户服务（含自定义权限处理）
│   ├── customer_service.go    # 客户服务
│   ├── contract_service.go    # 合同服务（含生命周期、归档、状态变更）
│   ├── approval_service.go    # 审批服务
│   ├── audit_service.go       # 审计服务
│   └── workflow_service.go   # 工作流服务
├── routes/                    # 路由模块
│   └── routes.go              # API路由统一配置
├── crypto/                    # 加密服务模块
│   └── service.go             # 加密服务接口（SM4/AES/HSM）
├── migrations/                # 数据库迁移脚本
├── scripts/                   # 测试脚本
├── docs/                      # API文档
├── frontend/                  # 前端项目
│   ├── src/
│   │   ├── api/              # API 接口定义
│   │   ├── components/        # 公共组件
│   │   ├── router/           # 路由配置（含权限守卫）
│   │   ├── store/            # 状态管理（含权限状态）
│   │   ├── utils/            # 工具函数
│   │   └── views/            # 页面组件
│   │       ├── Login.vue     # 登录页面（SHA-256 杂凑）
│   │       ├── Register.vue  # 注册页面
│   │       ├── Layout.vue    # 布局组件（含动态菜单）
│   │       ├── Dashboard.vue # 仪表盘
│   │       ├── Contract.vue  # 合同管理
│   │       ├── ContractDetail.vue  # 合同详情
│   │       ├── Customer.vue  # 客户管理
│   │       ├── User.vue      # 用户管理（含权限配置）
│   │       ├── Approval.vue  # 审批管理
│   │       ├── Reminder.vue # 到期提醒
│   │       └── Audit.vue    # 审计日志
│   ├── package.json
│   └── vite.config.js
├── main.go                    # 后端入口文件
├── go.mod                     # Go 模块定义
├── go.sum                     # Go 依赖校验
├── .env.example               # 环境变量示例
├── .env                       # 环境变量（需手动创建）
├── init.sql                   # 数据库初始化脚本
├── docker-compose.yml         # Docker Compose 配置
├── Dockerfile                 # 后端 Docker 构建文件
└── README.md                  # 项目文档
```

## 前端页面说明

| 页面 | 文件 | 说明 |
|------|------|------|
| 登录 | Login.vue | 用户登录，使用 SHA-256 杂凑密码 |
| 注册 | Register.vue | 用户注册 |
| 布局 | Layout.vue | 主框架布局，根据权限动态显示菜单 |
| 仪表盘 | Dashboard.vue | 数据统计、图表展示、即将到期合同 |
| 合同管理 | Contract.vue | 合同增删改查、状态管理 |
| 合同详情 | ContractDetail.vue | 合同详细信息，包含执行跟踪、文档管理、审批记录、生命周期时间线 |
| 客户管理 | Customer.vue | 客户/供应商信息管理，包含合同类型管理 |
| 用户管理 | User.vue | 用户信息管理、角色分配、用户级别权限配置 |
| 审批管理 | Approval.vue | 合同审批流程、审批历史、状态变更审批 |
| 到期提醒 | Reminder.vue | 合同到期提醒管理 |
| 审计日志 | Audit.vue | 系统操作审计日志（审计管理员可见） |

## 环境变量

| 变量名 | 说明 | 默认值 | 必填 |
|--------|------|--------|------|
| APP_NAME | 应用名称 | 合同管理系统 | 否 |
| APP_VERSION | 版本号 | 1.0.0 | 否 |
| MYSQL_HOST | MySQL 主机地址 | localhost | 是 |
| MYSQL_PORT | MySQL 端口 | 3306 | 是 |
| MYSQL_USER | MySQL 用户名 | root | 是 |
| MYSQL_PASSWORD | MySQL 密码 | - | 是 |
| MYSQL_DATABASE | 数据库名称 | contract_manage | 是 |
| SECRET_KEY | JWT 签名密钥 | - | 是 |
| JWT_ALGORITHM | JWT 算法 | HS256 | 否 |
| ACCESS_TOKEN_EXPIRE_MINUTES | Token 过期时间(分钟) | 30 | 否 |
| UPLOAD_DIR | 文件上传目录 | uploads | 否 |
| ADMIN_USERNAME | 超级管理员用户名 | admin | 否 |
| ADMIN_PASSWORD | 超级管理员密码 | admin@123456 | 否 |
| ADMIN_EMAIL | 超级管理员邮箱 | admin@example.com | 否 |
| AUDIT_ADMIN_USERNAME | 审计管理员用户名 | auditadmin | 否 |
| AUDIT_ADMIN_PASSWORD | 审计管理员密码 | auditadmin@123456 | 否 |
| AUDIT_ADMIN_EMAIL | 审计管理员邮箱 | audit@example.com | 否 |
| HSM_ENABLED | 是否启用HSM密码机 | false | 否 |
| HSM_ENDPOINT | HSM密码机服务地址 | - | 否 |
| HSM_APP_ID | HSM密码机应用ID | - | 否 |
| SM4_ENABLED | 是否启用SM4加密 | false | 否 |
| SM4_KEY | SM4对称密钥(16位) | - | 否 |
| AES_ENABLED | 是否启用AES加密 | false | 否 |
| AES_KEY | AES对称密钥(32位) | - | 否 |

### SECRET_KEY 安全建议

- 使用至少 32 位的随机字符串
- 不要使用默认值
- 生产环境定期更换
- 可以使用以下命令生成：
```bash
openssl rand -base64 32
```

## Docker 操作

### 常用命令

```bash
# 启动所有服务
docker-compose up -d

# 停止所有服务
docker-compose down

# 重启服务
docker-compose restart

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 重建并启动
docker-compose up -d --build

# 进入后端容器
docker exec -it contract_backend sh

# 进入前端容器
docker exec -it contract_frontend sh

# 进入 MySQL 容器
docker exec -it contract_mysql mysql -u contract_user -p contract_manage
```

### 修改配置后重启

修改 `docker-compose.yml` 或环境变量后，需要重建容器：

```bash
docker-compose down
docker-compose up -d --build
```

### 端口说明

| 服务 | 端口 | 说明 |
|------|------|------|
| 前端 | 80 | HTTP 服务 |
| 后端 | 8000 | API 服务 |
| MySQL | 3306 | 数据库服务 |

## 数据备份与恢复

### 备份数据

```bash
# 备份整个数据库
docker exec contract_mysql mysqldump -u contract_user -pcontract123 contract_manage > backup_$(date +%Y%m%d).sql
```

### 恢复数据

```bash
# 从备份文件恢复
cat backup_20240101.sql | docker exec -i contract_mysql mysql -u contract_user -pcontract123 contract_manage
```

### 数据持久化

Docker Compose 配置了 MySQL 数据卷 `mysql_data`，数据会持久化存储。即使删除容器，重新启动后数据依然存在。

如需完全清除数据：

```bash
docker-compose down -v
```

## 安全说明

- **密码加密**：用户密码使用 bcrypt 加密存储，不可逆
- **密码杂凑验证**：登录时前端使用 SHA-256 对密码进行杂凑，后端比对杂凑值并记录验证状态
- **数据完整性验证**：用户鉴别信息使用 SHA-256 生成完整性哈希，用于检测数据是否被篡改
- **认证机制**：除登录注册外，所有 API 需要有效的 JWT Token
- **Token 时效**：Token 默认 30 分钟过期，需要重新登录
- **权限控制**：用户级别细粒度权限控制，路由和 API 双重权限验证
- **登录超时**：30分钟无操作自动退出，提前2分钟弹窗提醒
- **加密服务**：预留与服务器密码机（HSM）对接接口，支持 SM4/AES 加密
- **生产环境建议**：
  - 修改默认的 SECRET_KEY
  - 使用 HTTPS 部署
  - 配置防火墙规则
  - 定期备份数据库
  - 生产环境启用加密服务

## 加密服务

系统预留了与服务器密码机（HSM）对接的加密服务接口，支持以下加密方式：

### 支持的加密算法
- **HSM**：对接硬件安全模块/服务器密码机
- **SM4**：国密对称加密算法
- **AES**：高级加密标准

### API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/crypto/encrypt` | POST | 加密数据 |
| `/api/crypto/decrypt` | POST | 解密数据 |
| `/api/crypto/config-hsm` | POST | 配置HSM密码机（仅管理员） |
| `/api/crypto/config-sm4` | POST | 配置SM4（仅管理员） |
| `/api/crypto/config-aes` | POST | 配置AES（仅管理员） |
| `/api/crypto/generate-key` | POST | 生成密钥（仅管理员） |
| `/api/crypto/status` | GET | 查看加密服务状态 |

### 使用示例

```bash
# 配置SM4加密
curl -X POST http://localhost:8000/api/crypto/config-sm4 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"key": "1234567890123456"}'

# 加密数据
curl -X POST http://localhost:8000/api/crypto/encrypt \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"data": "敏感数据", "algo_type": "sm4"}'

# 解密数据
curl -X POST http://localhost:8000/api/crypto/decrypt \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"data": "加密后的数据", "algo_type": "sm4"}'
```

## 常见问题

### 1. 启动失败，提示数据库连接失败

检查：
- MySQL 服务是否启动
- `.env` 中的数据库配置是否正确
- 数据库用户是否有权限访问数据库

### WSL 连接 Windows MySQL

如果后端运行在 WSL 中，数据库在 Windows 上，配置如下：

1. 获取 Windows IP（在 WSL 中执行）：
   ```bash
   cat /etc/resolv.conf
   ```

2. 在 `.env` 中配置：
   ```env
   MYSQL_HOST=172.x.x.x  # 从上面获取的 IP
   ```

3. 确保 Windows 防火墙允许 MySQL 端口（3306）

### 2. 前端无法访问后端 API

检查：
- 后端服务是否正常运行（http://localhost:8000/health）
- 前端 `vite.config.js` 中的代理配置是否正确
- 防火墙是否允许对应端口

### 3. Token 过期后怎么办

Token 过期后，前端会自动跳转到登录页面，需要重新登录。

### 4. 如何修改用户权限

在用户管理页面编辑用户，可配置用户自定义权限（追加到角色权限）。

### 5. 上传文件大小限制

默认限制 10MB，如需修改请编辑 `docker-compose.yml` 或 Nginx 配置。

### 6. 如何查看后端 API 文档

后端未集成 Swagger，可参考本文档的 API 端点说明。

## 文档手册

系统提供以下手册文档：

### 运维手册类

| 手册 | 文件 | 说明 |
|------|------|------|
| 用户操作手册 | `docs/用户操作手册.md` | 系统功能使用指南 |
| 运维管理手册 | `docs/运维管理手册.md` | 系统运维管理指南 |
| 运维操作手册 | `docs/运维操作手册.md` | 常见运维操作步骤 |
| 系统部署手册 | `docs/系统部署手册.md` | 详细部署指南 |
| 系统调试手册 | `docs/系统调试手册.md` | 问题排查与调试指南 |

### 技术文档类

| 手册 | 文件 | 说明 |
|------|------|------|
| 架构设计文档 | `docs/架构设计文档.md` | 系统架构设计 |
| 数据库设计文档 | `docs/数据库设计文档.md` | 数据库表结构设计 |
| 安全配置指南 | `docs/安全配置指南.md` | 系统安全配置 |
| 合规说明文档 | `docs/合规说明文档.md` | 合规性说明 |

### 运维流程类

| 手册 | 文件 | 说明 |
|------|------|------|
| 监控告警配置 | `docs/监控告警配置.md` | 监控和告警配置 |
| 备份恢复方案 | `docs/备份恢复方案.md` | 数据备份恢复 |
| 版本升级指南 | `docs/版本升级指南.md` | 系统升级步骤 |

### 测试文档类

| 手册 | 文件 | 说明 |
|------|------|------|
| 测试策略文档 | `docs/测试策略文档.md` | 测试策略和方法 |
| 测试结果报告 | `docs/测试结果报告.md` | 测试结果记录 |

### API文档

| 手册 | 文件 | 说明 |
|------|------|------|
| API文档 | `docs/swagger.json` | REST API接口文档 |
| API文档 | `docs/swagger.html` | 在线API文档 |

请访问 docs 目录查看完整手册内容。

## 许可证

MIT License
