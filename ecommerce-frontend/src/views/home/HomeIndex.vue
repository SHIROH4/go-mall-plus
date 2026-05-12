<template>
  <div class="home-page">
    <!-- 轮播图 -->
    <div class="banner-section">
      <el-carousel height="400px" indicator-position="outside" trigger="click">
        <el-carousel-item v-for="(banner, index) in banners" :key="index">
          <div class="banner-item" :style="{ background: banner.bg }">
            <div class="banner-content">
              <h2>{{ banner.title }}</h2>
              <p>{{ banner.subtitle }}</p>
              <el-button type="primary" size="large" @click="$router.push(banner.link)">
                {{ banner.btnText }}
              </el-button>
            </div>
          </div>
        </el-carousel-item>
      </el-carousel>
    </div>
    
    <!-- 分类导航 -->
    <div class="category-section">
      <div class="section-title">
        <h2>热门分类</h2>
        <router-link to="/product/list" class="more-link">查看全部 →</router-link>
      </div>
      <div class="category-list">
        <div
          v-for="category in categories"
          :key="category.id"
          class="category-item"
          @click="goToCategory(category.id)"
        >
          <div class="category-icon">
            <el-icon :size="36"><component :is="category.icon" /></el-icon>
          </div>
          <span>{{ category.name }}</span>
        </div>
      </div>
    </div>
    
    <!-- 秒杀专区 -->
    <div class="seckill-section">
      <div class="section-title">
        <div class="title-left">
          <el-icon :size="28" color="#ff4d4f"><Lightning /></el-icon>
          <h2>限时秒杀</h2>
        </div>
        <router-link to="/seckill" class="more-link">更多秒杀 →</router-link>
      </div>
      <div class="seckill-countdown">
        <span class="countdown-label">距离结束</span>
        <div class="countdown-timer">
          <span class="time-box">{{ countdownHours }}</span>
          <span class="time-separator">:</span>
          <span class="time-box">{{ countdownMinutes }}</span>
          <span class="time-separator">:</span>
          <span class="time-box">{{ countdownSeconds }}</span>
        </div>
      </div>
      <div class="seckill-products">
        <div
          v-for="product in seckillProducts"
          :key="product.id"
          class="seckill-product-card"
          @click="$router.push(`/product/detail/${product.id}`)"
        >
          <div class="product-image">
            <el-image :src="product.image_url" fit="cover" />
            <div class="seckill-tag">限时特惠</div>
          </div>
          <div class="product-info">
            <h3>{{ product.name }}</h3>
            <p class="product-desc">{{ product.desc }}</p>
            <div class="price-row">
              <span class="seckill-price">¥{{ formatPrice(product.price) }}</span>
              <span class="original-price">¥{{ formatPrice(product.price * 1.5) }}</span>
            </div>
          </div>
        </div>
        <div v-if="seckillProducts.length === 0" class="empty-tip">
          暂无秒杀商品
        </div>
      </div>
    </div>
    
    <!-- 热门商品 -->
    <div class="products-section">
      <div class="section-title">
        <h2>热门推荐</h2>
        <router-link to="/product/list" class="more-link">查看全部 →</router-link>
      </div>
      <div class="product-grid">
        <div
          v-for="product in products"
          :key="product.id"
          class="product-card"
          @click="$router.push(`/product/detail/${product.id}`)"
        >
          <div class="product-image">
            <el-image :src="product.image_url" fit="cover" />
          </div>
          <div class="product-info">
            <h3>{{ product.name }}</h3>
            <p class="product-desc">{{ product.desc }}</p>
            <div class="price-row">
              <span class="price">¥{{ formatPrice(product.price) }}</span>
              <span class="stock">库存: {{ product.stock }}</span>
            </div>
          </div>
          <div class="product-actions">
            <el-button type="primary" size="small" @click.stop="handleAddToCart(product)">
              加入购物车
            </el-button>
          </div>
        </div>
        <div v-if="products.length === 0" class="empty-tip">
          暂无商品
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lightning, Foods, Clothes, Digital, Book, Gift } from '@element-plus/icons-vue'
import { listProduct } from '@/api/product'
import { useCartStore } from '@/stores/cart'
import { useUserStore } from '@/stores/user'
import { formatPrice } from '@/utils'
import type { ProductItem } from '@/types'

const router = useRouter()
const cartStore = useCartStore()
const userStore = useUserStore()

const products = ref<ProductItem[]>([])
const seckillProducts = ref<ProductItem[]>([])
const loading = ref(false)

// 轮播图数据
const banners = [
  {
    title: '新品上市',
    subtitle: '精选优质商品，品质保障',
    btnText: '立即选购',
    link: '/product/list',
    bg: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
  },
  {
    title: '限时秒杀',
    subtitle: '每日10点、20点准时开抢',
    btnText: '去秒杀',
    link: '/seckill',
    bg: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
  },
  {
    title: '品质保障',
    subtitle: '7天无理由退换货',
    btnText: '了解更多',
    link: '/home',
    bg: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)'
  }
]

// 分类数据
const categories = [
  { id: 1, name: '数码电子', icon: Digital },
  { id: 2, name: '服装鞋包', icon: Clothes },
  { id: 3, name: '美食生鲜', icon: Foods },
  { id: 4, name: '图书音像', icon: Book },
  { id: 5, name: '礼品箱包', icon: Gift }
]

