#!/bin/bash
# ============================================
# 电商压测脚本 - 模拟真实业务场景
# ============================================
set -e

GATEWAY="${GATEWAY:-http://localhost:30088}"
LOOPS="${LOOPS:-50}"
SLEEP="${SLEEP:-0.5}"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 随机用户名（避免冲突）
TIMESTAMP=$(date +%s)
USERNAME="testuser_${TIMESTAMP}"
PASSWORD="test123456"
TOKEN=""
USER_ID=""

# ---- 1. 注册 ----
register() {
    log_info "1. 注册用户: $USERNAME"
    resp=$(curl -s -X POST "$GATEWAY/api/user/register" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
    echo "   注册: $resp"
}

# ---- 2. 登录 ----
login() {
    log_info "2. 登录"
    resp=$(curl -s -X POST "$GATEWAY/api/user/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
    echo "   登录: $resp"
    TOKEN=$(echo "$resp" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)
    USER_ID=$(echo "$resp" | grep -o '"id":[0-9]*' | cut -d':' -f2)
    if [ -z "$TOKEN" ]; then
        log_error "登录失败，无法获取 token"
        exit 1
    fi
    log_info "   Token: ${TOKEN:0:20}..."
    log_info "   UserID: $USER_ID"
}

# ---- 3. 获取商品列表 ----
list_products() {
    log_info "3. 获取商品列表"
    resp=$(curl -s "$GATEWAY/api/product/list" \
        -H "Content-Type: application/json")
    product_count=$(echo "$resp" | grep -o '"id"' | wc -l | tr -d ' ')
    log_info "   商品数: $product_count"
}

# ---- 4. 获取分类 ----
get_categories() {
    log_info "4. 获取分类"
    resp=$(curl -s "$GATEWAY/api/product/categories" \
        -H "Content-Type: application/json")
    log_info "   分类: $(echo "$resp" | grep -o '"name":"[^"]*"' | head -3 | cut -d'"' -f4 | tr '\n' ', ')"
}

# ---- 5. 商品详情 ----
product_detail() {
    local product_id=$(( (RANDOM % 10) + 1 ))
    log_info "5. 商品详情 (ID=$product_id)"
    resp=$(curl -s "$GATEWAY/api/product/$product_id" \
        -H "Content-Type: application/json")
    name=$(echo "$resp" | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
    log_info "   商品: $name"
}

# ---- 6. 加入购物车 ----
add_to_cart() {
    local product_id=$(( (RANDOM % 10) + 1 ))
    log_info "6. 加入购物车 (商品ID=$product_id)"
    resp=$(curl -s -X POST "$GATEWAY/api/cart/add" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"product_id\":$product_id,\"product_name\":\"测试商品\",\"price\":1000,\"image_url\":\"\",\"count\":1}")
    log_info "   结果: $resp"
}

# ---- 7. 查看购物车 ----
view_cart() {
    log_info "7. 查看购物车"
    resp=$(curl -s "$GATEWAY/api/cart/list" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN")
    item_count=$(echo "$resp" | grep -o '"product_id"' | wc -l | tr -d ' ')
    log_info "   购物车商品数: $item_count"
}

# ---- 8. 下单 ----
create_order() {
    local product_id=$(( (RANDOM % 10) + 1 ))
    log_info "8. 下单 (商品ID=$product_id, 数量=${1:-1})"
    resp=$(curl -s -X POST "$GATEWAY/api/order/create" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"productId\":$product_id,\"count\":${1:-1}}")
    log_info "   结果: $resp"
}

# ---- 9. 查看订单列表 ----
list_orders() {
    log_info "9. 查看订单列表"
    resp=$(curl -s "$GATEWAY/api/order/list?page=1&page_size=10" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN")
    log_info "   结果: $resp"
}

# ---- 10. 支付（模拟） ----
pay_order() {
    # 先获取一个待支付的订单号
    resp=$(curl -s "$GATEWAY/api/order/list?page=1&page_size=50" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN")
    order_no=$(echo "$resp" | grep -o '"orderNo":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$order_no" ]; then
        log_warn "   没有待支付订单"
        return
    fi
    log_info "10. 支付订单: $order_no"
    pay_resp=$(curl -s -X POST "$GATEWAY/api/pay/create" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"orderNo\":\"$order_no\",\"paymentMethod\":\"alipay\"}")
    log_info "   结果: $pay_resp"
}

# ---- 主流程 ----
main() {
    echo "==========================================="
    echo "  电商压测脚本"
    echo "  Gateway: $GATEWAY"
    echo "  循环次数: $LOOPS"
    echo "  间隔: ${SLEEP}s"
    echo "==========================================="
    echo ""

    # 注册登录
    register
    sleep 1
    login
    sleep 1

    # 第一轮：浏览商品
    list_products
    sleep "$SLEEP"
    get_categories
    sleep "$SLEEP"
    product_detail
    sleep "$SLEEP"

    # 压测循环
    log_info ""
    log_info "==========================================="
    log_info "  开始压测循环 ($LOOPS 轮)"
    log_info "==========================================="
    log_info ""

    SUCCESS_ORDER=0
    FAIL_ORDER=0

    for i in $(seq 1 $LOOPS); do
        echo ""
        log_info "======== 第 $i / $LOOPS 轮 ========"

        # 随机浏览
        if [ $((RANDOM % 2)) -eq 0 ]; then
            list_products
        else
            product_detail
        fi
        sleep "$SLEEP"

        # 加入购物车
        add_to_cart
        sleep "$SLEEP"

        # 查看购物车
        if [ $((RANDOM % 3)) -eq 0 ]; then
            view_cart
            sleep "$SLEEP"
        fi

        # 下单
        qty=$(( (RANDOM % 3) + 1 ))
        resp=$(curl -s -X POST "$GATEWAY/api/order/create" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "{\"productId\":$(( (RANDOM % 10) + 1 )),\"count\":$qty}")

        if echo "$resp" | grep -q '"orderNo"'; then
            SUCCESS_ORDER=$((SUCCESS_ORDER + 1))
            log_info "   下单成功 ✅ (累计成功: $SUCCESS_ORDER)"
        else
            FAIL_ORDER=$((FAIL_ORDER + 1))
            log_warn "   下单失败 ❌ (累计失败: $FAIL_ORDER) - $resp"
        fi
        sleep "$SLEEP"

        # 随机支付
        if [ $((RANDOM % 5)) -eq 0 ] && [ -n "$TOKEN" ]; then
            pay_order
            sleep "$SLEEP"
        fi
    done

    # 最终查看订单
    echo ""
    log_info "==========================================="
    log_info "  最终订单列表"
    log_info "==========================================="
    list_orders

    echo ""
    log_info "==========================================="
    log_info "  压测完成!"
    log_info "  下单成功: $SUCCESS_ORDER"
    log_info "  下单失败: $FAIL_ORDER"
    log_info "==========================================="
}

main
