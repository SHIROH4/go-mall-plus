import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type {
  AddressItem,
  GetAddressListResp,
  GetAddressResp,
  AddAddressReq,
  AddAddressResp,
  UpdateAddressReq,
  UpdateAddressResp,
  DeleteAddressResp,
  SetDefaultAddressResp
} from '@/types'

// 获取地址列表
export const getAddressList = (): Promise<ApiResponse<GetAddressListResp>> => {
  return request({
    url: '/address/list',
    method: 'GET'
  })
}

// 获取单个地址
export const getAddress = (id: number): Promise<ApiResponse<GetAddressResp>> => {
  return request({
    url: `/address/${id}`,
    method: 'GET'
  })
}

// 添加地址
export const addAddress = (data: AddAddressReq): Promise<ApiResponse<AddAddressResp>> => {
  return request({
    url: '/address/add',
    method: 'POST',
    data
  })
}

// 更新地址
export const updateAddress = (data: UpdateAddressReq): Promise<ApiResponse<UpdateAddressResp>> => {
  return request({
    url: '/address/update',
    method: 'PUT',
    data
  })
}

// 删除地址
export const deleteAddress = (id: number): Promise<ApiResponse<DeleteAddressResp>> => {
  return request({
    url: `/address/${id}`,
    method: 'DELETE'
  })
}

// 设置默认地址
export const setDefaultAddress = (id: number): Promise<ApiResponse<SetDefaultAddressResp>> => {
  return request({
    url: '/address/set-default',
    method: 'POST',
    data: { id }
  })
}
