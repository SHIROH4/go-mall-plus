#!/bin/bash
# =============================================================================
# 单机部署脚本 - 按需启动基础设施和单个服务
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

NETWORK_NAME="demo_infra_network"

# 创建共享网络
create_network() {
    if ! docker network inspect $NETWORK_NAME &> /dev/null; then
        log_info "创建 Docker 网络: $NETWORK_NAME"
        docker network create $NETWORK_NAME
    else
        log_info "网络 $NETWORK_NAME 已存在"
    fi
}

# 预构建所有依赖 (只运行一次)
build_deps() {
    log_info "预构建 Go 依赖 (加速后续构建)..."
    cd "$SCRIPT_DIR/../.."
    
    # 构建一个包含所有依赖的基础镜像
    docker build \
        --target builder \
        --tag ecommerce-deps:latest \
        --file app/gateway/Dockerfile .
    
    log_info "依赖预构建完成!"
}

# 启动基础设施
start_infra() {
    create_network
    log_info "启动基础设施..."
    cd "$SCRIPT_DIR"
    docker compose -f docker-compose.infra.yml up -d
    
    log_info "等待基础设施就绪..."
    for i in {1..30}; do
        if docker exec demo-mysql mysqladmin ping -uroot -proot123456 &> /dev/null; then
            log_info "MySQL 已就绪"
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    log_info "基础设施启动完成!"
}

# 停止基础设施
stop_infra() {
    log_info "停止基础设施..."
    cd "$SCRIPT_DIR"
    docker compose -f docker-compose.infra.yml down
}

# 启动单个服务 (优先使用缓存镜像)
start_service() {
    local service=$1
    local use_cache=${2:-false}
    
    create_network
    
    # 检查镜像是否已存在
    local image_name="ecommerce-$service:latest"
    if docker image inspect $image_name &> /dev/null; then
        log_info "镜像 $image_name 已存在，直接启动..."
        cd "$SCRIPT_DIR"
        docker compose -f "docker-compose.$service.yml" up -d "$service"
    else
        log_info "构建 $service..."
        cd "$SCRIPT_DIR"
        
        # 尝试使用 BuildKit 缓存
        export DOCKER_BUILDKIT=1
        
        docker compose -f "docker-compose.$service.yml" build --build-arg BUILDKIT_INLINE_CACHE=1 "$service"
        docker compose -f "docker-compose.$service.yml" up -d "$service"
    fi
    
    log_info "$service 启动完成!"
}

# 停止单个服务
stop_service() {
    local service=$1
    log_info "停止 $service..."
    cd "$SCRIPT_DIR"
    docker compose -f "docker-compose.$service.yml" down
}

# 查看服务状态
status() {
    echo ""
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  服务状态${NC}"
    echo -e "${BLUE}===============================================${NC}"
    echo ""
    
    echo -e "${GREEN}基础设施:${NC}"
    cd "$SCRIPT_DIR"
    docker compose -f docker-compose.infra.yml ps 2>/dev/null || echo "  (基础设施未启动)"
    
    echo ""
    echo -e "${GREEN}应用服务:${NC}"
    for svc in gateway user product cart order payment address; do
        if docker ps --filter "name=demo-$svc" --format "{{.Names}}" 2>/dev/null | grep -q "demo-$svc"; then
            echo -e "  ${GREEN}● demo-$svc${NC} 运行中"
        else
            echo -e "  ${RED}○ demo-$svc${NC} 已停止"
        fi
    done
}

# 查看日志
logs() {
    local service=${1:-gateway}
    cd "$SCRIPT_DIR"
    
    if [ "$service" = "mysql" ] || [ "$service" = "redis" ] || [ "$service" = "etcd" ] || [ "$service" = "rabbitmq" ]; then
        docker compose -f docker-compose.infra.yml logs -f "$service"
    else
        docker compose -f "docker-compose.$service.yml" logs -f "$service"
    fi
}

# 清理所有
clean() {
    log_warn "清理所有容器..."
    cd "$SCRIPT_DIR"
    
    for yml in docker-compose.*.yml; do
        svc=$(echo $yml | sed 's/docker-compose.\(.*\)\.yml/\1/')
        docker compose -f "$yml" down 2>/dev/null || true
    done
    
    docker network rm $NETWORK_NAME 2>/dev/null || true
    log_info "清理完成"
}

# 构建所有服务镜像 (批量，加速后续启动)
build_all() {
    log_info "构建所有服务镜像..."
    cd "$SCRIPT_DIR"
    export DOCKER_BUILDKIT=1
    
    # 批量构建
    for svc in gateway user product cart order payment address; do
        log_info "构建 $svc..."
        docker compose -f "docker-compose.$svc.yml" build --no-cache "$svc" || true
    done
    
    log_info "所有镜像构建完成!"
}

# 显示帮助
help() {
    cat << EOF

===============================================
  电商微服务 - 单机部署脚本
===============================================

基础设施:
  infra-start      启动基础设施 (MySQL/Redis/Etcd/RabbitMQ)
  infra-stop       停止基础设施

构建 (只需运行一次):
  build-deps       预构建 Go 依赖 (加速首次构建)
  build-all        预构建所有服务镜像

服务:
  start <svc>      启动单个服务
  stop <svc>       停止单个服务
  gateway          启动 Gateway

其他:
  status           查看所有服务状态
  logs [svc]       查看日志 (默认: gateway)
  clean            清理所有容器和网络

可用服务: gateway, user, product, cart, order, payment, address

快速开始:
  1. make run-build-deps     # 只运行一次
  2. make run-infra          # 启动基础设施
  3. make run-gateway        # 启动网关 (现在会很快!)
  4. make run-start SVC=user # 启动其他服务 (会使用缓存)

EOF
}

case "${1:-help}" in
    build-deps)
        build_deps
        ;;
    build-all)
        build_all
        ;;
    infra-start)
        start_infra
        ;;
    infra-stop)
        stop_infra
        ;;
    start)
        [ -z "$2" ] && { log_error "请指定服务名"; exit 1; }
        start_service "$2"
        ;;
    stop)
        [ -z "$2" ] && { log_error "请指定服务名"; exit 1; }
        stop_service "$2"
        ;;
    gateway)
        start_service "gateway"
        ;;
    status)
        status
        ;;
    logs)
        logs "${2:-gateway}"
        ;;
    clean)
        clean
        ;;
    help|*)
        help
        ;;
esac
