<template>
  <div class="product-detail-page" v-loading="loading">
    <!-- 商品详情 -->
    <div class="detail-container" v-if="product">
      <!-- 商品图片 -->
      <div class="product-images">
        <div class="main-image">
          <el-image :src="product.image_url" fit="contain" />
        </div>
      </div>
      
      <!-- 商品信息 -->
      <div class="product-info">
        <h1 class="product-title">{{ product.name }}</h1>
        <p class="product-desc">{{ product.desc }}</p>
        
        <div class="price-info">
          <div class="current-price">
            <span class="label">价格</span>
            <span class="price">¥{{ formatPrice(product.price) }}</span>
          </div>
          <div class="stock-info">
            <span class="label">库存</span>
            <span class="value">{{ product.stock }} 件</span>
          </div>
          <div class="category-info">
            <span class="label">分类</span>
            <span class="value">{{ product.category_name || '未分类' }}</span>
          </div>
        </div>
        
        <div class="quantity-selector">
          <span class="label">数量</span>
          <el-input-number
            v-model="quantity"
            :min="1"
            :max="product.stock"
            :step="1"
          />
        </div>
        
        <div class="action-buttons">
          <el-button type="primary" size="large" @click="handleBuyNow">
            立即购买
          </el-button>
          <el-button size="large" @click="handleAddToCart">
            加入购物车
          </el-button>
        </div>
      </div>
    </div>
    
    <!-- 商品推荐 -->
    <div class="recommend-section" v-if="recommendations.length > 0">
      <h2>猜你喜欢</h2>
      <div class="recommend-grid">
        <div
          v-for="item in recommendations"
          :key="item.id"
          class="recommend-item"
          @click="$router.push(`/product/detail/${item.id}`)"
        >
          <div class="recommend-image">
            <el-image :src="item.image_url" fit="cover" />
          </div>
          <p class="recommend-name">{{ item.name }}</p>
          <p class="recommend-price">¥{{ formatPrice(item.price) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getProduct, listProductByPage } from '@/api/product'
import { useCartStore } from '@/stores/cart'
import { useUserStore } from '@/stores/user'
import { formatPrice } from '@/utils'
import type { ProductItem } from '@/types'

const route = useRoute()
const router = useRouter()
const cartStore = useCartStore()
const userStore = useUserStore()

const loading = ref(false)
const product = ref<ProductItem | null>(null)
const quantity = ref(1)
const recommendations = ref<ProductItem[]>([])

const productId = parseInt(route.params.id as string)

// 获取商品详情
const fetchProductDetail = async () => {
  loading.value = true
  try {
    const res = await getProduct(productId)
    product.value = res.product
  } catch (error) {
    console.error('Failed to fetch product:', error)
    ElMessage.error('商品不存在')
  } finally {
    loading.value = false
  }
}

// 获取推荐商品
const fetchRecommendations = async () => {
  try {
    const res = await listProductByPage({ page: 1, page_size: 6 })
    recommendations.value = (res.products || []).filter(p => p.id !== productId)
  } catch (error) {
    console.error('Failed to fetch recommendations:', error)
  }
}

// 添加到购物车
const handleAddToCart = async () => {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  
  if (!product.value) return
  
  const success = await cartStore.addToCart({
    product_id: product.value.id,
    product_name: product.value.name,
    price: product.value.price,
    image_url: product.value.image_url,
    count: quantity.value
  })
  
  if (success) {
    ElMessage.success('已加入购物车')
  }
}

// 立即购买
const handleBuyNow = () => {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  
  if (!product.value) return
  
  // 先添加到购物车
  handleAddToCart().then(() => {
    // 跳转到确认订单页面
    router.push('/order/confirm')
  })
}

onMounted(() => {
  fetchProductDetail()
  fetchRecommendations()
})
</script>

<style scoped>
.product-detail-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.detail-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 40px;
  background: white;
  border-radius: 12px;
  padding: 30px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

/* 商品图片 */
.product-images {
  width: 400px;
}

.main-image {
  width: 100%;
  height: 400px;
  background: #f5f5f5;
  border-radius: 8px;
  overflow: hidden;
}

.main-image .el-image {
  width: 100%;
  height: 100%;
}

/* 商品信息 */
.product-info {
  display: flex;
  flex-direction: column;
}

.product-title {
  font-size: 24px;
  color: #333;
  margin-bottom: 12px;
  line-height: 1.4;
}

.product-desc {
  font-size: 14px;
  color: #666;
  margin-bottom: 24px;
  line-height: 1.6;
}

.price-info {
  background: #f5f7ff;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 24px;
}

.current-price {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 12px;
}

.current-price .price {
  font-size: 32px;
  color: #ff4d4f;
  font-weight: bold;
}

.stock-info,
.category-info {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
}

.label {
  color: #999;
  font-size: 14px;
  width: 60px;
}

.value {
  color: #333;
  font-size: 14px;
}

.quantity-selector {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
}

.action-buttons {
  display: flex;
  gap: 16px;
}

.action-buttons .el-button {
  flex: 1;
  height: 48px;
  font-size: 16px;
}

/* 推荐商品 */
.recommend-section {
  max-width: 1200px;
  margin: 30px auto;
  padding: 0 20px;
}

.recommend-section h2 {
  font-size: 20px;
  color: #333;
  margin-bottom: 20px;
}

.recommend-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 16px;
}

.recommend-item {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.recommend-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(24, 144, 255, 0.2);
}

.recommend-image {
  height: 140px;
  background: #f5f5f5;
}

.recommend-name {
  padding: 8px;
  font-size: 13px;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recommend-price {
  padding: 0 8px 8px;
  font-size: 14px;
  color: #ff4d4f;
  font-weight: bold;
}

/* 响应式 */
@media (max-width: 1024px) {
  .detail-container {
    grid-template-columns: 1fr;
  }
  
  .product-images {
    width: 100%;
  }
  
  .recommend-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (max-width: 768px) {
  .recommend-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .product-title {
    font-size: 20px;
  }
  
  .current-price .price {
    font-size: 24px;
  }
}
</style>
