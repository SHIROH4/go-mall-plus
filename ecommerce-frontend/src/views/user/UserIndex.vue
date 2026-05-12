<template>
  <div class="user-center-page">
    <div class="user-container">
      <!-- 用户信息卡片 -->
      <div class="user-info-card">
        <div class="avatar-section">
          <div class="avatar">
            <el-icon :size="48"><UserFilled /></el-icon>
          </div>
          <div class="user-detail">
            <h2>{{ userStore.userInfo?.username || '用户' }}</h2>
            <p>欢迎回来</p>
          </div>
        </div>
        <div class="user-stats">
          <div class="stat-item">
            <span class="stat-value">{{ orderStats.pending }}</span>
            <span class="stat-label">待支付</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ orderStats.paid }}</span>
            <span class="stat-label">已完成</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ cartStore.totalCount }}</span>
            <span class="stat-label">购物车</span>
          </div>
        </div>
      </div>
      
      <!-- 功能菜单 -->
      <div class="menu-section">
        <h2>我的服务</h2>
        <div class="menu-grid">
          <div class="menu-item" @click="$router.push('/order/list')">
            <div class="menu-icon" style="background: #e6f7ff;">
              <el-icon :size="28" color="#1890ff"><List /></el-icon>
            </div>
            <span>我的订单</span>
          </div>
          
          <div class="menu-item" @click="$router.push('/cart')">
            <div class="menu-icon" style="background: #fff7e6;">
              <el-icon :size="28" color="#fa8c16"><ShoppingCart /></el-icon>
            </div>
            <span>购物车</span>
          </div>
          
          <div class="menu-item" @click="$router.push('/user/address')">
            <div class="menu-icon" style="background: #f6ffed;">
              <el-icon :size="28" color="#52c41a"><Location /></el-icon>
            </div>
            <span>收货地址</span>
          </div>
          
          <div class="menu-item" @click="$router.push('/user')">
            <div class="menu-icon" style="background: #fff0f0;">
              <el-icon :size="28" color="#ff4d4f"><Star /></el-icon>
            </div>
            <span>我的收藏</span>
          </div>
          
          <div class="menu-item" @click="$router.push('/user')">
            <div class="menu-icon" style="background: #f9f0ff;">
              <el-icon :size="28" color="#722ed1"><Setting /></el-icon>
            </div>
            <span>账户设置</span>
          </div>
          
          <div class="menu-item" @click="handleLogout">
            <div class="menu-icon" style="background: #f5f5f5;">
              <el-icon :size="28" color="#999"><SwitchButton /></el-icon>
            </div>
            <span>退出登录</span>
          </div>
        </div>
      </div>
      
      <!-- 快捷操作 -->
      <div class="quick-action-section">
        <h2>快捷操作</h2>
        <div class="action-list">
          <router-link to="/product/list" class="action-item">
            <el-icon :size="20"><ShoppingBag /></el-icon>
            <span>继续购物</span>
          </router-link>
          <router-link to="/seckill" class="action-item">
            <el-icon :size="20"><Lightning /></el-icon>
            <span>秒杀专区</span>
          </router-link>
          <router-link to="/order/list" class="action-item">
            <el-icon :size="20"><Document /></el-icon>
            <span>查看订单</span>
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UserFilled, List, ShoppingCart, Location, Star, Setting, SwitchButton, ShoppingBag, Lightning, Document } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { useCartStore } from '@/stores/cart'
import { listOrder } from '@/api/order'

const router = useRouter()
const userStore = useUserStore()
const cartStore = useCartStore()

const orderStats = ref({
  pending: 0,
  paid: 0
})

// 获取订单统计
const fetchOrderStats = async () => {
  try {
    const res = await listOrder({ page: 1, page_size: 100 })
    
    orderStats.value.pending = res.orders?.filter(o => o.status === 0).length || 0
    orderStats.value.paid = res.orders?.filter(o => o.status === 1).length || 0
  } catch (error) {
    console.error('Failed to fetch order stats:', error)
  }
}

// 退出登录
const handleLogout = async () => {
  await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  
  await userStore.logout()
  cartStore.clearCart()
  ElMessage.success('已退出登录')
  router.push('/home')
}

onMounted(() => {
  fetchOrderStats()
  cartStore.fetchCart()
})
</script>

<style scoped>
.user-center-page {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 20px 0;
}

.user-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 20px;
}

/* 用户信息卡片 */
.user-info-card {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border-radius: 16px;
  padding: 30px;
  color: white;
  margin-bottom: 20px;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.3);
}

.avatar-section {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 24px;
}

.avatar {
  width: 80px;
  height: 80px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.user-detail h2 {
  font-size: 24px;
  margin-bottom: 4px;
}

.user-detail p {
  font-size: 14px;
  opacity: 0.9;
}

.user-stats {
  display: flex;
  justify-content: space-around;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.2);
}

.stat-item {
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 28px;
  font-weight: bold;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 13px;
  opacity: 0.9;
}

/* 功能菜单 */
.menu-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.menu-section h2 {
  font-size: 16px;
  color: #333;
  margin-bottom: 16px;
}

.menu-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.menu-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px 16px;
  background: #fafafa;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
}

.menu-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.menu-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.menu-item span {
  font-size: 14px;
  color: #333;
}

/* 快捷操作 */
.quick-action-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.quick-action-section h2 {
  font-size: 16px;
  color: #333;
  margin-bottom: 16px;
}

.action-list {
  display: flex;
  gap: 16px;
}

.action-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  background: #f5f5f5;
  border-radius: 8px;
  color: #666;
  text-decoration: none;
  font-size: 14px;
  transition: all 0.3s;
}

.action-item:hover {
  background: #1890ff;
  color: white;
}

/* 响应式 */
@media (max-width: 768px) {
  .menu-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .action-list {
    flex-direction: column;
  }
}
</style>
