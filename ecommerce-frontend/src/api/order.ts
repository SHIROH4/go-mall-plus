import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { 
  CreateOrderReq, 
  CreateOrderResp, 
  ListOrderReq, 
  ListOrderResp, 
  GetOrderDetailReq,
  GetOrderDetailResp,
  CancelOrderReq,
  CancelOrderResp
} from '@/types'

// 创建订单
export const createOrder = (data: CreateOrderReq): Promise<ApiResponse<CreateOrderResp>> => {
  return request({
    url: '/order/create',
    method: 'POST',
    data
  })
}

// 获取订单列表
export const listOrder = (params?: ListOrderReq): Promise<ApiResponse<ListOrderResp>> => {
  return request({
    url: '/order/list',
    method: 'GET',
    params
  })
}

// 获取订单详情
export const getOrderDetail = (params: GetOrderDetailReq): Promise<ApiResponse<GetOrderDetailResp>> => {
  return request({
    url: '/order/detail',
    method: 'GET',
    params
  })
}

// 取消订单
export const cancelOrder = (data: CancelOrderReq): Promise<ApiResponse<CancelOrderResp>> => {
  return request({
    url: '/order/cancel',
    method: 'POST',
    data
  })
}
