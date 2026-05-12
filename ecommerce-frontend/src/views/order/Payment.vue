<template>
  <div class="payment-page" v-loading="loading">
    <div class="payment-container" v-if="orderInfo">
      <!-- 订单信息 -->
      <div class="order-summary">
        <div class="order-amount">
          <span class="label">应付金额</span>
          <span class="amount">¥{{ formatPrice(orderInfo.total_amount) }}</span>
        </div>
        <div class="order-no">
          订单号：{{ orderNo }}
        </div>
      </div>
      
      <!-- 支付方式 -->
      <div class="payment-methods">
        <h2>选择支付方式</h2>
        <div class="method-list">
          <div
            class="method-item"
            :class="{ selected: selectedMethod === 'alipay' }"
            @click="selectedMethod = 'alipay'"
          >
            <el-icon :size="40" color="#1677ff"><Coin /></el-icon>
            <div class="method-info">
              <span class="method-name">支付宝</span>
              <span class="method-desc">推荐有支付宝账户的用户使用</span>
            </div>
            <el-icon v-if="selectedMethod === 'alipay'" :size="24" color="#1890ff"><Check /></el-icon>
          </div>
          
          <div
            class="method-item"
            :class="{ selected: selectedMethod === 'wechat' }"
            @click="selectedMethod = 'wechat'"
          >
            <el-icon :size="40" color="#07c160"><Wallet /></el-icon>
            <div class="method-info">
              <span class="method-name">微信支付</span>
              <span class="method-desc">推荐有微信账户的用户使用</span>
            </div>
            <el-icon v-if="selectedMethod === 'wechat'" :size="24" color="#1890ff"><Check /></el-icon>
          </div>
        </div>
      </div>
      
      <!-- 支付按钮 -->
      <div class="payment-action">
        <el-button
          type="primary"
          size="large"
          :loading="paying"
          @click="handlePay"
        >
          确认支付 ¥{{ formatPrice(orderInfo.total_amount) }}
        </el-button>
        
        <el-button size="large" @click="handleCancel">
          取消支付
        </el-button>
      </div>
      
      <!-- 倒计时提示 -->
      <div class="countdown-tip" v-if="orderInfo.expire_time">
        <el-icon><Clock /></el-icon>
        <span>请在 {{ formatCountdown(orderInfo.expire_time) }} 内完成支付，超时订单将自动取消</span>
      </div>
    </div>
    
    <!-- 支付成功弹窗 -->
    <el-dialog
      v-model="showSuccessDialog"
      title="支付成功"
      width="400px"
      :close-on-click-modal="false"
      :show-close="false"
    >
      <div class="success-content">
        <el-icon :size="64" color="#52c41a"><CircleCheck /></el-icon>
        <h2>恭喜您，支付成功！</h2>
        <p>您的订单正在处理中...</p>
      </div>
      <template #footer>
        <el-button type="primary" @click="goToOrderList">查看订单</el-button>
        <el-button @click="goToHome">继续购物</el-button>
      </template>
    </el-dialog>
    
    <!-- 支付取消确认 -->
    <el-dialog
      v-model="showCancelDialog"
      title="提示"
      width="400px"
    >
      <p>确定要取消支付吗？取消后订单将被关闭。</p>
      <template #footer>
        <el-button type="danger" @click="confirmCancel">确定取消</el-button>
        <el-button @click="showCancelDialog = false">继续支付</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Coin, Wallet, Check, Clock, CircleCheck } from '@element-plus/icons-vue'
import { getOrderDetail } from '@/api/order'
import { createPay, getPayStatus } from '@/api/payment'
import { formatPrice, formatCountdown } from '@/utils'
import type { GetOrderDetailResp } from '@/types'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const orderInfo = ref<GetOrderDetailResp | null>(null)
const selectedMethod = ref<'alipay' | 'wechat'>('alipay')
const paying = ref(false)
const showSuccessDialog = ref(false)
const showCancelDialog = ref(false)

const orderNo = route.params.orderNo as string

let pollTimer: number | null = null

