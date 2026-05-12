<template>
  <div class="register-page">
    <div class="register-container">
      <div class="register-card">
        <div class="register-header">
          <div class="logo">
            <el-icon :size="48" color="#1890ff"><ShoppingBag /></el-icon>
          </div>
          <h1>创建账户</h1>
          <p>加入我们，开始购物之旅</p>
        </div>
        
        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          class="register-form"
          @submit.prevent="handleRegister"
        >
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              placeholder="请输入用户名 (4-20个字符)"
              size="large"
              :prefix-icon="User"
            />
          </el-form-item>
          
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码 (至少6个字符)"
              size="large"
              :prefix-icon="Lock"
              show-password
            />
          </el-form-item>
          
          <el-form-item prop="confirmPassword">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              placeholder="请确认密码"
              size="large"
              :prefix-icon="Lock"
              show-password
              @keyup.enter="handleRegister"
            />
          </el-form-item>
          
          <el-form-item>
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="register-btn"
              @click="handleRegister"
            >
              注册
            </el-button>
          </el-form-item>
        </el-form>
        
        <div class="register-footer">
          <span>已有账户？</span>
          <router-link to="/login">立即登录</router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance, FormRules } from 'element-plus'
import { User, Lock, ShoppingBag } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const validateConfirmPassword = (rule: any, value: any, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 4, max: 20, message: '用户名长度在 4 到 20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少 6 个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const handleRegister = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  
  loading.value = true
  
  try {
    await userStore.register({
      username: form.username,
      password: form.password
    })
    
    ElMessage.success('注册成功！请登录')
    
    // 跳转到登录页
    router.push('/login')
  } catch (error: any) {
    ElMessage.error(error.message || '注册失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.register-container {
  width: 100%;
  max-width: 420px;
}

.register-card {
  background: white;
  border-radius: 16px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.register-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  margin-bottom: 16px;
}

.register-header h1 {
  font-size: 24px;
  color: #333;
  margin-bottom: 8px;
}

.register-header p {
  color: #666;
  font-size: 14px;
}

.register-form {
  margin-bottom: 24px;
}

.register-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
}

.register-btn:hover {
  background: linear-gradient(135deg, #40a9ff 0%, #1890ff 100%);
}

.register-footer {
  text-align: center;
  color: #666;
  font-size: 14px;
}

.register-footer a {
  color: #1890ff;
  text-decoration: none;
  margin-left: 4px;
}

.register-footer a:hover {
  text-decoration: underline;
}

:deep(.el-input__wrapper) {
  padding: 4px 15px;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #1890ff inset;
}
</style>
