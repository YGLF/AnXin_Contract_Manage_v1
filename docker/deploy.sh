#!/bin/bash
set -e

#===============================================================================
# CentOS 7 一键部署脚本 - 安信合同管理系统
# 使用方式: chmod +x deploy.sh && ./deploy.sh
#===============================================================================

PROJECT_NAME="contract-manage"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="${PROJECT_ROOT}/backups"
KEEP_DAYS=7
MYSQL_ROOT_PASSWORD="root123456"
MYSQL_USER="contract_user"
MYSQL_DATABASE="contract_manage"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

#===============================================================================
# 1. 系统检测
#===============================================================================
check_system() {
    log_step "1/9 检测系统环境..."
    if [[ "$EUID" -ne 0 ]]; then
        log_error "请使用root用户运行此脚本"
        exit 1
    fi

    if [[ ! -f /etc/centos-release ]]; then
        log_error "仅支持CentOS 7系统"
        exit 1
    fi

    if ! grep -q "CentOS.*7" /etc/centos-release; then
        log_error "仅支持CentOS 7系统，当前检测到: $(cat /etc/centos-release)"
        exit 1
    fi

    log_info "系统检测通过"
}

#===============================================================================
# 2. 安装Docker
#===============================================================================
install_docker() {
    log_step "2/9 安装Docker..."

    if command -v docker &> /dev/null; then
        log_info "Docker已安装: $(docker --version)"
        return 0
    fi

    log_info "开始安装Docker CE..."

    # 关闭防火墙（Docker容器网络需要）
    systemctl stop firewalld 2>/dev/null || true
    systemctl disable firewalld 2>/dev/null || true

    # 关闭SELinux（避免容器权限问题）
    setenforce 0 2>/dev/null || true
    if grep -q "SELINUX=enforcing" /etc/selinux/config 2>/dev/null; then
        sed -i 's/SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config
        log_warn "已禁用SELinux，需要重启生效（本次部署继续）"
    fi

    # 安装必要组件
    yum install -y yum-utils device-mapper-persistent-data lvm2 wget curl git

    # 添加Docker官方仓库（使用国内镜像）
    log_info "配置Docker仓库..."
    cat > /etc/yum.repos.d/docker-ce.repo <<'EOF'
[docker-ce-stable]
name=Docker CE Stable
baseurl=https://mirrors.aliyun.com/docker-ce/linux/centos/7/x86_64/stable
enabled=1
gpgcheck=0
EOF

    # 安装Docker（指定稳定版本）
    yum install -y docker-ce-24.0.7 docker-ce-cli-24.0.7 containerd.io docker-buildx-plugin docker-compose-plugin

    # 配置Docker镜像加速器（使用阿里云）
    log_info "配置Docker镜像加速器..."
    mkdir -p /etc/docker
    cat > /etc/docker/daemon.json <<'EOF'
{
    "registry-mirrors": [
        "https://registry.docker-cn.com",
        "https://mirror.ccs.tencentyun.com",
        "https://docker.mirrors.ustc.edu.cn"
    ],
    "log-driver": "json-file",
    "log-opts": {
        "max-size": "100m",
        "max-file": "3"
    },
    "storage-driver": "overlay2"
}
EOF
    # 重启Docker使配置生效
    systemctl restart docker
    log_info "Docker配置完成"

    # 启动Docker
    systemctl start docker
    systemctl enable docker

    log_info "Docker安装完成: $(docker --version)"
}

#===============================================================================
# 3. 安装Docker Compose
#===============================================================================
install_docker_compose() {
    log_step "3/9 安装Docker Compose..."

    # 检查新版docker compose
    if docker compose version &> /dev/null; then
        log_info "Docker Compose已安装: $(docker compose version)"
        return 0
    fi

    # 检查旧版docker compose
    if command -v docker compose &> /dev/null; then
        log_info "docker compose已安装: $(docker compose --version)"
        return 0
    fi

    log_info "安装Docker Compose..."

    # 下载二进制文件（使用国内镜像）
    curl -SL "https://github.com/docker/compose/releases/download/v2.24.5/docker-compose-linux-x86_64" -o /usr/local/bin/docker-compose

    chmod +x /usr/local/bin/docker compose
    ln -sf /usr/local/bin/docker compose /usr/bin/docker compose

    log_info "Docker Compose安装完成: $(docker compose --version)"
}

