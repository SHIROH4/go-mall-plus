import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { 
  AddCartReq, 
  AddCartResp, 
  GetCartResp, 
  UpdateCartReq, 
  UpdateCartResp,
  RemoveCartResp,
  ClearCartResp,
  SelectCartReq,
  SelectCartResp,
  GetSelectedCartResp
} from '@/types'

// 添加到购物车
export const addCart = (data: AddCartReq): Promise<ApiResponse<AddCartResp>> => {
  return request({
    url: '/cart/add',
    method: 'POST',
    data
  })
}

// 获取购物车列表
export const getCart = (): Promise<ApiResponse<GetCartResp>> => {
  return request({
    url: '/cart/list',
    method: 'GET'
  })
}

// 更新购物车数量
export const updateCart = (data: UpdateCartReq): Promise<ApiResponse<UpdateCartResp>> => {
  return request({
    url: '/cart/update',
    method: 'PUT',
    data
  })
}

// 删除购物车商品
export const removeCart = (productId: number): Promise<ApiResponse<RemoveCartResp>> => {
  return request({
    url: `/cart/${productId}`,
    method: 'DELETE'
  })
}

// 清空购物车
export const clearCart = (): Promise<ApiResponse<ClearCartResp>> => {
  return request({
    url: '/cart/clear',
    method: 'DELETE'
  })
}

// 选择/取消选择商品
export const selectCart = (data: SelectCartReq): Promise<ApiResponse<SelectCartResp>> => {
  return request({
    url: '/cart/select',
    method: 'PUT',
    data
  })
}

// 获取已选中的商品
export const getSelectedCart = (): Promise<ApiResponse<GetSelectedCartResp>> => {
  return request({
    url: '/cart/selected',
    method: 'GET'
  })
}
