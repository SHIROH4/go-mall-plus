#!/bin/bash
# =============================================================================
# Kind 本地测试脚本
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
NAMESPACE="${NAMESPACE:-ecommerce}"
CHART_DIR="$PROJECT_DIR/deploy/helm/ecommerce"

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

# 创建 Kind 集群
create_cluster() {
    log_info "创建 Kind 集群..."

    if kubectl config get-contexts kind-kind &> /dev/null; then
        log_warn "Kind 集群已存在"
        return
    fi

    cat <<EOF | kind create cluster --name kind --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30080
        hostPort: 30080
        protocol: TCP
      - containerPort: 30081
        hostPort: 30081
        protocol: TCP
    labels:
      ingress-ready: "true"
EOF

    log_info "Kind 集群创建完成"
}

# 安装 Ingress Controller
install_ingress() {
    log_info "安装 Nginx Ingress Controller..."

    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.0/deploy/static/provider/kind/deploy.yaml

    # 等待 Ingress Controller 就绪
    log_info "等待 Ingress Controller 就绪..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=120s || true

    log_info "Ingress Controller 安装完成"
}

# 安装 grpc_health_probe
install_grpc_health_probe() {
    log_info "安装 grpc_health_probe..."

    # 使用 DaemonSet 方式部署到每个节点
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: grpc-health-probe
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: grpc-health-probe
  template:
    metadata:
      labels:
        name: grpc-health-probe
    spec:
      containers:
        - name: grpc-health-probe
          image: registry.k8s.io/grpc-health-probe:v0.4.12
          command:
            - /bin/sh
            - -c
            - |
              while true; do
                sleep 3600
              done
      hostNetwork: true
      tolerations:
        - operator: Exists
EOF

    log_info "grpc_health_probe 安装完成"
}

# 构建并加载镜像到 Kind
build_and_load_images() {
    log_info "构建并加载镜像到 Kind..."

    # 服务列表
    SERVICES=("gateway" "user" "product" "cart" "order" "payment" "address")
    ORDER_CMDS=("delay" "cron" "dlq")

    for svc in "${SERVICES[@]}"; do
        local dockerfile="$PROJECT_DIR/app/$svc/Dockerfile"
        if [ -f "$dockerfile" ]; then
            log_info "构建 $svc..."
            docker build -t ecommerce-$svc:latest -f "$dockerfile" "$PROJECT_DIR"
            kind load docker-image ecommerce-$svc:latest --name kind
        fi
    done

    for cmd in "${ORDER_CMDS[@]}"; do
        local dockerfile="$PROJECT_DIR/app/order/cmd/$cmd/Dockerfile"
        if [ -f "$dockerfile" ]; then
            log_info "构建 order-$cmd..."
            docker build -t ecommerce-order-$cmd:latest -f "$dockerfile" "$PROJECT_DIR"
            kind load docker-image ecommerce-order-$cmd:latest --name kind
        fi
    done

    log_info "镜像构建并加载完成"
}

# 部署电商应用
deploy_app() {
    log_info "部署电商应用..."

    # 先构建并加载镜像
    build_and_load_images

    # 使用本地 Helm Chart，registry 设置为本地
    helm upgrade --install ecommerce "$CHART_DIR" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --set global.registry=ecommerce \
        --set gateway.imageTag=latest \
        --wait --timeout 10m

    log_info "应用部署完成"
}

# 显示状态
status() {
    log_info "集群状态:"
    kubectl get nodes

    echo ""

    log_info "应用状态:"
    kubectl get pods -n "$NAMESPACE"

    echo ""

    log_info "服务状态:"
    kubectl get svc -n "$NAMESPACE"

    echo ""

    log_info "Ingress 状态:"
    kubectl get ingress -n "$NAMESPACE"

    echo ""
    log_info "访问地址:"
    log_info "  NodePort: http://localhost:30088"
    log_info "  端口转发: kubectl port-forward -n $NAMESPACE svc/ecommerce-ecommerce-gateway 8888:80"
}

# 清理
cleanup() {
    log_warn "删除 Kind 集群..."
    kind delete cluster --name kind
    log_info "清理完成"
}

# 显示帮助
help() {
    echo "用法: $0 <command>"
    echo ""
    echo "命令:"
    echo "  create    创建 Kind 集群并安装必要组件"
    echo "  deploy    部署电商应用到 Kind"
    echo "  status    查看集群和应用状态"
    echo "  cleanup   删除 Kind 集群"
    echo "  all       一键创建并部署"
}

# 主函数
case "${1:-help}" in
    create)
        create_cluster
        install_ingress
        install_grpc_health_probe
        ;;
    deploy)
        deploy_app
        ;;
    status)
        status
        ;;
    cleanup)
        cleanup
        ;;
    all)
        create_cluster
        install_ingress
        install_grpc_health_probe
        deploy_app
        status
        ;;
    help|*)
        help
        ;;
esac
