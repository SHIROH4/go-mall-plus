// 后端接口统一响应格式
export interface ApiResponse<T = any> {
  code: number
  msg: string
  data?: T
}

// 用户相关
export interface UserInfo {
  id: number
  username: string
}

export interface LoginReq {
  username: string
  password: string
}

export interface LoginResp {
  id: number
  accessToken: string
  refreshToken: string
}

export interface RegisterReq {
  username: string
  password: string
}

export interface RegisterResp {
  id: number
}

export interface RefreshTokenReq {
  refreshToken: string
}

export interface RefreshTokenResp {
  accessToken: string
  refreshToken: string
}

// 商品相关
export interface ProductItem {
  id: number
  name: string
  desc: string
  price: number
  image_url: string
  category_id: number
  category_name: string
  stock: number
}

export interface CategoryItem {
  id: number
  name: string
  icon: string
  sort: number
}

export interface ListProductResp {
  products: ProductItem[]
}

export interface GetProductResp {
  product: ProductItem
}

export interface ListProductByPageReq {
  category_id?: number
  keyword?: string
  page?: number
  page_size?: number
}

export interface ListProductByPageResp {
  products: ProductItem[]
  total: number
  page: number
  page_size: number
}

export interface GetCategoriesResp {
  categories: CategoryItem[]
}

// 购物车相关
export interface CartItem {
  product_id: number
  product_name: string
  price: number
  image_url: string
  count: number
  selected: boolean
  created_at: number
  updated_at: number
}

export interface AddCartReq {
  product_id: number
  product_name: string
  price: number
  image_url: string
  count: number
}

export interface AddCartResp {
  message: string
  total_count: number
}

export interface GetCartResp {
  items: CartItem[]
  total_count: number
  total_amount: number
}

export interface UpdateCartReq {
  product_id: number
  count: number
}

export interface UpdateCartResp {
  message: string
}

export interface RemoveCartResp {
  message: string
}

export interface ClearCartResp {
  message: string
  removed_count: number
}

export interface SelectCartReq {
  product_id: number
  selected: boolean
}

export interface SelectCartResp {
  message: string
}

export interface GetSelectedCartResp {
  items: CartItem[]
  selected_count: number
  total_amount: number
}

// 订单相关
export interface OrderItem {
  id: number
  order_no: string
  product_id: number
  product_name: string
  count: number
  total_amount: number
  status: number
  status_text: string
  create_time: number
  pay_time: number
}

export interface CreateOrderReq {
  productId: number
  count: number
}

export interface CreateOrderResp {
  orderNo: string
}

export interface ListOrderReq {
  page?: number
  page_size?: number
  status?: number
}

export interface ListOrderResp {
  orders: OrderItem[]
  total: number
  page: number
  page_size: number
}

export interface GetOrderDetailReq {
  order_no: string
}

export interface GetOrderDetailResp {
  id: number
  order_no: string
  product_id: number
  product_name: string
  product_desc: string
  product_image: string
  count: number
  total_amount: number
  status: number
  status_text: string
  create_time: number
  pay_time: number
  expire_time: number
}

export interface CancelOrderReq {
  order_no: string
}

export interface CancelOrderResp {
  success: boolean
  message: string
}

// 支付相关
export interface PaymentInfo {
  id: number
  payment_no: string
  order_no: string
  amount: number
  status: number
  status_text: string
  pay_channel: string
  pay_time: number
  expire_time: number
  created_at: number
}

export interface CreatePayReq {
  order_no: string
  amount: number
  pay_channel: 'alipay' | 'wechat'
}

export interface CreatePayResp {
  payment_no: string
  qr_code: string
  expire_time: number
}

export interface GetPayStatusResp {
  payment_no: string
  status: number
  status_text: string
  amount: number
  order_no: string
}

export interface CancelPayResp {
  message: string
}

export interface ListPayReq {
  page?: number
  page_size?: number
}

export interface ListPayResp {
  payments: PaymentInfo[]
  total: number
  page: number
  page_size: number
}

// 订单状态枚举
export const ORDER_STATUS = {
  PENDING_PAY: { value: 0, text: '待支付' },
  PAID: { value: 1, text: '已支付' },
  CANCELLED: { value: 2, text: '已取消' },
  EXPIRED: { value: 3, text: '已超时' }
} as const

// 支付状态枚举
export const PAY_STATUS = {
  PENDING: { value: 0, text: '待支付' },
  PAID: { value: 1, text: '已支付' },
  CANCELLED: { value: 2, text: '已取消' },
  EXPIRED: { value: 3, text: '已超时' }
} as const

// ============================================
// 收货地址相关类型
// ============================================
export interface AddressItem {
  id: number
  receiver_name: string
  phone: string
  province: string
  city: string
  district: string
  detail_address: string
  postal_code: string
  is_default: boolean
}

export interface GetAddressListResp {
  addresses: AddressItem[]
}

export interface GetAddressResp {
  address: AddressItem
}

export interface AddAddressReq {
  receiver_name: string
  phone: string
  province: string
  city: string
  district: string
  detail_address: string
  postal_code?: string
  is_default: boolean
}

export interface AddAddressResp {
  id: number
  success: boolean
  message: string
}

export interface UpdateAddressReq extends AddAddressReq {
  id: number
}

export interface UpdateAddressResp {
  success: boolean
  message: string
}

export interface DeleteAddressReq {
  id: number
}

export interface DeleteAddressResp {
  success: boolean
  message: string
}

export interface SetDefaultAddressReq {
  id: number
}

export interface SetDefaultAddressResp {
  success: boolean
  message: string
}
