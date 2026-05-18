#!/bin/bash
# ============================================
# 一键停机后完整启动脚本
# 处理：Redis Cluster 重建、MQ 清理、服务顺序启动、健康验证
# ============================================
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${BLUE}==> $1${NC}"; }

echo "==========================================="
echo "  微服务完整启动脚本"
echo "==========================================="

# ==========================================
# Step 1: 确保 namespace 和 secrets 存在
# ==========================================
log_step "1. 初始化基础设施配置..."
kubectl apply -f "$SCRIPT_DIR/namespace.yaml"
kubectl apply -f "$SCRIPT_DIR/secrets.yaml"
log_info "namespace + secrets 就绪"

# ==========================================
# Step 2: 部署 MySQL 和 RabbitMQ（不动 Redis）
# ==========================================
log_step "2. 部署 MySQL + RabbitMQ..."
kubectl apply -f "$SCRIPT_DIR/mysql-statefulset.yaml"
kubectl apply -f "$SCRIPT_DIR/rabbitmq.yaml" 2>/dev/null || true

log_info "等待 MySQL..."
kubectl wait --for=condition=ready pod -l app=mysql -n ecommerce --timeout=300s

log_info "等待 RabbitMQ..."
kubectl wait --for=condition=ready pod -l app=rabbitmq -n ecommerce --timeout=180s
log_info "MySQL + RabbitMQ 就绪"

# ==========================================
# Step 3: 部署 Redis Cluster StatefulSet（不含 Job）
# ==========================================
log_step "3. 部署 Redis Cluster StatefulSet..."

kubectl apply -f "$SCRIPT_DIR/redis-cluster.yaml"

# 立即删除 Job，避免在 Pod 就绪前半途运行破坏集群状态
kubectl delete job redis-cluster-init -n ecommerce --ignore-not-found=true

log_info "等待所有 Redis Pod Ready（最多 5 分钟）..."
for pod in redis-0 redis-1 redis-2 redis-3 redis-4 redis-5; do
    kubectl wait --for=condition=ready pod/${pod} -n ecommerce --timeout=300s
done
log_info "6 个 Redis Pod 全部就绪"

# ==========================================
# Step 4: 初始化 Redis Cluster
# ==========================================
log_step "4. 组建 Redis Cluster..."

# 确保所有节点 Redis 服务完全就绪
for i in 0 1 2 3 4 5; do
    for attempt in $(seq 1 15); do
        if kubectl exec -n ecommerce redis-$i -- redis-cli -a redis123456 --no-auth-warning ping 2>/dev/null | grep -q PONG; then
            break
        fi
        sleep 2
    done
done

# 重置所有节点（清理残留的集群状态）
for i in 0 1 2 3 4 5; do
    kubectl exec -n ecommerce redis-$i -- redis-cli -a redis123456 --no-auth-warning CLUSTER RESET HARD 2>/dev/null || true
done
sleep 2

# 重建 Job（此时所有节点干净且就绪）
kubectl apply -f "$SCRIPT_DIR/redis-cluster.yaml"

log_info "等待集群组建..."
if kubectl wait --for=condition=complete job/redis-cluster-init -n ecommerce --timeout=120s 2>/dev/null; then
    log_info "Job 完成"
else
    log_warn "Job 超时，重试一次..."
    kubectl logs job/redis-cluster-init -n ecommerce --tail=20 2>/dev/null
    kubectl delete job redis-cluster-init -n ecommerce --ignore-not-found=true
    sleep 3
    kubectl apply -f "$SCRIPT_DIR/redis-cluster.yaml"
    kubectl wait --for=condition=complete job/redis-cluster-init -n ecommerce --timeout=120s
fi

