<template>
  <div class="seckill-page">
    <!-- 秒杀头部 -->
    <div class="seckill-header">
      <div class="seckill-title">
        <el-icon :size="32" color="#ff4d4f"><Lightning /></el-icon>
        <h1>限时秒杀</h1>
        <p>每日10:00、20:00准时开抢</p>
      </div>
      
      <div class="seckill-timer">
        <span class="timer-label">{{ timerLabel }}</span>
        <div class="timer-display">
          <span class="time-block">{{ countdownHours }}</span>
          <span class="time-separator">:</span>
          <span class="time-block">{{ countdownMinutes }}</span>
          <span class="time-separator">:</span>
          <span class="time-block">{{ countdownSeconds }}</span>
        </div>
      </div>
    </div>
    
    <!-- 秒杀商品列表 -->
    <div class="seckill-content">
      <div class="seckill-grid" v-loading="loading">
        <div
          v-for="product in products"
          :key="product.id"
          class="seckill-card"
        >
          <div class="card-image" @click="$router.push(`/product/detail/${product.id}`)">
            <el-image :src="product.image_url" fit="cover" />
            <div class="seckill-badge">限时特惠</div>
            <div class="seckill-countdown-overlay" v-if="countdownHours !== '00' || countdownMinutes !== '00' || countdownSeconds !== '00'">
              <span>距开始</span>
              <span class="mini-timer">{{ countdownHours }}:{{ countdownMinutes }}:{{ countdownSeconds }}</span>
            </div>
          </div>
          
          <div class="card-content">
            <h3 class="product-name">{{ product.name }}</h3>
            <p class="product-desc">{{ product.desc }}</p>
            
            <div class="price-section">
              <div class="seckill-price">
                <span class="current">¥{{ formatPrice(product.price) }}</span>
                <span class="original">¥{{ formatPrice(product.price * 1.5) }}</span>
              </div>
              <div class="discount-tag">省 ¥{{ formatPrice(product.price * 0.5) }}</div>
            </div>
            
            <div class="stock-section">
              <div class="stock-bar">
                <div class="stock-progress" :style="{ width: getStockPercent(product.stock) + '%' }"></div>
              </div>
              <span class="stock-text">剩余 {{ product.stock }} 件</span>
            </div>
            
            <div class="action-section">
              <el-button
                type="danger"
                size="large"
                :disabled="!canSeckill"
                class="seckill-btn"
                @click="handleSeckill(product)"
                :loading="seckillLoading[product.id]"
              >
                {{ countdownHours === '00' && countdownMinutes === '00' && parseInt(countdownSeconds) <= 30 ? '立即抢购' : '即将开始' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <el-empty v-if="!loading && products.length === 0" description="暂无秒杀商品" />
    </div>
    
    <!-- 秒杀规则 -->
    <div class="seckill-rules">
      <h2>秒杀规则</h2>
      <ul>
        <li>秒杀商品数量有限，售完即止</li>
        <li>每个用户每种秒杀商品限购1件</li>
        <li>秒杀订单需在15分钟内完成支付，否则自动取消</li>
        <li>秒杀商品不支持退换货</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lightning } from '@element-plus/icons-vue'
import { listProductByPage } from '@/api/product'
import { useCartStore } from '@/stores/cart'
import { useUserStore } from '@/stores/user'
import { formatPrice, debounce } from '@/utils'
import type { ProductItem } from '@/types'

const router = useRouter()
const cartStore = useCartStore()
const userStore = useUserStore()

const loading = ref(false)
const products = ref<ProductItem[]>([])
const seckillLoading = ref<Record<number, boolean>>({})
const canSeckill = ref(true)

// 秒杀倒计时
const countdownHours = ref('00')
const countdownMinutes = ref('00')
const countdownSeconds = ref('00')
const timerLabel = ref('距离下一场')
let countdownTimer: number | null = null

// 更新倒计时
const updateCountdown = () => {
  const now = new Date()
  const target = new Date()
  
  // 判断下一个秒杀场次
  if (now.getHours() < 10) {
    target.setHours(10, 0, 0, 0)
    timerLabel.value = '距10:00场'
  } else if (now.getHours() < 20) {
    target.setHours(20, 0, 0, 0)
    timerLabel.value = '距20:00场'
  } else {
    target.setDate(target.getDate() + 1)
    target.setHours(10, 0, 0, 0)
    timerLabel.value = '距明日10:00场'
  }
  
  const diff = Math.max(0, target.getTime() - now.getTime())
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)
  
  countdownHours.value = hours.toString().padStart(2, '0')
  countdownMinutes.value = minutes.toString().padStart(2, '0')
  countdownSeconds.value = seconds.toString().padStart(2, '0')
}

// 计算库存百分比
const getStockPercent = (stock: number) => {
  const maxStock = 100 // 假设最大库存
  return Math.min(100, (stock / maxStock) * 100)
}

