<template>
  <div class="main-layout">
    <!-- 顶部导航栏 -->
    <header class="header">
      <div class="header-container">
        <div class="logo" @click="$router.push('/home')">
          <el-icon :size="28" color="#1890ff"><ShoppingBag /></el-icon>
          <span class="logo-text">电商商城</span>
        </div>
        
        <nav class="nav-menu">
          <router-link to="/home" class="nav-item" :class="{ active: $route.path === '/home' }">
            <el-icon><HomeFilled /></el-icon>
            <span>首页</span>
          </router-link>
          <router-link to="/product/list" class="nav-item" :class="{ active: $route.path === '/product/list' }">
            <el-icon><Goods /></el-icon>
            <span>全部商品</span>
          </router-link>
          <router-link to="/seckill" class="nav-item" :class="{ active: $route.path === '/seckill' }">
            <el-icon><Lightning /></el-icon>
            <span>秒杀专区</span>
          </router-link>
        </nav>
        
        <div class="header-right">
          <!-- 搜索框 -->
          <div class="search-box">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索商品..."
              @keyup.enter="handleSearch"
              class="search-input"
            >
              <template #append>
                <el-button :icon="Search" @click="handleSearch" />
              </template>
            </el-input>
          </div>
          
          <!-- 购物车 -->
          <router-link to="/cart" class="cart-btn">
            <el-badge :value="cartStore.totalCount" :hidden="cartStore.totalCount === 0" :max="99">
              <el-icon :size="22"><ShoppingCart /></el-icon>
            </el-badge>
            <span>购物车</span>
          </router-link>
          
          <!-- 用户菜单 -->
          <div class="user-menu" v-if="userStore.isLoggedIn">
            <el-dropdown @command="handleUserCommand">
              <span class="user-info">
                <el-icon><UserFilled /></el-icon>
                <span>{{ userStore.userInfo?.username || '用户' }}</span>
                <el-icon><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="order">
                    <el-icon><List /></el-icon>
                    我的订单
                  </el-dropdown-item>
                  <el-dropdown-item command="user">
                    <el-icon><User /></el-icon>
                    个人中心
                  </el-dropdown-item>
                  <el-dropdown-item command="logout" divided>
                    <el-icon><SwitchButton /></el-icon>
                    退出登录
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          
          <!-- 登录/注册 -->
          <div class="auth-btns" v-else>
            <router-link to="/login" class="login-btn">登录</router-link>
            <router-link to="/register" class="register-btn">注册</router-link>
          </div>
        </div>
      </div>
    </header>
    
    <!-- 主内容区 -->
    <main class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
    
    <!-- 底部 -->
    <footer class="footer">
      <div class="footer-content">
        <div class="footer-info">
          <p>电商商城 - 优质商品精选</p>
          <p class="copyright">© 2024 All Rights Reserved</p>
        </div>
        <div class="footer-links">
          <a href="#">关于我们</a>
          <a href="#">联系我们</a>
          <a href="#">帮助中心</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, HomeFilled, Goods, Lightning, ShoppingCart, UserFilled, ArrowDown, List, User, SwitchButton, ShoppingBag } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { useCartStore } from '@/stores/cart'

const router = useRouter()
const userStore = useUserStore()
const cartStore = useCartStore()

const searchKeyword = ref('')

onMounted(() => {
  // 初始化用户状态
  userStore.init()
  
  // 如果已登录，获取购物车数据
  if (userStore.isLoggedIn) {
    cartStore.fetchCart()
  }
})

const handleSearch = () => {
  if (searchKeyword.value.trim()) {
    router.push({
      path: '/product/list',
      query: { keyword: searchKeyword.value }
    })
  }
}

const handleUserCommand = async (command: string) => {
  switch (command) {
    case 'order':
      router.push('/order/list')
      break
    case 'user':
      router.push('/user')
      break
    case 'logout':
      await userStore.logout()
      cartStore.clearCart()
      router.push('/home')
      break
  }
}
</script>

<style scoped>
.main-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* 头部导航 */
.header {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
  position: sticky;
  top: 0;
  z-index: 1000;
}

.header-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: white;
}

.logo-text {
  font-size: 20px;
  font-weight: bold;
}

.nav-menu {
  display: flex;
  gap: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  color: rgba(255, 255, 255, 0.85);
  text-decoration: none;
  border-radius: 4px;
  transition: all 0.3s;
}

.nav-item:hover,
.nav-item.active {
  background: rgba(255, 255, 255, 0.15);
  color: white;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

.search-box {
  width: 280px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 4px 0 0 4px;
}

.cart-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  color: white;
  text-decoration: none;
  padding: 8px 12px;
  border-radius: 4px;
  transition: all 0.3s;
}

.cart-btn:hover {
  background: rgba(255, 255, 255, 0.15);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 6px;
  color: white;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 4px;
  transition: all 0.3s;
}

.user-info:hover {
  background: rgba(255, 255, 255, 0.15);
}

.auth-btns {
  display: flex;
  gap: 8px;
}

.login-btn,
.register-btn {
  color: white;
  text-decoration: none;
  padding: 8px 16px;
  border-radius: 4px;
  transition: all 0.3s;
}

.login-btn:hover,
.register-btn:hover {
  background: rgba(255, 255, 255, 0.15);
}

/* 主内容 */
.main-content {
  flex: 1;
  background: #f5f5f5;
}

/* 底部 */
.footer {
  background: #001529;
  color: rgba(255, 255, 255, 0.65);
  padding: 30px 20px;
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-info p {
  margin: 4px 0;
}

.copyright {
  font-size: 12px;
}

.footer-links {
  display: flex;
  gap: 24px;
}

.footer-links a {
  color: rgba(255, 255, 255, 0.65);
  text-decoration: none;
  transition: color 0.3s;
}

.footer-links a:hover {
  color: #1890ff;
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .header-container {
    padding: 0 10px;
  }
  
  .logo-text {
    display: none;
  }
  
  .nav-menu {
    display: none;
  }
  
  .search-box {
    width: 160px;
  }
  
  .cart-btn span {
    display: none;
  }
}
</style>