# 验证集群状态
CLUSTER_OK=$(kubectl exec -n ecommerce redis-0 -- redis-cli -a redis123456 --no-auth-warning cluster info 2>/dev/null | grep "cluster_state:ok" || true)
if [ -z "$CLUSTER_OK" ]; then
    log_error "Redis Cluster 组建失败！"
    kubectl exec -n ecommerce redis-0 -- redis-cli -a redis123456 --no-auth-warning cluster info 2>/dev/null
    exit 1
fi
log_info "Redis Cluster 状态: OK"

# ==========================================
# Step 5: 清理旧消息（防止重复消费）
# ==========================================
log_step "5. 清理旧消息队列..."

# RabbitMQ 清空队列
for q in order_create_queue order.delay.queue order.dlq order.dlq.delay; do
    kubectl exec -n ecommerce deploy/rabbitmq -- rabbitmqadmin purge queue name=$q 2>/dev/null || true
done

# MySQL 清理已完成/死信的 outbox 记录
kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo \
    -e "DELETE FROM outbox WHERE status IN (2,3)" 2>/dev/null || true

log_info "旧消息清理完成"

# ==========================================
# Step 6: 部署应用服务
# ==========================================
log_step "6. 部署应用服务..."

SERVICES=(user product stock cart order payment address gateway)

for svc in "${SERVICES[@]}"; do
    kubectl apply -f "$SCRIPT_DIR/services/${svc}.yaml"
done

log_info "应用服务配置已更新"

# ==========================================
# Step 7: 按依赖顺序重启并等待就绪
# ==========================================
log_step "7. 按顺序重启服务..."

# 先重启非 Redis 依赖的服务，再重启 Redis 依赖的
RESTART_ORDER=(user product payment address stock cart order gateway)

for svc in "${RESTART_ORDER[@]}"; do
    kubectl rollout restart deployment/${svc} -n ecommerce 2>/dev/null || true
done

log_info "等待所有服务就绪..."
for svc in "${RESTART_ORDER[@]}"; do
    kubectl rollout status deployment/${svc} -n ecommerce --timeout=120s || log_warn "${svc} 就绪超时"
done

# ==========================================
# Step 8: 验证
# ==========================================
log_step "8. 验证服务健康..."

PASS=0
FAIL=0

# 注册 + 登录
curl -s -X POST 'http://localhost:30088/api/user/register' \
    -H 'Content-Type: application/json' \
    -d '{"username":"healthcheck","password":"test123456"}' > /dev/null 2>&1 || true

TOKEN=$(curl -s -X POST 'http://localhost:30088/api/user/login' \
    -H 'Content-Type: application/json' \
    -d '{"username":"healthcheck","password":"test123456"}' | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('accessToken',''))" 2>/dev/null)

# 测试商品
if curl -s 'http://localhost:30088/api/product/list' | grep -q '"code":0'; then
    log_info "商品列表: OK"
    PASS=$((PASS+1))
else
    log_error "商品列表: FAIL"
    FAIL=$((FAIL+1))
fi

# 测试购物车
if curl -s -X POST 'http://localhost:30088/api/cart/add' \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"product_id":1,"product_name":"test","price":100,"image_url":"","count":1}' | grep -q '"code":0'; then
    log_info "加入购物车: OK"
    PASS=$((PASS+1))
else
    log_error "加入购物车: FAIL"
    FAIL=$((FAIL+1))
fi

# 测试下单
if curl -s -X POST 'http://localhost:30088/api/order/create' \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"productId":1,"count":1}' | grep -q '"code":0'; then
    log_info "下单: OK"
    PASS=$((PASS+1))
else
    log_error "下单: FAIL"
    FAIL=$((FAIL+1))
fi

echo ""
echo "==========================================="
echo "  启动完成 通过=${PASS}/3  失败=${FAIL}/3"
echo "==========================================="
echo ""
echo "  Gateway:    http://localhost:30088"
echo "  Prometheus: http://localhost:30909"
echo "  Grafana:    http://localhost:30300 (admin/admin)"
echo "  RabbitMQ:   http://localhost:31672"
echo ""
