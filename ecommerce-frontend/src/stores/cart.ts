import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { CartItem } from '@/types'
import * as api from '@/api/cart'

export const useCartStore = defineStore('cart', () => {
  // State
  const items = ref<CartItem[]>([])
  const totalCount = ref(0)
  const totalAmount = ref(0)
  const loading = ref(false)
  
  // Computed
  const selectedItems = computed(() => items.value.filter(item => item.selected))
  const selectedCount = computed(() => selectedItems.value.reduce((sum, item) => sum + item.count, 0))
  const selectedAmount = computed(() => selectedItems.value.reduce((sum, item) => sum + item.price * item.count, 0))
  const isAllSelected = computed(() => items.value.length > 0 && items.value.every(item => item.selected))
  
  // 获取购物车列表
  const fetchCart = async (): Promise<void> => {
    loading.value = true
    try {
      const res = await api.getCart()
      items.value = res.items || []
      totalCount.value = res.total_count || 0
      totalAmount.value = res.total_amount || 0
    } catch (error) {
      console.error('Failed to fetch cart:', error)
    } finally {
      loading.value = false
    }
  }
  
  // 添加到购物车
  const addToCart = async (item: {
    product_id: number
    product_name: string
    price: number
    image_url: string
    count: number
  }): Promise<boolean> => {
    try {
      await api.addCart(item)
      await fetchCart()
      return true
    } catch (error) {
      console.error('Failed to add to cart:', error)
      return false
    }
  }
  
  // 更新购物车数量
  const updateCartItem = async (productId: number, count: number): Promise<void> => {
    try {
      await api.updateCart({ product_id: productId, count })
      await fetchCart()
    } catch (error) {
      console.error('Failed to update cart:', error)
    }
  }
  
  // 删除购物车商品
  const removeCartItem = async (productId: number): Promise<void> => {
    try {
      await api.removeCart(productId)
      await fetchCart()
    } catch (error) {
      console.error('Failed to remove cart item:', error)
    }
  }
  
  // 清空购物车
  const clearCartAction = async (): Promise<void> => {
    try {
      await api.clearCart()
      items.value = []
      totalCount.value = 0
      totalAmount.value = 0
    } catch (error) {
      console.error('Failed to clear cart:', error)
    }
  }
  
  // 选择/取消选择商品
  const toggleSelect = async (productId: number, selected: boolean): Promise<void> => {
    try {
      await api.selectCart({ product_id: productId, selected })
      await fetchCart()
    } catch (error) {
      console.error('Failed to toggle select:', error)
    }
  }
  
  // 全选/取消全选
  const toggleAllSelect = async (selected: boolean): Promise<void> => {
    // 遍历所有未选中的商品进行选择
    for (const item of items.value) {
      if (item.selected !== selected) {
        await toggleSelect(item.product_id, selected)
      }
    }
  }
  
  // 获取已选商品
  const fetchSelectedCart = async (): Promise<{ items: CartItem[]; selectedCount: number; totalAmount: number }> => {
    try {
      const res = await api.getSelectedCart()
      return {
        items: res.items,
        selectedCount: res.selected_count,
        totalAmount: res.total_amount
      }
    } catch (error) {
      console.error('Failed to fetch selected cart:', error)
      return { items: [], selectedCount: 0, totalAmount: 0 }
    }
  }
  
  // 兼容方法名
  const clearCart = clearCartAction
  
  return {
    items,
    totalCount,
    totalAmount,
    loading,
    selectedItems,
    selectedCount,
    selectedAmount,
    isAllSelected,
    fetchCart,
    addToCart,
    updateCartItem,
    removeCartItem,
    clearCart,
    clearCartAction,
    toggleSelect,
    toggleAllSelect,
    fetchSelectedCart
  }
})
