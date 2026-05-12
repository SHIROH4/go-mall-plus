<template>
  <div class="simple-home">
    <div class="container">
      <header class="header">
        <h1>电商商城</h1>
        <p>Vue3 + Element Plus 前端示例</p>
      </header>
      
      <div class="content">
        <div class="status-card">
          <h2>当前状态</h2>
          <div class="status-item">
            <span>前端服务:</span>
            <el-tag type="success">运行中 (端口 3001)</el-tag>
          </div>
          <div class="status-item">
            <span>后端服务:</span>
            <el-tag :type="backendConnected ? 'success' : 'danger'">
              {{ backendConnected ? '已连接' : '未启动' }}
            </el-tag>
          </div>
          <div class="status-item">
            <span>用户登录:</span>
            <el-tag :type="userStore.isLoggedIn ? 'success' : 'info'">
              {{ userStore.isLoggedIn ? '已登录' : '未登录' }}
            </el-tag>
          </div>
        </div>
        
        <div class="nav-card">
          <h2>页面导航</h2>
          <div class="nav-grid">
            <router-link to="/home" class="nav-item">
              <el-icon :size="32"><HomeFilled /></el-icon>
              <span>首页</span>
            </router-link>
            <router-link to="/product/list" class="nav-item">
              <el-icon :size="32"><Goods /></el-icon>
              <span>商品列表</span>
            </router-link>
            <router-link to="/seckill" class="nav-item">
              <el-icon :size="32"><Lightning /></el-icon>
              <span>秒杀专区</span>
            </router-link>
            <router-link to="/cart" class="nav-item" :class="{ disabled: !userStore.isLoggedIn }">
              <el-icon :size="32"><ShoppingCart /></el-icon>
              <span>购物车</span>
            </router-link>
            <router-link to="/order/list" class="nav-item" :class="{ disabled: !userStore.isLoggedIn }">
              <el-icon :size="32"><List /></el-icon>
              <span>我的订单</span>
            </router-link>
            <router-link to="/user" class="nav-item" :class="{ disabled: !userStore.isLoggedIn }">
              <el-icon :size="32"><User /></el-icon>
              <span>个人中心</span>
            </router-link>
          </div>
        </div>
        
        <div class="login-card" v-if="!userStore.isLoggedIn">
          <h2>快速登录</h2>
          <el-form :model="loginForm" class="login-form">
            <el-form-item>
              <el-input v-model="loginForm.username" placeholder="用户名" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="loginForm.password" type="password" placeholder="密码" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" style="width: 100%" @click="handleLogin" :loading="loading">
                登录
              </el-button>
            </el-form-item>
          </el-form>
          <p class="register-tip">
            还没有账户？ <router-link to="/register">立即注册</router-link>
          </p>
        </div>
        
        <div class="tip-card" v-if="!backendConnected">
          <el-alert type="warning" :closable="false">
            <template #title>
              <strong>提示：后端服务未启动</strong>
            </template>
            <p>请先启动后端服务，否则无法正常使用登录、商品等功能。</p>
            <p style="margin-top: 8px;">启动命令：<code>cd ecommerce-demo && ./start.sh</code></p>
          </el-alert>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { HomeFilled, Goods, Lightning, ShoppingCart, List, User } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import request from '@/utils/request'

const userStore = useUserStore()
const loading = ref(false)
const backendConnected = ref(false)

const loginForm = reactive({
  username: '',
  password: ''
})

const checkBackend = async () => {
  try {
    await request({
      url: '/product/list',
      method: 'GET',
      timeout: 3000
    })
    backendConnected.value = true
  } catch {
    backendConnected.value = false
  }
}

const handleLogin = async () => {
  if (!loginForm.username || !loginForm.password) return
  
  loading.value = true
  try {
    await userStore.login(loginForm)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  userStore.init()
  checkBackend()
})
</script>

<style scoped>
.simple-home {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 40px 20px;
}

.container {
  max-width: 800px;
  margin: 0 auto;
}

.header {
  text-align: center;
  color: white;
  margin-bottom: 40px;
}

.header h1 {
  font-size: 48px;
  margin-bottom: 12px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.header p {
  font-size: 18px;
  opacity: 0.9;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.status-card,
.nav-card,
.login-card,
.tip-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

h2 {
  font-size: 18px;
  margin-bottom: 16px;
  color: #333;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}

.status-item span {
  color: #666;
  min-width: 80px;
}

.nav-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px;
  background: #f5f5f5;
  border-radius: 8px;
  text-decoration: none;
  color: #333;
  transition: all 0.3s;
}

.nav-item:hover:not(.disabled) {
  background: #1890ff;
  color: white;
  transform: translateY(-2px);
}

.nav-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.login-form {
  margin-top: 16px;
}

.register-tip {
  text-align: center;
  color: #666;
  font-size: 14px;
  margin-top: 12px;
}

.register-tip a {
  color: #1890ff;
}
</style>
