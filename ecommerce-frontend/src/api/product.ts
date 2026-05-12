import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { 
  ListProductResp, 
  GetProductResp, 
  ListProductByPageReq, 
  ListProductByPageResp,
  GetCategoriesResp,
  CategoryItem,
  ProductItem
} from '@/types'

// 获取商品列表
export const listProduct = (): Promise<ApiResponse<ListProductResp>> => {
  return request({
    url: '/product/list',
    method: 'GET'
  })
}

// 获取商品详情
export const getProduct = (id: number): Promise<ApiResponse<GetProductResp>> => {
  return request({
    url: `/product/${id}`,
    method: 'GET'
  })
}

// 分页获取商品列表
export const listProductByPage = (params: ListProductByPageReq): Promise<ApiResponse<ListProductByPageResp>> => {
  return request({
    url: '/product/list/page',
    method: 'GET',
    params
  })
}

// 获取商品分类
export const getCategories = (): Promise<ApiResponse<GetCategoriesResp>> => {
  return request({
    url: '/product/categories',
    method: 'GET'
  })
}

// 获取秒杀商品（使用分页接口，可通过分类或关键词筛选）
export const listSecKillProducts = (params?: ListProductByPageReq): Promise<ApiResponse<ListProductByPageResp>> => {
  return request({
    url: '/product/list/page',
    method: 'GET',
    params: {
      ...params,
      keyword: params?.keyword || '秒杀'
    }
  })
}