// 秒杀倒计时
const countdownHours = ref('00')
const countdownMinutes = ref('00')
const countdownSeconds = ref('00')
let countdownTimer: number | null = null

const updateCountdown = () => {
  const now = new Date()
  const nextSeckill = new Date()
  
  // 设置为下一个10点或20点
  if (now.getHours() < 10) {
    nextSeckill.setHours(10, 0, 0, 0)
  } else if (now.getHours() < 20) {
    nextSeckill.setHours(20, 0, 0, 0)
  } else {
    nextSeckill.setDate(nextSeckill.getDate() + 1)
    nextSeckill.setHours(10, 0, 0, 0)
  }
  
  const diff = Math.max(0, nextSeckill.getTime() - now.getTime())
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)
  
  countdownHours.value = hours.toString().padStart(2, '0')
  countdownMinutes.value = minutes.toString().padStart(2, '0')
  countdownSeconds.value = seconds.toString().padStart(2, '0')
}

const fetchProducts = async () => {
  loading.value = true
  try {
    const res = await listProduct()
    products.value = res.products || []
    // 秒杀商品取前4个（实际项目中应该有专门的秒杀接口）
    seckillProducts.value = products.value.slice(0, 4)
  } catch (error) {
    console.error('Failed to fetch products:', error)
  } finally {
    loading.value = false
  }
}

const goToCategory = (categoryId: number) => {
  router.push({ path: '/product/list', query: { category_id: categoryId.toString() } })
}

const handleAddToCart = async (product: ProductItem) => {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  
  const success = await cartStore.addToCart({
    product_id: product.id,
    product_name: product.name,
    price: product.price,
    image_url: product.image_url,
    count: 1
  })
  
  if (success) {
    ElMessage.success('已加入购物车')
  }
}

onMounted(() => {
  fetchProducts()
  updateCountdown()
  countdownTimer = window.setInterval(updateCountdown, 1000)
})

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
  }
})
</script>

<style scoped>
.home-page {
  background: #f5f5f5;
}

/* 轮播图 */
.banner-section {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 20px 0;
}

.banner-item {
  height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  text-align: center;
}

.banner-content h2 {
  font-size: 42px;
  margin-bottom: 16px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.banner-content p {
  font-size: 18px;
  margin-bottom: 24px;
  opacity: 0.9;
}

/* 分类导航 */
.category-section {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px;
}

.section-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-title h2 {
  font-size: 24px;
  color: #333;
}

.title-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.more-link {
  color: #1890ff;
  text-decoration: none;
  font-size: 14px;
}

.more-link:hover {
  text-decoration: underline;
}

.category-list {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
}

.category-item {
  background: white;
  border-radius: 12px;
  padding: 24px 16px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.category-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(24, 144, 255, 0.2);
}

.category-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 12px;
  background: linear-gradient(135deg, #e6f7ff 0%, #bae7ff 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #1890ff;
}

.category-item span {
  font-size: 14px;
  color: #333;
}

/* 秒杀专区 */
.seckill-section {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px;
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.seckill-countdown {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0;
}

.countdown-label {
  font-size: 14px;
  color: #666;
}

.countdown-timer {
  display: flex;
  align-items: center;
  gap: 4px;
}

.time-box {
  background: #ff4d4f;
  color: white;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 16px;
  font-weight: bold;
  font-family: monospace;
}

.time-separator {
  font-size: 16px;
  font-weight: bold;
  color: #ff4d4f;
}

.seckill-products {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.seckill-product-card {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid #f0f0f0;
}

.seckill-product-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(255, 77, 79, 0.2);
}

.seckill-product-card .product-image {
  position: relative;
  height: 180px;
}

.seckill-tag {
  position: absolute;
  top: 8px;
  left: 8px;
  background: #ff4d4f;
  color: white;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.seckill-product-card .product-info {
  padding: 12px;
}

.seckill-product-card h3 {
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.seckill-product-card .product-desc {
  font-size: 12px;
  color: #999;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.seckill-price {
  color: #ff4d4f;
  font-size: 18px;
  font-weight: bold;
}

.original-price {
  color: #999;
  font-size: 12px;
  text-decoration: line-through;
}

/* 热门商品 */
.products-section {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px 40px;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
}

.product-card {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.product-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(24, 144, 255, 0.2);
}

.product-card .product-image {
  height: 180px;
  background: #f5f5f5;
}

.product-info {
  padding: 12px;
}

.product-info h3 {
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-desc {
  font-size: 12px;
  color: #999;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price {
  color: #ff4d4f;
  font-size: 18px;
  font-weight: bold;
}

.stock {
  color: #999;
  font-size: 12px;
  margin-left: auto;
}

.product-actions {
  padding: 0 12px 12px;
}

/* 空状态 */
.empty-tip {
  grid-column: 1 / -1;
  text-align: center;
  padding: 40px;
  color: #999;
  font-size: 14px;
}

/* 响应式 */
@media (max-width: 1024px) {
  .category-list {
    grid-template-columns: repeat(3, 1fr);
  }
  
  .seckill-products,
  .product-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 768px) {
  .category-list {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .seckill-products,
  .product-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .banner-content h2 {
    font-size: 28px;
  }
  
  .banner-content p {
    font-size: 14px;
  }
}
</style>
