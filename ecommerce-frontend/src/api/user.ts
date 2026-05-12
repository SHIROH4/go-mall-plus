import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { LoginReq, LoginResp, RegisterReq, RegisterResp, RefreshTokenReq, RefreshTokenResp, UserInfo } from '@/types'

// 登录
export const login = (data: LoginReq): Promise<ApiResponse<LoginResp>> => {
  return request({
    url: '/user/login',
    method: 'POST',
    data
  })
}

// 注册
export const register = (data: RegisterReq): Promise<ApiResponse<RegisterResp>> => {
  return request({
    url: '/user/register',
    method: 'POST',
    data
  })
}

// 刷新 Token
export const refreshToken = (data: RefreshTokenReq): Promise<ApiResponse<RefreshTokenResp>> => {
  return request({
    url: '/user/refresh',
    method: 'POST',
    data
  })
}

// 获取用户信息
export const getUserInfo = (): Promise<ApiResponse<UserInfo>> => {
  return request({
    url: '/user/info',
    method: 'GET'
  })
}

// 退出登录
export const logout = (): Promise<ApiResponse<null>> => {
  return request({
    url: '/user/logout',
    method: 'POST'
  })
}
