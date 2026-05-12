<template>
  <div class="order-list-page">
    <div class="list-container">
      <div class="list-header">
        <h1>我的订单</h1>
      </div>
      
      <!-- 状态筛选 -->
      <div class="filter-tabs">
        <el-tabs v-model="activeStatus" @tab-change="handleStatusChange">
          <el-tab-pane label="全部" :name="-1" />
          <el-tab-pane label="待支付" :name="0" />
          <el-tab-pane label="已支付" :name="1" />
          <el-tab-pane label="已取消" :name="2" />
          <el-tab-pane label="已超时" :name="3" />
        </el-tabs>
      </div>
      
      <!-- 订单列表 -->
      <div class="order-list" v-loading="loading">
        <div v-if="orders.length === 0" class="empty-order">
          <el-empty description="暂无订单">
            <el-button type="primary" @click="$router.push('/product/list')">
              去购物
            </el-button>
          </el-empty>
        </div>
        
        <div v-else>
          <div
            v-for="order in orders"
            :key="order.order_no"
            class="order-card"
          >
            <div class="order-header">
              <span class="order-no">订单号：{{ order.order_no }}</span>
              <span class="order-time">{{ formatDateTime(order.create_time) }}</span>
              <span class="order-status" :class="getStatusClass(order.status)">
                {{ order.status_text }}
              </span>
            </div>
            
            <div class="order-body" @click="$router.push(`/order/detail/${order.order_no}`)">
              <div class="product-info">
                <span class="product-name">{{ order.product_name }}</span>
                <span class="product-count">× {{ order.count }}</span>
              </div>
              <div class="order-amount">
                <span class="amount-label">实付金额：</span>
                <span class="amount-value">¥{{ formatPrice(order.total_amount) }}</span>
              </div>
            </div>
            
            <div class="order-footer">
              <div class="footer-left">
                <el-button
                  v-if="order.status === 0"
                  type="danger"
                  size="small"
                  @click="$router.push(`/payment/${order.order_no}`)"
                >
                  立即支付
                </el-button>
                <el-button
                  v-if="order.status === 0"
                  size="small"
                  @click="handleCancel(order.order_no)"
                >
                  取消订单
                </el-button>
              </div>
              <div class="footer-right">
                <el-button type="text" @click="$router.push(`/order/detail/${order.order_no}`)">
                  查看详情
                </el-button>
              </div>
            </div>
          </div>
        </div>
        
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listOrder, cancelOrder } from '@/api/order'
import { formatPrice, formatDateTime } from '@/utils'
import type { OrderItem } from '@/types'

const loading = ref(false)
const orders = ref<OrderItem[]>([])
const activeStatus = ref(-1)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const getStatusClass = (status: number) => {
  const classMap: Record<number, string> = {
    0: 'status-pending',
    1: 'status-paid',
    2: 'status-cancelled',
    3: 'status-expired'
  }
  return classMap[status] || ''
}

const fetchOrders = async () => {
  loading.value = true
  try {
    const res = await listOrder({
      page: page.value,
      page_size: pageSize.value,
      status: activeStatus.value === -1 ? undefined : activeStatus.value
    })
    
    orders.value = res.orders || []
    total.value = res.total || 0
  } catch (error) {
    console.error('Failed to fetch orders:', error)
  } finally {
    loading.value = false
  }
}

const handleStatusChange = () => {
  page.value = 1
  fetchOrders()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  page.value = 1
  fetchOrders()
}

const handlePageChange = (val: number) => {
  page.value = val
  fetchOrders()
}

const handleCancel = async (orderNo: string) => {
  await ElMessageBox.confirm('确定要取消这个订单吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  try {
    await cancelOrder({ order_no: orderNo })
    ElMessage.success('订单已取消')
    fetchOrders()
  } catch (error: any) {
    ElMessage.error(error.message || '取消失败')
  }
}

onMounted(() => {
  fetchOrders()
})
</script>

<style scoped>
.order-list-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.list-container {
  max-width: 900px;
  margin: 0 auto;
  padding: 0 20px;
}

.list-header {
  margin-bottom: 20px;
}

.list-header h1 {
  font-size: 24px;
  color: #333;
}

.filter-tabs {
  background: white;
  border-radius: 12px;
  padding: 0 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.order-list {
  margin-top: 20px;
}

.empty-order {
  background: white;
  padding: 80px 0;
  border-radius: 12px;
}

.order-card {
  background: white;
  border-radius: 12px;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.order-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
}

.order-no {
  font-size: 14px;
  color: #333;
  font-weight: bold;
}

.order-time {
  font-size: 13px;
  color: #999;
}

.order-status {
  margin-left: auto;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
}

.status-pending {
  background: #fff7e6;
  color: #fa8c16;
}

.status-paid {
  background: #f6ffed;
  color: #52c41a;
}

.status-cancelled {
  background: #f5f5f5;
  color: #999;
}

.status-expired {
  background: #f5f5f5;
  color: #ff4d4f;
}

.order-body {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  cursor: pointer;
  transition: background 0.3s;
}

.order-body:hover {
  background: #fafafa;
}

.product-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.product-name {
  font-size: 14px;
  color: #333;
}

.product-count {
  color: #999;
  font-size: 14px;
}

.order-amount {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.amount-label {
  font-size: 14px;
  color: #666;
}

.amount-value {
  font-size: 18px;
  color: #ff4d4f;
  font-weight: bold;
}

.order-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid #f0f0f0;
}

.footer-left {
  display: flex;
  gap: 12px;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 20px 0;
}
</style>
