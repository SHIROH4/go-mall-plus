<template>
  <div class="order-confirm-page">
    <div class="confirm-container">
      <h1>确认订单</h1>

      <!-- 收货地址 -->
      <div class="section address-section">
        <h2>收货信息</h2>
        <div v-if="addresses.length === 0" class="no-address">
          <el-empty description="暂无收货地址" :image-size="60">
            <template #default>
              <el-button type="primary" @click="$router.push('/user/address')">
                添加收货地址
              </el-button>
            </template>
          </el-empty>
        </div>
        <div v-else class="address-list">
          <div
            v-for="addr in addresses"
            :key="addr.id"
            class="address-card"
            :class="{ selected: selectedAddressId === addr.id }"
            @click="selectedAddressId = addr.id"
          >
            <div class="address-info">
              <div class="address-header">
                <span class="receiver">{{ addr.receiver_name }}</span>
                <span class="phone">{{ addr.phone }}</span>
                <el-tag v-if="addr.is_default" type="success" size="small">默认</el-tag>
              </div>
              <p class="address-detail">
                {{ addr.province }} {{ addr.city }} {{ addr.district }} {{ addr.detail_address }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- 商品列表 -->
      <div class="section goods-section">
        <h2>商品清单</h2>
        <div class="goods-list">
          <div v-if="selectedItems.length === 0" class="empty-tip">
            暂无选中的商品
          </div>

          <div
            v-for="item in selectedItems"
            :key="item.product_id"
            class="goods-item"
          >
            <el-image :src="item.image_url" class="goods-image" />
            <div class="goods-info">
              <h3>{{ item.product_name }}</h3>
              <p class="goods-price">¥{{ formatPrice(item.price) }}</p>
            </div>
            <div class="goods-count">
              × {{ item.count }}
            </div>
            <div class="goods-subtotal">
              ¥{{ formatPrice(item.price * item.count) }}
            </div>
          </div>
        </div>
      </div>

      <!-- 订单总结 -->
      <div class="section summary-section">
        <h2>订单总结</h2>
        <div class="summary-row">
          <span class="label">商品件数：</span>
          <span class="value">{{ selectedCount }} 件</span>
        </div>
        <div class="summary-row">
          <span class="label">商品总额：</span>
          <span class="value">¥{{ formatPrice(selectedAmount) }}</span>
        </div>
        <div class="summary-row total">
          <span class="label">应付总额：</span>
          <span class="value price">¥{{ formatPrice(selectedAmount) }}</span>
        </div>
      </div>

      <!-- 提交按钮 -->
      <div class="submit-section">
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          :disabled="selectedItems.length === 0 || addresses.length === 0"
          @click="handleSubmit"
        >
          提交订单
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useCartStore } from '@/stores/cart'
import { getAddressList } from '@/api/address'
import { formatPrice } from '@/utils'
import type { CartItem, AddressItem } from '@/types'

const router = useRouter()
const cartStore = useCartStore()

const loading = ref(false)
const selectedItems = ref<CartItem[]>([])
const selectedCount = ref(0)
const selectedAmount = ref(0)
const addresses = ref<AddressItem[]>([])
const selectedAddressId = ref<number | null>(null)

onMounted(async () => {
  // 获取已选中的商品
  const data = await cartStore.fetchSelectedCart()
  selectedItems.value = data.items
  selectedCount.value = data.selectedCount
  selectedAmount.value = data.totalAmount

  // 获取收货地址列表
  try {
    const res = await getAddressList()
    addresses.value = res.addresses || []
    // 默认选择第一个或默认地址
    if (addresses.value.length > 0) {
      const defaultAddr = addresses.value.find(a => a.is_default)
      selectedAddressId.value = defaultAddr ? defaultAddr.id : addresses.value[0].id
    }
  } catch (error) {
    console.error('Failed to fetch addresses:', error)
  }
})

// 提交订单
const handleSubmit = async () => {
  if (selectedItems.value.length === 0) {
    ElMessage.warning('请选择要购买的商品')
    return
  }

  if (!selectedAddressId.value) {
    ElMessage.warning('请选择收货地址')
    return
  }

  loading.value = true

  try {
    const { createOrder } = await import('@/api/order')

    // 逐个创建订单（或者可以批量创建）
    const orderNoList: string[] = []

    for (const item of selectedItems.value) {
      const res = await createOrder({
        productId: item.product_id,
        count: item.count
      })
      orderNoList.push(res.orderNo)
    }

    // 跳转到支付页面（使用第一个订单号）
    ElMessage.success('订单创建成功')
    router.push(`/payment/${orderNoList[0]}`)
  } catch (error: any) {
    ElMessage.error(error.message || '订单创建失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.order-confirm-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.confirm-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 20px;
}

.confirm-container > h1 {
  font-size: 24px;
  color: #333;
  margin-bottom: 20px;
}

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

/* 收货地址 */
.no-address {
  text-align: center;
  padding: 20px;
}

.address-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.address-card {
  border: 2px solid #f0f0f0;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s;
}

.address-card:hover {
  border-color: #1890ff;
}

.address-card.selected {
  border-color: #1890ff;
  background: #f0f7ff;
}

.address-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.receiver {
  font-size: 16px;
  font-weight: bold;
  color: #333;
}

.phone {
  font-size: 14px;
  color: #666;
}

.address-detail {
  color: #666;
  font-size: 14px;
  line-height: 1.5;
}

/* 商品列表 */
.goods-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.goods-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
}

.goods-image {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  flex-shrink: 0;
}

.goods-info {
  flex: 1;
}

.goods-info h3 {
  font-size: 14px;
  color: #333;
  margin-bottom: 8px;
}

.goods-price {
  color: #ff4d4f;
  font-size: 14px;
}

.goods-count {
  color: #666;
  font-size: 14px;
}

.goods-subtotal {
  color: #ff4d4f;
  font-size: 16px;
  font-weight: bold;
  min-width: 80px;
  text-align: right;
}

.empty-tip {
  text-align: center;
  padding: 40px;
  color: #999;
}

/* 订单总结 */
.summary-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
}

.summary-row .label {
  color: #666;
}

.summary-row .value {
  color: #333;
}

.summary-row.total {
  border-top: 1px solid #f0f0f0;
  margin-top: 12px;
  padding-top: 16px;
}

.summary-row.total .label {
  font-size: 16px;
  color: #333;
  font-weight: bold;
}

.summary-row.total .price {
  font-size: 24px;
  color: #ff4d4f;
  font-weight: bold;
}

/* 提交按钮 */
.submit-section {
  text-align: right;
  padding: 20px 0;
}

.submit-section .el-button {
  width: 200px;
  height: 48px;
  font-size: 16px;
}
</style>