// 获取秒杀商品
const fetchSeckillProducts = async () => {
  loading.value = true
  try {
    const res = await listProductByPage({ page: 1, page_size: 10 })
    products.value = res.products || []
  } catch (error) {
    console.error('Failed to fetch seckill products:', error)
  } finally {
    loading.value = false
  }
}

// 处理秒杀（防抖）
const handleSeckill = debounce(async (product: ProductItem) => {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  
  // 防止重复点击
  if (seckillLoading.value[product.id]) return
  
  seckillLoading.value[product.id] = true
  
  try {
    // 直接创建订单
    const { createOrder } = await import('@/api/order')
    const res = await createOrder({
      productId: product.id,
      count: 1
    })
    
    ElMessage.success('秒杀成功！请在15分钟内完成支付')
    router.push(`/payment/${res.orderNo}`)
  } catch (error: any) {
    ElMessage.error(error.message || '秒杀失败，请重试')
  } finally {
    seckillLoading.value[product.id] = false
  }
}, 1000)

onMounted(() => {
  updateCountdown()
  countdownTimer = window.setInterval(updateCountdown, 1000)
  fetchSeckillProducts()
})

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
  }
})
</script>

<style scoped>
.seckill-page {
  min-height: 100vh;
  background: #f5f5f5;
}

/* 秒杀头部 */
.seckill-header {
  background: linear-gradient(135deg, #ff4d4f 0%, #ff6b6b 100%);
  color: white;
  padding: 40px 20px;
  text-align: center;
}

.seckill-title {
  margin-bottom: 24px;
}

.seckill-title h1 {
  font-size: 36px;
  margin: 12px 0 8px;
}

.seckill-title p {
  font-size: 16px;
  opacity: 0.9;
}

.seckill-timer {
  display: inline-block;
  background: rgba(255, 255, 255, 0.2);
  padding: 16px 32px;
  border-radius: 8px;
}

.timer-label {
  display: block;
  font-size: 14px;
  margin-bottom: 8px;
  opacity: 0.9;
}

.timer-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.time-block {
  background: white;
  color: #ff4d4f;
  font-size: 28px;
  font-weight: bold;
  padding: 8px 12px;
  border-radius: 4px;
  min-width: 50px;
  font-family: monospace;
}

.time-separator {
  font-size: 28px;
  font-weight: bold;
}

/* 秒杀内容 */
.seckill-content {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px;
}

.seckill-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}

.seckill-card {
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  transition: all 0.3s;
}

.seckill-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(255, 77, 79, 0.2);
}

.card-image {
  position: relative;
  height: 220px;
  background: #f5f5f5;
  cursor: pointer;
}

.card-image .el-image {
  width: 100%;
  height: 100%;
}

.seckill-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  background: #ff4d4f;
  color: white;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: bold;
}

.seckill-countdown-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(0, 0, 0, 0.7);
  color: white;
  padding: 8px;
  text-align: center;
  font-size: 12px;
}

.mini-timer {
  font-family: monospace;
  font-weight: bold;
  margin-left: 4px;
}

.card-content {
  padding: 16px;
}

.product-name {
  font-size: 16px;
  color: #333;
  margin-bottom: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-desc {
  font-size: 12px;
  color: #999;
  margin-bottom: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.seckill-price .current {
  font-size: 24px;
  color: #ff4d4f;
  font-weight: bold;
  margin-right: 8px;
}

.seckill-price .original {
  font-size: 14px;
  color: #999;
  text-decoration: line-through;
}

.discount-tag {
  background: #fff0f0;
  color: #ff4d4f;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.stock-section {
  margin-bottom: 16px;
}

.stock-bar {
  height: 6px;
  background: #f0f0f0;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 6px;
}

.stock-progress {
  height: 100%;
  background: linear-gradient(90deg, #ff4d4f, #ff6b6b);
  border-radius: 3px;
  transition: width 0.3s;
}

.stock-text {
  font-size: 12px;
  color: #999;
}

.seckill-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: bold;
}

/* 秒杀规则 */
.seckill-rules {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px 40px;
  background: white;
  border-radius: 12px;
  padding: 24px;
}

.seckill-rules h2 {
  font-size: 18px;
  color: #333;
  margin-bottom: 16px;
}

.seckill-rules ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.seckill-rules li {
  font-size: 14px;
  color: #666;
  padding: 8px 0;
  padding-left: 24px;
  position: relative;
}

.seckill-rules li::before {
  content: '•';
  position: absolute;
  left: 8px;
  color: #ff4d4f;
}

/* 响应式 */
@media (max-width: 1024px) {
  .seckill-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 768px) {
  .seckill-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .seckill-title h1 {
    font-size: 28px;
  }
  
  .time-block {
    font-size: 20px;
    padding: 6px 8px;
    min-width: 36px;
  }
}
</style>
