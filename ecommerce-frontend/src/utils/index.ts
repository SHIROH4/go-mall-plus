// 格式化金额（分转元）
export const formatPrice = (price: number): string => {
  if (price === null || price === undefined) return '0.00'
  return (price / 100).toFixed(2)
}

// 格式化金额（分转元，带符号）
export const formatPriceWithSymbol = (price: number): string => {
  return `¥${formatPrice(price)}`
}

// 格式化日期时间
export const formatDateTime = (timestamp: number): string => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 格式化日期
export const formatDate = (timestamp: number): string => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString('zh-CN')
}

// 格式化倒计时
export const formatCountdown = (expireTime: number): string => {
  const now = Math.floor(Date.now() / 1000)
  const diff = expireTime - now
  
  if (diff <= 0) return '已过期'
  
  const hours = Math.floor(diff / 3600)
  const minutes = Math.floor((diff % 3600) / 60)
  const seconds = diff % 60
  
  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

// 获取订单状态文本
export const getOrderStatusText = (status: number): string => {
  const statusMap: Record<number, string> = {
    0: '待支付',
    1: '已支付',
    2: '已取消',
    3: '已超时'
  }
  return statusMap[status] || '未知'
}

// 获取支付状态文本
export const getPayStatusText = (status: number): string => {
  const statusMap: Record<number, string> = {
    0: '待支付',
    1: '已支付',
    2: '已取消',
    3: '已超时'
  }
  return statusMap[status] || '未知'
}

// 防抖函数
export const debounce = <T extends (...args: any[]) => any>(
  func: T,
  wait: number
): ((...args: Parameters<T>) => void) => {
  let timeout: ReturnType<typeof setTimeout> | null = null
  
  return (...args: Parameters<T>) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => {
      func(...args)
    }, wait)
  }
}

// 节流函数
export const throttle = <T extends (...args: any[]) => any>(
  func: T,
  wait: number
): ((...args: Parameters<T>) => void) => {
  let timeout: ReturnType<typeof setTimeout> | null = null
  let lastTime = 0
  
  return (...args: Parameters<T>) => {
    const now = Date.now()
    if (now - lastTime >= wait) {
      lastTime = now
      func(...args)
    } else if (!timeout) {
      timeout = setTimeout(() => {
        lastTime = Date.now()
        func(...args)
      }, wait - (now - lastTime))
    }
  }
}

// 生成分页数据
export const generatePagination = (current: number, total: number, size: number = 10) => {
  const pages: (number | string)[] = []
  const totalPages = Math.ceil(total / size)
  
  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) {
      pages.push(i)
    }
  } else {
    if (current <= 3) {
      for (let i = 1; i <= 5; i++) pages.push(i)
      pages.push('...')
      pages.push(totalPages)
    } else if (current >= totalPages - 2) {
      pages.push(1)
      pages.push('...')
      for (let i = totalPages - 4; i <= totalPages; i++) pages.push(i)
    } else {
      pages.push(1)
      pages.push('...')
      for (let i = current - 1; i <= current + 1; i++) pages.push(i)
      pages.push('...')
      pages.push(totalPages)
    }
  }
  
  return pages
}
