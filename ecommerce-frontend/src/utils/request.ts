import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useUserStore } from '@/stores/user'
import { useCartStore } from '@/stores/cart'

// 创建 axios 实例
const service: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 标记是否正在刷新 Token
let isRefreshing = false
// 存储等待刷新 Token 的请求队列
let refreshSubscribers: Array<(token: string) => void> = []

// 添加 Token 到请求队列
const subscribeTokenRefresh = (callback: (token: string) => void) => {
  refreshSubscribers.push(callback)
}

// 通知所有等待的请求
const onTokenRefreshed = (token: string) => {
  refreshSubscribers.forEach(callback => callback(token))
  refreshSubscribers = []
}

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const userStore = useUserStore()
    
    if (userStore.accessToken && config.headers) {
      config.headers.Authorization = `Bearer ${userStore.accessToken}`
    }
    
    return config
  },
  (error: AxiosError) => {
    console.error('请求错误:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response) => {
    const res = response.data

    // 根据后端返回的 code 判断
    if (res.code === 0) {
      return res.data  // 返回 data 部分，而不是整个响应对象
    }

    // 业务错误
    ElMessage.error(res.msg || '请求失败')
    return Promise.reject(new Error(res.msg || '请求失败'))
  },
  async (error: AxiosError<{ code?: number; msg?: string }>) => {
    const userStore = useUserStore()
    const cartStore = useCartStore()
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    // 处理业务 code 为 401（Token 过期）- 直接退出登录
    if (error.response?.data?.code === 401) {
      // 如果是刷新 Token 的请求失败，直接退出
      if (originalRequest.url?.includes('/user/refresh')) {
        userStore.logout()
        cartStore.clearCart()
        router.push('/login')
        return Promise.reject(error)
      }

      // 有 refreshToken，尝试刷新
      if (userStore.refreshToken) {
        if (!isRefreshing) {
          isRefreshing = true
          try {
            const res = await userStore.refreshTokenAction()
            if (res) {
              onTokenRefreshed(res.accessToken)
              if (originalRequest.headers) {
                originalRequest.headers.Authorization = `Bearer ${res.accessToken}`
              }
              return service(originalRequest)
            }
          } catch {
            // 刷新失败，退出登录
          } finally {
            isRefreshing = false
            refreshSubscribers = []
          }
        }

        // 等待 token 刷新完成后重试
        return new Promise((resolve, _reject) => {
          subscribeTokenRefresh((token: string) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`
            }
            resolve(service(originalRequest))
          })
        })
      }

      // 没有 refreshToken，直接退出登录
      userStore.logout()
      cartStore.clearCart()
      router.push('/login')
      return Promise.reject(error)
    }

    // 处理 HTTP 401 错误
    if (error.response?.status === 401) {
      userStore.logout()
      cartStore.clearCart()
      router.push('/login')
      return Promise.reject(error)
    }

    // 处理其他错误
    if (error.response?.data?.msg) {
      ElMessage.error(error.response.data.msg)
    } else if (error.message) {
      ElMessage.error(error.message)
    } else {
      ElMessage.error('网络错误，请稍后重试')
    }

    return Promise.reject(error)
  }
)

export default service
