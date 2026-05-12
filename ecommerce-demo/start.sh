#!/bin/bash

# ============================================
# 电商微服务一键启动脚本
# ============================================

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "  电商微服务一键启动脚本"
echo "==========================================="

# 1. 检查并启动基础设施服务
echo ""
echo "📦 检查基础设施服务..."

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 检查并启动 docker-compose 服务
if ! docker ps --format '{{.Names}}' | grep -q "demo-rabbitmq"; then
    echo "🚀 启动 Docker Compose 服务 (MySQL, Redis, Etcd, RabbitMQ)..."
    docker-compose -f deploy/docker-compose.yml up -d
    echo "⏳ 等待服务启动..."
    sleep 5
else
    echo "✅ 基础设施服务已在运行"
fi

# 2. 检查 RabbitMQ 是否就绪
echo ""
echo "🔍 检查 RabbitMQ 连接..."
RABBITMQ_RUNNING=$(docker ps --filter "name=demo-rabbitmq" --filter "status=running" -q)
if [ -z "$RABBITMQ_RUNNING" ]; then
    echo "❌ RabbitMQ 未运行，请检查 Docker 容器状态"
    echo "   运行: docker logs demo-rabbitmq 查看日志"
    exit 1
fi
echo "✅ RabbitMQ 运行正常"

# 3. 启动 Go 微服务
echo ""
echo "🚀 启动 Go 微服务..."

# 使用系统通知显示启动状态（macOS）
notify() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        osascript -e "display notification \"$2\" with title \"$1\""
    fi
}

start_service() {
    local name=$1
    local dir=$2
    local cmd=$3

    echo "   ▶️  启动 $name..."
    osascript -e "tell application \"Terminal\" to do script \"cd '$SCRIPT_DIR/app/$dir' && $cmd\""
    sleep 1
}

# 启动顺序很重要：先启动基础服务，再启动依赖服务
start_service "User RPC" "user" "go run user.go -f etc/user.yaml"
start_service "Product RPC" "product" "go run product.go -f etc/product.yaml"
start_service "Order RPC (主服务)" "order" "go run order.go -f etc/order.yaml"
start_service "Order 延迟消费者" "order/cmd/delay" "go run main.go -f ../../etc/order.yaml"
start_service "Order 定时扫描" "order/cmd/cron" "go run main.go -f ../../etc/order.yaml"
start_service "Cart RPC" "cart" "go run cart.go -f etc/cart.yaml"
start_service "Payment RPC" "payment" "go run payment.go -f etc/payment.yaml"
start_service "Address RPC" "address" "go run address.go -f etc/address.yaml"
start_service "API Gateway" "gateway" "go run gateway.go -f etc/gateway.yaml"

echo ""
echo "==========================================="
echo "  ✅ 所有服务启动中！"
echo "==========================================="
echo ""
echo "📋 服务端口参考："
echo "   - API Gateway:    http://localhost:8888"
echo "   - User RPC:       localhost:8080"
echo "   - Product RPC:    localhost:8081"
echo "   - Order RPC:      localhost:8082"
echo "   - Cart RPC:       localhost:8083"
echo "   - Payment RPC:    localhost:8084"
echo "   - Address RPC:    localhost:8085"
echo ""
echo "🔧 运维端口："
echo "   - RabbitMQ:       http://localhost:15672 (guest/guest)"
echo "   - MySQL:          localhost:3306 (root/root)"
echo "   - Redis:          localhost:6379"
echo "   - Etcd:           localhost:2379"
echo ""
echo "==========================================="
echo ""

# 通知用户
notify "微服务启动完成" "所有微服务已在后台终端中启动"
