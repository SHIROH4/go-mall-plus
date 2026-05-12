<template>
  <div class="product-list-page">
    <!-- 筛选区域 -->
    <div class="filter-section">
      <div class="filter-container">
        <div class="filter-row">
          <div class="filter-item">
            <span class="filter-label">分类:</span>
            <div class="category-tags">
              <el-tag
                :type="selectedCategory === null ? 'primary' : 'info'"
                class="category-tag"
                @click="selectedCategory = null"
              >
                全部
              </el-tag>
              <el-tag
                v-for="cat in categories"
                :key="cat.id"
                :type="selectedCategory === cat.id ? 'primary' : 'info'"
                class="category-tag"
                @click="selectedCategory = cat.id"
              >
                {{ cat.name }}
              </el-tag>
            </div>
          </div>
        </div>
        
        <div class="filter-row">
          <div class="filter-item">
            <span class="filter-label">搜索:</span>
            <el-input
              v-model="keyword"
              placeholder="输入关键词搜索"
              clearable
              style="width: 240px;"
              @keyup.enter="handleSearch"
            >
              <template #append>
                <el-button :icon="Search" @click="handleSearch" />
              </template>
            </el-input>
          </div>
          
          <div class="filter-item">
            <span class="filter-label">排序:</span>
            <el-select v-model="sortBy" style="width: 140px;">
              <el-option label="综合推荐" value="default" />
              <el-option label="价格从低到高" value="price_asc" />
              <el-option label="价格从高到低" value="price_desc" />
            </el-select>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 商品列表 -->
    <div class="product-list-container">
      <div class="product-grid" v-loading="loading">
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
            <div class="category-badge" v-if="product.category_name">
              {{ product.category_name }}
            </div>
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
      </div>
      
      <!-- 空状态 -->
      <el-empty v-if="!loading && products.length === 0" description="暂无商品" />
      
      <!-- 分页 -->
      <div class="pagination-container" v-if="total > 0">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { listProductByPage, getCategories } from '@/api/product'
import { useCartStore } from '@/stores/cart'
import { useUserStore } from '@/stores/user'
import { formatPrice } from '@/utils'
import type { ProductItem, CategoryItem } from '@/types'

const route = useRoute()
const router = useRouter()
const cartStore = useCartStore()
const userStore = useUserStore()

const loading = ref(false)
const products = ref<ProductItem[]>([])
const categories = ref<CategoryItem[]>([])
const selectedCategory = ref<number | null>(null)
const keyword = ref('')
const sortBy = ref('default')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 获取分类
const fetchCategories = async () => {
  try {
    const res = await getCategories()
    categories.value = res.categories || []
  } catch (error) {
    console.error('Failed to fetch categories:', error)
  }
}

// 获取商品列表
const fetchProducts = async () => {
  loading.value = true
  try {
    const res = await listProductByPage({
      category_id: selectedCategory.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      page_size: pageSize.value
    })
    
    products.value = res.products || []
    total.value = res.total || 0
    
    // 根据排序方式排序
    if (sortBy.value === 'price_asc') {
      products.value.sort((a, b) => a.price - b.price)
    } else if (sortBy.value === 'price_desc') {
      products.value.sort((a, b) => b.price - a.price)
    }
  } catch (error) {
    console.error('Failed to fetch products:', error)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  page.value = 1
  fetchProducts()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  page.value = 1
  fetchProducts()
}

const handlePageChange = (val: number) => {
  page.value = val
  fetchProducts()
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

// 监听分类变化
watch(selectedCategory, () => {
  page.value = 1
  fetchProducts()
})

// 监听排序变化
watch(sortBy, () => {
  // 排序变化时重新排序当前列表
  if (sortBy.value === 'price_asc') {
    products.value.sort((a, b) => a.price - b.price)
  } else if (sortBy.value === 'price_desc') {
    products.value.sort((a, b) => b.price - a.price)
  } else {
    // 默认排序，重新获取数据
    fetchProducts()
  }
})

onMounted(() => {
  // 从 URL 参数初始化
  if (route.query.category_id) {
    selectedCategory.value = parseInt(route.query.category_id as string)
  }
  if (route.query.keyword) {
    keyword.value = route.query.keyword as string
  }
  
  fetchCategories()
  fetchProducts()
})
</script>

<style scoped>
.product-list-page {
  min-height: 100vh;
  background: #f5f5f5;
}

/* 筛选区域 */
.filter-section {
  background: white;
  padding: 20px 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.filter-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  margin-bottom: 16px;
}

.filter-row:last-child {
  margin-bottom: 0;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-label {
  font-size: 14px;
  color: #666;
  white-space: nowrap;
}

.category-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.category-tag {
  cursor: pointer;
  transition: all 0.3s;
}

.category-tag:hover {
  transform: scale(1.05);
}

/* 商品列表 */
.product-list-container {
  max-width: 1200px;
  margin: 20px auto;
  padding: 0 20px;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.product-card {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
}

.product-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(24, 144, 255, 0.2);
}

.product-image {
  height: 180px;
  background: #f5f5f5;
}

.product-image .el-image {
  width: 100%;
  height: 100%;
}

.product-info {
  padding: 12px;
  flex: 1;
}

.product-info h3 {
  font-size: 14px;
  color: #333;
  margin-bottom: 6px;
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

.category-badge {
  display: inline-block;
  background: #e6f7ff;
  color: #1890ff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  margin-bottom: 8px;
}

.price-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.price {
  color: #ff4d4f;
  font-size: 18px;
  font-weight: bold;
}

.stock {
  color: #999;
  font-size: 12px;
}

.product-actions {
  padding: 0 12px 12px;
}

/* 分页 */
.pagination-container {
  display: flex;
  justify-content: center;
  padding: 20px 0;
}

/* 响应式 */
@media (max-width: 1024px) {
  .product-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (max-width: 768px) {
  .product-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .filter-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