// 获取订单信息
const fetchOrderInfo = async () => {
  loading.value = true
  try {
    const res = await getOrderDetail({ order_no: orderNo })
    orderInfo.value = res
    
    // 如果订单已支付，跳转到订单列表
    if (res.status === 1) {
      ElMessage.success('订单已支付')
      router.push('/order/list')
    }
  } catch (error) {
    console.error('Failed to fetch order info:', error)
    ElMessage.error('订单不存在')
  } finally {
    loading.value = false
  }
}

// 处理支付
const handlePay = async () => {
  if (!orderInfo.value) return
  
  paying.value = true
  
  try {
    // 创建支付
    const payRes = await createPay({
      order_no: orderNo,
      amount: orderInfo.value.total_amount,
      pay_channel: selectedMethod.value
    })
    
    ElMessage.success('支付创建成功')
    
    // 轮询支付状态
    startPollPayStatus(payRes.payment_no)
  } catch (error: any) {
    ElMessage.error(error.message || '支付创建失败')
    paying.value = false
  }
}

// 轮询支付状态
const startPollPayStatus = (paymentNo: string) => {
  pollTimer = window.setInterval(async () => {
    try {
      const res = await getPayStatus(paymentNo)
      
      if (res.status === 1) {
        // 支付成功
        stopPollPayStatus()
        paying.value = false
        showSuccessDialog.value = true
      } else if (res.status === 2 || res.status === 3) {
        // 支付取消或超时
        stopPollPayStatus()
        paying.value = false
        ElMessage.error(res.status_text)
      }
    } catch (error) {
      console.error('Failed to poll pay status:', error)
    }
  }, 2000)
}

const stopPollPayStatus = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 取消支付
const handleCancel = () => {
  showCancelDialog.value = true
}

const confirmCancel = async () => {
  try {
    const { cancelOrder } = await import('@/api/order')
    await cancelOrder({ order_no: orderNo })
    ElMessage.success('订单已取消')
    router.push('/order/list')
  } catch (error: any) {
    ElMessage.error(error.message || '取消失败')
  }
}

// 跳转
const goToOrderList = () => {
  showSuccessDialog.value = false
  router.push('/order/list')
}

const goToHome = () => {
  showSuccessDialog.value = false
  router.push('/home')
}

onMounted(() => {
  fetchOrderInfo()
})

onUnmounted(() => {
  stopPollPayStatus()
})
</script>

<style scoped>
.payment-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 40px 0;
}

.payment-container {
  max-width: 500px;
  margin: 0 auto;
  padding: 0 20px;
}

/* 订单摘要 */
.order-summary {
  background: white;
  border-radius: 12px;
  padding: 30px;
  text-align: center;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.order-amount {
  margin-bottom: 16px;
}

.order-amount .label {
  display: block;
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.order-amount .amount {
  font-size: 36px;
  color: #ff4d4f;
  font-weight: bold;
}

.order-no {
  font-size: 13px;
  color: #999;
}

/* 支付方式 */
.payment-methods {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.payment-methods h2 {
  font-size: 16px;
  color: #333;
  margin-bottom: 16px;
}

.method-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.method-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border: 2px solid #f0f0f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.method-item:hover {
  border-color: #1890ff;
}

.method-item.selected {
  border-color: #1890ff;
  background: #f0f7ff;
}

.method-info {
  flex: 1;
}

.method-name {
  display: block;
  font-size: 15px;
  color: #333;
  font-weight: 500;
  margin-bottom: 2px;
}

.method-desc {
  font-size: 12px;
  color: #999;
}

/* 支付按钮 */
.payment-action {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.payment-action .el-button {
  height: 48px;
  font-size: 16px;
}

/* 倒计时提示 */
.countdown-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  background: #fff7e6;
  border-radius: 8px;
  color: #fa8c16;
  font-size: 13px;
}

/* 支付成功弹窗 */
.success-content {
  text-align: center;
  padding: 20px 0;
}

.success-content h2 {
  font-size: 20px;
  color: #333;
  margin: 16px 0 8px;
}

.success-content p {
  color: #666;
  font-size: 14px;
}
</style>