#===============================================================================
# 4. 检查并配置环境变量
#===============================================================================
setup_env() {
    log_step "4/9 配置环境变量..."

    # 生成随机密码
    generate_password() {
        openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16
    }

    # 生成随机SECRET_KEY
    SECRET_KEY=$(openssl rand -base64 32)

    # 生成数据库密码
    DB_PASSWORD=$(generate_password)

    # 生成管理员密码
    ADMIN_PASSWORD=$(generate_password)
    AUDIT_PASSWORD=$(generate_password)

    # 切换到项目根目录操作
    cd "$PROJECT_ROOT"

    # 创建生产环境配置文件
    cat > .env.production <<EOF
#===============================================================================
# 安信合同管理系统 - 生产环境配置
# 请根据实际情况修改以下配置
#===============================================================================

# 应用配置
APP_NAME=安信合同管理系统
APP_VERSION=1.0.0

# MySQL 数据库配置
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_USER=${MYSQL_USER}
MYSQL_PASSWORD=${DB_PASSWORD}
MYSQL_DATABASE=${MYSQL_DATABASE}

# JWT安全配置（必须使用随机密钥）
SECRET_KEY=${SECRET_KEY}
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=60

# 文件上传配置
UPLOAD_DIR=uploads
TZ=Asia/Shanghai

# 管理员账号配置（生产环境请修改密码）
ADMIN_USERNAME=admin
ADMIN_PASSWORD=${ADMIN_PASSWORD}
ADMIN_EMAIL=admin@anxin.com

# 审计管理员账号配置
AUDIT_ADMIN_USERNAME=auditadmin
AUDIT_ADMIN_PASSWORD=${AUDIT_PASSWORD}
AUDIT_ADMIN_EMAIL=audit@anxin.com

# 加密服务配置（如不需要请保持false）
# HSM密码机配置
# HSM_ENABLED=false
# HSM_ENDPOINT=http://hsm-server:8888
# HSM_APP_ID=your-app-id

# SM4对称加密配置
# SM4_ENABLED=false
# SM4_KEY=1234567890123456

# AES对称加密配置
# AES_ENABLED=false
# AES_KEY=12345678901234567890123456789012
EOF

    # 更新docker-compose.yml中的数据库密码
    if grep -q "MYSQL_PASSWORD: contract123" docker-compose.yml; then
        sed -i "s/MYSQL_PASSWORD: contract123/MYSQL_PASSWORD: ${DB_PASSWORD}/g" docker-compose.yml
    fi

    if grep -q "MYSQL_PASSWORD= contract123" docker-compose.yml; then
        sed -i "s/MYSQL_PASSWORD= contract123/MYSQL_PASSWORD=${DB_PASSWORD}/g" docker-compose.yml
    fi

    log_info "已创建生产环境配置文件: ${PROJECT_ROOT}/.env.production"
    log_warn "请及时修改 .env.production 中的密码配置"
}

#===============================================================================
# 5. 检查端口占用
#===============================================================================
check_ports() {
    log_step "5/9 检查端口占用情况..."

    for port in 3306 8000 80; do
        if netstat -tuln 2>/dev/null | grep -q ":$port "; then
            log_warn "端口 $port 已被占用，尝试终止占用进程..."
            fuser -k $port/tcp 2>/dev/null || true
            sleep 1
        fi
    done
}

#===============================================================================
# 6. 构建和启动服务
#===============================================================================
deploy_services() {
    log_step "6/9 构建并启动服务..."

    # 切换到项目根目录
    cd "$PROJECT_ROOT"

    # 创建必要目录
    mkdir -p uploads backups

    # 停止旧容器
    log_info "停止旧容器（如有）..."
    docker compose down 2>/dev/null || true

    # 清理旧的MySQL数据（全新部署时）
    if [[ ! -d "./mysql_data" ]] && [[ -z "$(docker volume ls -q contract-manage_mysql_data 2>/dev/null)" ]]; then
        log_warn "首次部署，清理旧数据卷..."
        docker volume rm contract-manage_mysql_data 2>/dev/null || true
    fi

    # 构建镜像（失败时重试）
    local max_retries=3
    local retry=0

    while [[ $retry -lt $max_retries ]]; do
        log_info "构建镜像... (尝试 $((retry + 1))/$max_retries)"
        if docker compose build --parallel 2>&1; then
            break
        fi
        retry=$((retry + 1))
        log_warn "构建失败，清理缓存后重试..."
        docker system prune -af --volumes 2>/dev/null || true
        sleep 5
    done

    if [[ $retry -eq $max_retries ]]; then
        log_error "构建失败，请检查Docker日志: docker compose logs"
        exit 1
    fi

    # 启动服务
    log_info "启动所有服务..."
    docker compose up -d

    # 等待MySQL就绪
    log_info "等待MySQL初始化..."
    local count=0
    local db_password=$(grep "^MYSQL_PASSWORD=" .env.production | cut -d'=' -f2)

    while ! docker exec contract_mysql mysqladmin ping -h localhost -u${MYSQL_USER} -p"${db_password}" &>/dev/null; do
        sleep 2
        count=$((count + 1))
        if [[ $count -gt 60 ]]; then
            log_error "MySQL启动超时，请检查: docker logs contract_mysql"
            exit 1
        fi
        echo -n "."
    done
    echo ""

    # 等待后端就绪
    log_info "等待后端服务就绪..."
    sleep 5
    count=0
    while ! curl -s http://localhost:8000/health &>/dev/null; do
        sleep 2
        count=$((count + 1))
        if [[ $count -gt 30 ]]; then
            log_warn "后端服务可能尚未就绪，请手动检查: docker logs contract_backend"
            break
        fi
        echo -n "."
    done
    echo ""

    log_info "所有服务启动完成"
}

