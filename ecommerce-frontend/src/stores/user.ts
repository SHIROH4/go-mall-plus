import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserInfo, LoginReq, LoginResp, RefreshTokenResp } from '@/types'
import * as api from '@/api/user'

export const useUserStore = defineStore('user', () => {
  // State
  const userInfo = ref<UserInfo | null>(null)
  const accessToken = ref<string>('')
  const refreshToken = ref<string>('')
  const isLoggedIn = computed(() => !!accessToken.value)
  
  // 初始化 - 从 localStorage 恢复登录状态
  const init = () => {
    const storedAccessToken = localStorage.getItem('accessToken')
    const storedRefreshToken = localStorage.getItem('refreshToken')
    const storedUserInfo = localStorage.getItem('userInfo')
    
    if (storedAccessToken) {
      accessToken.value = storedAccessToken
    }
    if (storedRefreshToken) {
      refreshToken.value = storedRefreshToken
    }
    if (storedUserInfo) {
      try {
        userInfo.value = JSON.parse(storedUserInfo)
      } catch (e) {
        console.error('Failed to parse user info:', e)
      }
    }
  }
  
  // 登录
  const login = async (data: LoginReq): Promise<LoginResp> => {
    const res = await api.login(data)
    accessToken.value = res.accessToken
    refreshToken.value = res.refreshToken
    
    // 保存到 localStorage
    localStorage.setItem('accessToken', res.accessToken)
    localStorage.setItem('refreshToken', res.refreshToken)
    
    // 获取用户信息
    await fetchUserInfo()
    
    return res
  }
  
  // 注册
  const register = async (data: { username: string; password: string }): Promise<number> => {
    const res = await api.register(data)
    return res.id
  }
  
  // 获取用户信息
  const fetchUserInfo = async (): Promise<void> => {
    try {
      const res = await api.getUserInfo()
      userInfo.value = {
        id: res.id,
        username: res.username
      }
      localStorage.setItem('userInfo', JSON.stringify(userInfo.value))
    } catch (error) {
      console.error('Failed to fetch user info:', error)
    }
  }
  
  // 刷新 Token
  const refreshTokenAction = async (): Promise<RefreshTokenResp | null> => {
    if (!refreshToken.value) return null
    
    try {
      const res = await api.refreshToken({ refreshToken: refreshToken.value })
      
      accessToken.value = res.accessToken
      refreshToken.value = res.refreshToken
      
      localStorage.setItem('accessToken', res.accessToken)
      localStorage.setItem('refreshToken', res.refreshToken)
      
      return res
    } catch (error) {
      console.error('Failed to refresh token:', error)
      return null
    }
  }
  
  // 退出登录
  const logout = async (): Promise<void> => {
    try {
      await api.logout()
    } catch (error) {
      console.error('Logout failed:', error)
    }
    
    // 清理状态
    userInfo.value = null
    accessToken.value = ''
    refreshToken.value = ''
    
    // 清理 localStorage
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('userInfo')
  }
  
  // 兼容方法名
  const refreshTokenFn = refreshTokenAction

  return {
    userInfo,
    accessToken,
    refreshToken,
    isLoggedIn,
    init,
    login,
    register,
    fetchUserInfo,
    refreshTokenFn,
    refreshTokenAction,
    logout
  }
})
