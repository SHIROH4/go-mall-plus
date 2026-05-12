import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { 
  CreatePayReq, 
  CreatePayResp, 
  GetPayStatusResp, 
  CancelPayResp,
  ListPayReq,
  ListPayResp
} from '@/types'

// 发起支付
export const createPay = (data: CreatePayReq): Promise<ApiResponse<CreatePayResp>> => {
  return request({
    url: '/pay/create',
    method: 'POST',
    data
  })
}

// 查询支付状态
export const getPayStatus = (paymentNo: string): Promise<ApiResponse<GetPayStatusResp>> => {
  return request({
    url: `/pay/status/${paymentNo}`,
    method: 'GET'
  })
}

// 取消支付
export const cancelPay = (paymentNo: string): Promise<ApiResponse<CancelPayResp>> => {
  return request({
    url: `/pay/cancel/${paymentNo}`,
    method: 'POST'
  })
}

// 获取支付记录列表
export const listPay = (params?: ListPayReq): Promise<ApiResponse<ListPayResp>> => {
  return request({
    url: '/pay/list',
    method: 'GET',
    params
  })
}
