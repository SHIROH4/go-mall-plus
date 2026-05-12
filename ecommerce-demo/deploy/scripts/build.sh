#!/bin/bash
# =============================================================================
# 构建脚本 - 构建所有服务的 Docker 镜像
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# 镜像仓库前缀
REGISTRY="${REGISTRY:-your-registry.com}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 需要构建的服务列表
SERVICES=(
    "gateway"
    "user"
    "product"
    "cart"
    "order"
    "payment"
    "address"
)

ORDER_CMDS=(
    "delay"
    "cron"
    "dlq"
)

# 构建单个服务镜像
build_service() {
    local service=$1
    local image_name="$REGISTRY/ecommerce-$service:$IMAGE_TAG"

    log_info "构建 $service..."

    if [ ! -f "$PROJECT_DIR/app/$service/Dockerfile" ]; then
        log_warn "$service 没有 Dockerfile，跳过"
        return
    fi

    docker build -t "$image_name" -f "$PROJECT_DIR/app/$service/Dockerfile" "$PROJECT_DIR"
    log_info "$service 构建完成: $image_name"
}

# 构建 order 后台任务
build_order_cmd() {
    local cmd=$1
    local image_name="$REGISTRY/ecommerce-order-$cmd:$IMAGE_TAG"

    log_info "构建 order-$cmd..."

    if [ ! -f "$PROJECT_DIR/app/order/cmd/$cmd/Dockerfile" ]; then
        log_warn "order-$cmd 没有 Dockerfile，跳过"
        return
    fi

    docker build -t "$image_name" -f "$PROJECT_DIR/app/order/cmd/$cmd/Dockerfile" "$PROJECT_DIR"
    log_info "order-$cmd 构建完成: $image_name"
}

# 推送镜像
push_service() {
    local service=$1
    local image_name="$REGISTRY/ecommerce-$service:$IMAGE_TAG"

    log_info "推送 $service..."
    docker push "$image_name"
    log_info "$service 推送完成"
}

# 构建所有服务
build_all() {
    log_info "开始构建所有服务镜像..."
    log_info "镜像仓库: $REGISTRY"
    log_info "镜像标签: $IMAGE_TAG"

    for service in "${SERVICES[@]}"; do
        build_service "$service"
    done

    for cmd in "${ORDER_CMDS[@]}"; do
        build_order_cmd "$cmd"
    done

    log_info "所有服务构建完成!"
}

# 推送所有镜像
push_all() {
    log_info "开始推送所有镜像..."

    for service in "${SERVICES[@]}"; do
        push_service "$service"
    done

    for cmd in "${ORDER_CMDS[@]}"; do
        local image_name="$REGISTRY/ecommerce-order-$cmd:$IMAGE_TAG"
        log_info "推送 order-$cmd..."
        docker push "$image_name"
    done

    log_info "所有镜像推送完成!"
}

# 构建并推送
build_push() {
    build_all
    push_all
}

# 显示帮助
help() {
    echo "用法: $0 <command> [options]"
    echo ""
    echo "命令:"
    echo "  build     构建所有镜像"
    echo "  push      推送所有镜像"
    echo "  all       构建并推送所有镜像"
    echo ""
    echo "环境变量:"
    echo "  REGISTRY  镜像仓库地址 (默认: your-registry.com)"
    echo "  IMAGE_TAG 镜像标签 (默认: latest)"
    echo ""
    echo "示例:"
    echo "  REGISTRY=docker.io/myuser IMAGE_TAG=v1.0.0 $0 all"
}

# 主函数
case "${1:-help}" in
    build)
        build_all
        ;;
    push)
        push_all
        ;;
    all)
        build_push
        ;;
    help|*)
        help
        ;;
esac
