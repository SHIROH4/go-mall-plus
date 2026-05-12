<template>
  <div class="order-detail-page" v-loading="loading">
    <div class="detail-container" v-if="order">
      <h1>订单详情</h1>
      
      <!-- 订单状态 -->
      <div class="status-card" :class="getStatusClass(order.status)">
        <div class="status-icon">
          <el-icon :size="48"><component :is="getStatusIcon(order.status)" /></el-icon>
        </div>
        <div class="status-info">
          <h2>{{ order.status_text }}</h2>
          <p v-if="order.status === 0">请在 {{ formatCountdown(order.expire_time) }} 内完成支付</p>
          <p v-else-if="order.status === 1">支付时间：{{ formatDateTime(order.pay_time) }}</p>
        </div>
      </div>
      
      <!-- 订单信息 -->
      <div class="section order-info">
        <h2>订单信息</h2>
        <div class="info-grid">
          <div class="info-item">
            <span class="label">订单编号</span>
            <span class="value">{{ order.order_no }}</span>
          </div>
          <div class="info-item">
            <span class="label">下单时间</span>
            <span class="value">{{ formatDateTime(order.create_time) }}</span>
          </div>
          <div class="info-item">
            <span class="label">订单状态</span>
            <span class="value" :class="getStatusText(order.status)">{{ order.status_text }}</span>
          </div>
          <div class="info-item" v-if="order.status === 0">
            <span class="label">支付截止</span>
            <span class="value countdown">{{ formatCountdown(order.expire_time) }}</span>
          </div>
        </div>
      </div>
      
      <!-- 商品信息 -->
      <div class="section goods-info">
        <h2>商品信息</h2>
        <div class="goods-item">
          <el-image :src="order.product_image" class="goods-image" />
          <div class="goods-detail">
            <h3>{{ order.product_name }}</h3>
            <p class="goods-desc">{{ order.product_desc }}</p>
            <div class="goods-meta">
              <span>数量：{{ order.count }}</span>
              <span class="goods-price">¥{{ formatPrice(order.total_amount) }}</span>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 金额信息 -->
      <div class="section amount-info">
        <h2>金额信息</h2>
        <div class="amount-row">
          <span class="label">商品总额</span>
          <span class="value">¥{{ formatPrice(order.total_amount) }}</span>
        </div>
        <div class="amount-row total">
          <span class="label">实付金额</span>
          <span class="value">¥{{ formatPrice(order.total_amount) }}</span>
        </div>
      </div>
      
      <!-- 操作按钮 -->
      <div class="action-section">
        <el-button
          v-if="order.status === 0"
          type="danger"
          size="large"
          @click="$router.push(`/payment/${order.order_no}`)"
        >
          立即支付
        </el-button>
        <el-button
          v-if="order.status === 0"
          size="large"
          @click="handleCancel"
        >
          取消订单
        </el-button>
        <el-button size="large" @click="$router.push('/order/list')">
          返回订单列表
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Clock, CheckCircle, Close, ExclamationPoint, CircleCheck } from '@element-plus/icons-vue'
import { getOrderDetail, cancelOrder } from '@/api/order'
import { formatPrice, formatDateTime, formatCountdown } from '@/utils'
import type { GetOrderDetailResp } from '@/types'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const order = ref<GetOrderDetailResp | null>(null)

const orderNo = route.params.orderNo as string

const getStatusClass = (status: number) => {
  const classMap: Record<number, string> = {
    0: 'status-pending',
    1: 'status-paid',
    2: 'status-cancelled',
    3: 'status-expired'
  }
  return classMap[status] || ''
}

const getStatusIcon = (status: number) => {
  const iconMap: Record<number, any> = {
    0: Clock,
    1: CircleCheck,
    2: Close,
    3: ExclamationPoint
  }
  return iconMap[status] || Clock
}

const getStatusText = (status: number) => {
  const classMap: Record<number, string> = {
    0: 'text-warning',
    1: 'text-success',
    2: 'text-muted',
    3: 'text-danger'
  }
  return classMap[status] || ''
}

const fetchOrderDetail = async () => {
  loading.value = true
  try {
    const res = await getOrderDetail({ order_no: orderNo })
    order.value = res
  } catch (error) {
    console.error('Failed to fetch order detail:', error)
    ElMessage.error('订单不存在')
  } finally {
    loading.value = false
  }
}

const handleCancel = async () => {
  await ElMessageBox.confirm('确定要取消这个订单吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await cancelOrder({ order_no: orderNo })
    ElMessage.success('订单已取消')
    fetchOrderDetail()
  } catch (error: any) {
    ElMessage.error(error.message || '取消失败')
  }
}

onMounted(() => {
  fetchOrderDetail()
})
</script>

<style scoped>
.order-detail-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.detail-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 20px;
}

.detail-container > h1 {
  font-size: 24px;
  color: #333;
  margin-bottom: 20px;
}

/* 状态卡片 */
.status-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 30px;
  border-radius: 12px;
  margin-bottom: 20px;
  color: white;
}

.status-pending {
  background: linear-gradient(135deg, #fa8c16, #ff9a3c);
}

.status-paid {
  background: linear-gradient(135deg, #52c41a, #73d13d);
}

.status-cancelled {
  background: linear-gradient(135deg, #999, #bfbfbf);
}

.status-expired {
  background: linear-gradient(135deg, #ff4d4f, #ff7875);
}

.status-info h2 {
  font-size: 24px;
  margin-bottom: 4px;
}

.status-info p {
  font-size: 14px;
  opacity: 0.9;
}

/* 通用区域 */
.section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.section h2 {
  font-size: 16px;
  color: #333;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

/* 订单信息 */
.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item .label {
  font-size: 13px;
  color: #999;
}

.info-item .value {
  font-size: 14px;
  color: #333;
}

.text-warning { color: #fa8c16; }
.text-success { color: #52c41a; }
.text-muted { color: #999; }
.text-danger { color: #ff4d4f; }

/* 商品信息 */
.goods-item {
  display: flex;
  gap: 16px;
}

.goods-image {
  width: 120px;
  height: 120px;
  border-radius: 8px;
  flex-shrink: 0;
}

.goods-detail {
  flex: 1;
}

.goods-detail h3 {
  font-size: 16px;
  color: #333;
  margin-bottom: 8px;
}

.goods-desc {
  font-size: 13px;
  color: #999;
  margin-bottom: 12px;
}

.goods-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: #666;
}

.goods-price {
  font-size: 18px;
  color: #ff4d4f;
  font-weight: bold;
}

/* 金额信息 */
.amount-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
}

.amount-row.total {
  border-top: 1px solid #f0f0f0;
  margin-top: 12px;
  padding-top: 16px;
}

.amount-row .label {
  color: #666;
}

.amount-row .value {
  color: #333;
}

.amount-row.total .value {
  font-size: 20px;
  color: #ff4d4f;
  font-weight: bold;
}

/* 操作按钮 */
.action-section {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 20px 0;
}

.action-section .el-button {
  min-width: 120px;
}

@media (max-width: 768px) {
  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