#===============================================================================
# 7. 验证服务状态
#===============================================================================
verify_services() {
    log_step "7/9 验证服务状态..."

    # 切换到项目根目录
    cd "$PROJECT_ROOT"

    echo ""
    echo "容器运行状态:"
    docker compose ps
    echo ""

    # 检查端口
    log_info "检查服务端口..."

    sleep 3

    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/health | grep -q "200\|404"; then
        log_info "✓ 后端API (8000端口) 正常"
    else
        log_warn "✗ 后端API可能异常"
    fi

    if curl -s -o /dev/null -w "%{http_code}" http://localhost/ | grep -q "200\|302\|404"; then
        log_info "✓ 前端Web (80端口) 正常"
    else
        log_warn "✗ 前端Web可能异常"
    fi

    if docker exec contract_mysql mysqladmin ping -h localhost &>/dev/null; then
        log_info "✓ MySQL数据库 (3306端口) 正常"
    else
        log_warn "✗ MySQL数据库可能异常"
    fi
}

#===============================================================================
# 8. 配置定时备份
#===============================================================================
setup_backup() {
    log_step "8/9 配置定时备份任务..."

    # 切换到项目根目录
    cd "$PROJECT_ROOT"

    # 创建备份脚本
    cat > ./backup.sh <<'BACKUP_EOF'
#!/bin/bash
#===============================================================================
# 数据库自动备份脚本
#===============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backups"
KEEP_DAYS=7

# 加载环境变量
if [[ -f .env.production ]]; then
    export $(grep -v '^#' .env.production | xargs)
fi

DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p $BACKUP_DIR

# 获取数据库密码
DB_PASSWORD=${MYSQL_PASSWORD:-contract123}
DB_USER=${MYSQL_USER:-contract_user}
DB_NAME=${MYSQL_DATABASE:-contract_manage}

# 备份MySQL
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 开始备份数据库..."
docker exec contract_mysql mysqldump -u${DB_USER} -p"${DB_PASSWORD}" ${DB_NAME} 2>/dev/null | gzip > "$BACKUP_DIR/contract_manage_${DATE}.sql.gz"

if [[ $? -eq 0 ]]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 备份完成: contract_manage_${DATE}.sql.gz"
else
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 备份失败!"
    exit 1
fi

# 清理旧备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$KEEP_DAYS -delete 2>/dev/null
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 清理过期备份完成"
BACKUP_EOF

    chmod +x ./backup.sh

    # 添加crontab任务
    (crontab -l 2>/dev/null | grep -v "backup.sh"; echo "0 3 * * * cd ${PROJECT_ROOT} && ./backup.sh >> ./backups/backup.log 2>&1") | crontab -

    log_info "定时备份已配置（每天凌晨3点自动执行）"
    log_info "备份文件保存目录: ${PROJECT_ROOT}/backups/"
}

#===============================================================================
# 9. 打印访问信息和账号
#===============================================================================
print_info() {
    log_step "9/9 部署完成！"

    # 读取配置
    cd "$PROJECT_ROOT"
    source .env.production 2>/dev/null

    echo ""
    echo "========================================================================"
    echo -e "${GREEN}  安信合同管理系统 - 部署完成${NC}"
    echo "========================================================================"
    echo ""
    echo "  【服务访问地址】"
    echo "    前端Web:    http://服务器IP/"
    echo "    后端API:    http://服务器IP:8000"
    echo "    数据库:     localhost:3306"
    echo ""
    echo "  【管理员账号】"
    echo "    超级管理员: ${ADMIN_USERNAME} / ${ADMIN_PASSWORD}"
    echo "    审计管理员: ${AUDIT_ADMIN_USERNAME} / ${AUDIT_ADMIN_PASSWORD}"
    echo ""
echo " 【备份信息】"
    echo " 备份目录: ${PROJECT_ROOT}/backups/"
    echo "    备份时间:   每天凌晨3:00"
    echo "    保留天数:   ${KEEP_DAYS}天"
    echo ""
    echo "========================================================================"
    echo "  【常用命令】"
    echo "    查看日志:   docker compose logs -f"
    echo "    重启服务:   docker compose restart"
    echo "    停止服务:   docker compose down"
    echo "    启动服务:   docker compose up -d"
    echo "    手动备份:   ./backup.sh"
    echo "========================================================================"
    echo ""
    log_warn "请及时保存上述账号信息到安全位置！"
    log_warn "配置文件: .env.production"
    echo ""
}

#===============================================================================
# 主流程
#===============================================================================
main() {
    clear
    echo "========================================================================"
    echo "  安信合同管理系统 - CentOS 7 一键部署脚本"
    echo "========================================================================"
    echo ""

    check_system
    install_docker
    install_docker_compose
    check_ports
    setup_env
    deploy_services
    verify_services
    setup_backup
    print_info
}

main "$@"