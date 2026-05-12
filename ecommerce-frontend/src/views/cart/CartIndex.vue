<template>
  <div class="cart-page">
    <div class="cart-container">
      <div class="cart-header">
        <h1>我的购物车</h1>
        <span class="cart-count">共 {{ cartStore.totalCount }} 件商品</span>
      </div>
      
      <!-- 购物车列表 -->
      <div class="cart-list" v-loading="cartStore.loading">
        <div v-if="cartStore.items.length === 0" class="empty-cart">
          <el-empty description="购物车是空的">
            <el-button type="primary" @click="$router.push('/product/list')">
              去逛逛
            </el-button>
          </el-empty>
        </div>
        
        <div v-else>
          <!-- 购物车表格 -->
          <el-table :data="cartStore.items" style="width: 100%">
            <el-table-column width="50">
              <template #default="{ row }">
                <el-checkbox
                  :model-value="row.selected"
                  @change="handleSelectChange(row.product_id, $event)"
                />
              </template>
            </el-table-column>
            
            <el-table-column label="商品" min-width="300">
              <template #default="{ row }">
                <div class="product-cell" @click="$router.push(`/product/detail/${row.product_id}`)">
                  <el-image :src="row.image_url" class="product-image" />
                  <div class="product-info">
                    <span class="product-name">{{ row.product_name }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column label="单价" width="150">
              <template #default="{ row }">
                <span class="price">¥{{ formatPrice(row.price) }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="数量" width="180">
              <template #default="{ row }">
                <el-input-number
                  :model-value="row.count"
                  :min="1"
                  :max="99"
                  size="small"
                  @change="handleCountChange(row.product_id, $event)"
                />
              </template>
            </el-table-column>
            
            <el-table-column label="小计" width="150">
              <template #default="{ row }">
                <span class="subtotal">¥{{ formatPrice(row.price * row.count) }}</span>
              </template>
            </el-table-column>
            
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button type="danger" link @click="handleRemove(row.product_id)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          
          <!-- 底部操作栏 -->
          <div class="cart-footer">
            <div class="footer-left">
              <el-checkbox
                :model-value="cartStore.isAllSelected"
                @change="handleSelectAll"
              >
                全选
              </el-checkbox>
              <el-button type="text" @click="handleClearSelected" :disabled="cartStore.selectedCount === 0">
                清空已选
              </el-button>
            </div>
            
            <div class="footer-right">
              <div class="total-info">
                <span class="total-label">已选 {{ cartStore.selectedCount }} 件商品，总计：</span>
                <span class="total-price">¥{{ formatPrice(cartStore.selectedAmount) }}</span>
              </div>
              <el-button
                type="primary"
                size="large"
                :disabled="cartStore.selectedCount === 0"
                @click="handleCheckout"
              >
                去结算
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { useCartStore } from '@/stores/cart'
import { formatPrice } from '@/utils'

const router = useRouter()
const cartStore = useCartStore()

onMounted(() => {
  cartStore.fetchCart()
})

// 选择/取消选择单个商品
const handleSelectChange = async (productId: number, selected: boolean) => {
  await cartStore.toggleSelect(productId, selected)
}

// 全选/取消全选
const handleSelectAll = async (selected: boolean) => {
  await cartStore.toggleAllSelect(selected)
}

// 修改数量
const handleCountChange = async (productId: number, count: number) => {
  if (count === 0) {
    await handleRemove(productId)
  } else {
    await cartStore.updateCartItem(productId, count)
  }
}

// 删除单个商品
const handleRemove = async (productId: number) => {
  await ElMessageBox.confirm('确定要删除这件商品吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  await cartStore.removeCartItem(productId)
  ElMessage.success('已删除')
}

// 清空已选商品
const handleClearSelected = async () => {
  await ElMessageBox.confirm('确定要清空已选商品吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  for (const item of cartStore.selectedItems) {
    await cartStore.removeCartItem(item.product_id)
  }
  
  ElMessage.success('已清空')
}

// 去结算
const handleCheckout = () => {
  router.push('/order/confirm')
}
</script>

<style scoped>
.cart-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.cart-container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 0 20px;
}

.cart-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  background: white;
  padding: 24px;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.cart-header h1 {
  font-size: 24px;
  color: #333;
}

.cart-count {
  color: #999;
  font-size: 14px;
}

.cart-list {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.empty-cart {
  padding: 80px 0;
}

.product-cell {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.product-image {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  flex-shrink: 0;
}

.product-info {
  flex: 1;
}

.product-name {
  font-size: 14px;
  color: #333;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.price {
  color: #ff4d4f;
  font-weight: bold;
}

.subtotal {
  color: #ff4d4f;
  font-size: 16px;
  font-weight: bold;
}

.cart-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.footer-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.total-info {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.total-label {
  color: #666;
  font-size: 14px;
}

.total-price {
  color: #ff4d4f;
  font-size: 24px;
  font-weight: bold;
}

@media (max-width: 768px) {
  .cart-footer {
    flex-direction: column;
    gap: 16px;
  }
  
  .footer-right {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
