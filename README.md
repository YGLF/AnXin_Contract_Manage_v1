# 合同管理系统

基于 Go + Gin + MySQL（后端）和 Vue3 + Element Plus（前端）的合同管理系统。

## 功能模块

- 用户权限管理：用户注册、登录、角色管理
- 客户/供应商管理：客户信息增删改查
- 合同管理：合同信息管理、分类管理
- 合同执行跟踪：进度跟踪、付款记录
- 审批流程：合同审批、多级审批
- 到期提醒：合同到期提醒、续期管理
- 统计报表：数据统计分析
- 文档管理：合同文件上传、版本管理

## 技术栈

### 后端
- Go 1.21+
- Gin Web Framework
- GORM
- MySQL
- JWT 认证

### 前端
- Vue 3
- Vite
- Element Plus
- Pinia
- Vue Router
- Axios
- ECharts

## 安装

### 后端安装

1. 进入项目目录
```bash
cd AnXin_Contract_Manage
```

2. 安装依赖
```bash
go mod download
```

3. 配置数据库

编辑 `.env` 文件：
```env
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=contract_manage
```

4. 创建数据库
```bash
mysql -u root -p
CREATE DATABASE contract_manage CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

5. 运行后端服务
```bash
go run main.go
```

后端 API 文档地址: http://localhost:8000/docs

### 前端安装

1. 进入前端目录
```bash
cd frontend
```

2. 安装依赖
```bash
npm install
```

3. 运行前端服务
```bash
npm run dev
```

前端访问地址: http://localhost:3000

4. 构建生产版本
```bash
npm run build
```

## API 端点

### 认证
- POST /api/auth/register - 用户注册
- POST /api/auth/login - 用户登录
- GET /api/auth/users - 获取用户列表
- GET /api/auth/users/{user_id} - 获取用户详情
- PUT /api/auth/users/{user_id} - 更新用户
- DELETE /api/auth/users/{user_id} - 删除用户

### 客户管理
- GET /api/customers - 获取客户列表
- GET /api/customers/{customer_id} - 获取客户详情
- POST /api/customers - 创建客户
- PUT /api/customers/{customer_id} - 更新客户
- DELETE /api/customers/{customer_id} - 删除客户
- GET /api/contract-types - 获取合同类型列表
- POST /api/contract-types - 创建合同类型

### 合同管理
- GET /api/contracts - 获取合同列表
- GET /api/contracts/{contract_id} - 获取合同详情
- POST /api/contracts - 创建合同
- PUT /api/contracts/{contract_id} - 更新合同
- DELETE /api/contracts/{contract_id} - 删除合同
- GET /api/contracts/{contract_id}/executions - 获取合同执行记录
- POST /api/contracts/{contract_id}/executions - 创建执行记录
- GET /api/contracts/{contract_id}/documents - 获取合同文档
- POST /api/contracts/{contract_id}/documents - 上传文档
- DELETE /api/documents/{document_id} - 删除文档

### 审批与提醒
- GET /api/contracts/{contract_id}/approvals - 获取审批记录
- POST /api/contracts/{contract_id}/approvals - 创建审批记录
- PUT /api/approvals/{approval_id} - 更新审批状态
- GET /api/contracts/{contract_id}/reminders - 获取提醒列表
- POST /api/contracts/{contract_id}/reminders - 创建提醒
- POST /api/reminders/{reminder_id}/send - 发送提醒
- GET /api/expiring-contracts - 获取即将到期的合同
- GET /api/statistics - 获取统计数据

## 项目结构

```
AnXin_Contract_Manage/
├── config/                    # 配置
│   └── config.go              # 配置加载
├── handlers/                  # HTTP 处理器
│   ├── auth.go                # 认证相关
│   ├── customer.go            # 客户管理
│   ├── contract.go            # 合同管理
│   └── approval.go            # 审批与提醒
├── middleware/                # 中间件
│   └── auth.go                # JWT 认证中间件
├── models/                    # 数据库模型
│   └── models.go
├── services/                  # 业务逻辑
│   ├── user_service.go
│   ├── customer_service.go
│   ├── contract_service.go
│   └── approval_service.go
├── frontend/                  # 前端目录
│   ├── src/
│   │   ├── api/               # API 接口
│   │   │   ├── auth.js
│   │   │   ├── customer.js
│   │   │   ├── contract.js
│   │   │   └── approval.js
│   │   ├── components/        # 公共组件
│   │   ├── router/            # 路由配置
│   │   │   └── index.js
│   │   ├── store/             # 状态管理
│   │   │   └── user.js
│   │   ├── utils/             # 工具函数
│   │   │   └── request.js
│   │   ├── views/             # 页面组件
│   │   │   ├── Login.vue
│   │   │   ├── Layout.vue
│   │   │   ├── Dashboard.vue
│   │   │   ├── Contract.vue
│   │   │   ├── Customer.vue
│   │   │   ├── User.vue
│   │   │   ├── Approval.vue
│   │   │   └── Reminder.vue
│   │   ├── App.vue
│   │   └── main.js
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── main.go                    # 后端入口
├── go.mod                     # Go 模块定义
├── .env                       # 环境变量配置
└── Dockerfile                 # Docker 构建文件
```

## 前端页面说明

- **登录页面** (Login.vue) - 用户登录
- **布局页面** (Layout.vue) - 主框架布局，包含侧边栏导航
- **仪表盘** (Dashboard.vue) - 数据统计、图表展示、即将到期合同
- **合同管理** (Contract.vue) - 合同的增删改查、状态管理
- **客户管理** (Customer.vue) - 客户/供应商信息管理
- **用户管理** (User.vue) - 用户信息管理、角色分配
- **审批管理** (Approval.vue) - 合同审批流程
- **到期提醒** (Reminder.vue) - 合同到期提醒管理

## 使用说明

1. 启动后端服务：`go run main.go`
2. 启动前端服务：`cd frontend && npm run dev`
3. 访问前端：http://localhost:3000
4. 使用注册功能创建管理员账户，或直接登录测试

## Docker 部署

### 使用 Docker Compose 快速部署（推荐）

这是最简单的方式，会自动启动 MySQL、后端和前端服务。

1. 确保已安装 Docker 和 Docker Compose
2. 在项目根目录执行：
```bash
docker-compose up -d
```

3. 访问系统：
- 前端：http://localhost
- 后端 API：http://localhost:8000

4. 停止服务：
```bash
docker-compose down
```

5. 查看日志：
```bash
docker-compose logs -f
```

### 手动构建和部署

#### 构建并运行后端

1. 构建后端镜像：
```bash
docker build -t contract-backend .
```

2. 运行后端容器：
```bash
docker run -d -p 8000:8000 \
  -e MYSQL_HOST=mysql \
  -e MYSQL_PORT=3306 \
  -e MYSQL_USER=contract_user \
  -e MYSQL_PASSWORD=contract123 \
  -e MYSQL_DATABASE=contract_manage \
  --name contract-backend \
  contract-backend
```

#### 构建并运行前端

1. 构建前端镜像：
```bash
cd frontend
docker build -t contract-frontend .
```

2. 运行前端容器：
```bash
docker run -d -p 80:80 --name contract-frontend contract-frontend
```

### 部署到服务器

1. 将项目代码上传到服务器
2. 修改 `docker-compose.yml` 中的端口映射（如需要）
3. 修改数据库密码等敏感配置
4. 运行 `docker-compose up -d`
5. 配置反向代理（如 Nginx）和 SSL 证书

### 数据持久化

Docker Compose 配置已包含 MySQL 数据卷持久化，数据存储在 Docker 卷 `mysql_data` 中。即使删除容器，数据也不会丢失。

### 备份数据

备份 MySQL 数据：
```bash
docker exec contract_mysql mysqldump -u contract_user -pcontract123 contract_manage > backup.sql
```

恢复 MySQL 数据：
```bash
cat backup.sql | docker exec -i contract_mysql mysql -u contract_user -pcontract123 contract_manage
```

## 开发说明

- 后端使用 Gin 框架，路由自动注册
- 前端使用 Vue 3 Composition API 开发
- 使用 Pinia 进行状态管理
- 使用 Element Plus 作为 UI 组件库
- 使用 ECharts 进行数据可视化