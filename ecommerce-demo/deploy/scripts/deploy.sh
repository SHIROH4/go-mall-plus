#!/bin/bash
# =============================================================================
# 部署脚本 - 完整部署
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
NAMESPACE="${NAMESPACE:-ecommerce}"
CHART_DIR="$PROJECT_DIR/deploy/helm/ecommerce"

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

# 检查 helm 是否安装
check_helm() {
    if ! command -v helm &> /dev/null; then
        log_error "Helm 未安装，请先安装 Helm: https://helm.sh/docs/intro/install/"
        exit 1
    fi
}

# 检查 kubectl 是否安装
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装，请先安装 kubectl"
        exit 1
    fi
}

# 部署函数
deploy() {
    local release_name="${RELEASE_NAME:-ecommerce}"
    local registry="${REGISTRY:-your-registry.com}"

    check_helm
    check_kubectl

    log_info "开始部署电商微服务..."
    log_info "命名空间: $NAMESPACE"
    log_info "Release名称: $release_name"

    # 1. 创建命名空间
    log_info "Step 1: 创建命名空间..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # 2. 打包 Helm Chart
    log_info "Step 2: 打包 Helm Chart..."
    cd "$CHART_DIR/.."
    helm package ecommerce
    cd - > /dev/null

    # 3. 安装/升级 Release
    log_info "Step 3: 安装/升级 Helm Release..."
    helm upgrade --install "$release_name" "$CHART_DIR/../ecommerce-1.0.0.tgz" \
        --namespace "$NAMESPACE" \
        --set global.registry="$registry" \
        --wait --timeout 10m

    # 4. 等待 Pod 就绪
    log_info "Step 4: 等待 Pod 就绪..."
    kubectl rollout status deployment -n "$NAMESPACE" --timeout=300s || true

    # 5. 显示状态
    log_info "部署完成!"
    kubectl get pods -n "$NAMESPACE"
}

# 卸载函数
uninstall() {
    local release_name="${RELEASE_NAME:-ecommerce}"

    check_helm

    log_warn "即将删除 $release_name Release..."
    helm uninstall "$release_name" -n "$NAMESPACE" || true

    log_warn "是否删除命名空间? (Ctrl+C 取消)"
    sleep 3
    kubectl delete namespace "$NAMESPACE" || true

    log_info "卸载完成"
}

# 查看状态
status() {
    kubectl get pods -n "$NAMESPACE" -o wide
    echo ""
    kubectl get svc -n "$NAMESPACE"
    echo ""
    kubectl get ingress -n "$NAMESPACE"
}

# 查看日志
logs() {
    local service="${1:-gateway}"
    kubectl logs -n "$NAMESPACE" -l "app=$service" --tail=100 -f
}

# 端口转发
port_forward() {
    local service="${1:-gateway}"
    local port="${2:-8888}"

    log_info "端口转发: $service:$port"
    kubectl port-forward -n "$NAMESPACE" svc/"$release_name-$service" $port:80
}

# 进入 Pod 调试
debug() {
    local service="${1:-gateway}"

    log_info "进入 $service Pod..."
    kubectl exec -it -n "$NAMESPACE" \
        "$(kubectl get pod -n "$NAMESPACE" -l "app=$service" -o jsonpath='{.items[0].metadata.name}')" \
        -- sh
}

# 显示帮助
help() {
    echo "用法: $0 <command> [options]"
    echo ""
    echo "命令:"
    echo "  deploy        部署所有服务"
    echo "  uninstall    卸载所有服务"
    echo "  status       查看部署状态"
    echo "  logs <svc>   查看服务日志 (默认: gateway)"
    echo "  port-forward <svc> <port>  端口转发"
    echo "  debug <svc>  进入 Pod 调试"
    echo ""
    echo "环境变量:"
    echo "  NAMESPACE    命名空间 (默认: ecommerce)"
    echo "  REGISTRY     镜像仓库地址 (默认: your-registry.com)"
    echo "  RELEASE_NAME Helm Release名称 (默认: ecommerce)"
}

# 主函数
case "${1:-help}" in
    deploy)
        deploy
        ;;
    uninstall)
        uninstall
        ;;
    status)
        status
        ;;
    logs)
        logs "${2:-gateway}"
        ;;
    port-forward)
        port_forward "${2:-gateway}" "${3:-8888}"
        ;;
    debug)
        debug "${2:-gateway}"
        ;;
    help|*)
        help
        ;;
esac
