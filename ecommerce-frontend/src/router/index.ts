import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

// 路由配置
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/home',
    children: [
      {
        path: 'home',
        name: 'Home',
        component: () => import('@/views/SimpleHome.vue'),
        meta: { title: '首页' }
      },
      {
        path: 'product/list',
        name: 'ProductList',
        component: () => import('@/views/product/ProductList.vue'),
        meta: { title: '商品列表' }
      },
      {
        path: 'product/detail/:id',
        name: 'ProductDetail',
        component: () => import('@/views/product/ProductDetail.vue'),
        meta: { title: '商品详情' }
      },
      {
        path: 'seckill',
        name: 'SecKill',
        component: () => import('@/views/product/SecKill.vue'),
        meta: { title: '秒杀专区' }
      },
      {
        path: 'cart',
        name: 'Cart',
        component: () => import('@/views/cart/CartIndex.vue'),
        meta: { title: '购物车', requireAuth: true }
      },
      {
        path: 'order/confirm',
        name: 'OrderConfirm',
        component: () => import('@/views/order/OrderConfirm.vue'),
        meta: { title: '确认订单', requireAuth: true }
      },
      {
        path: 'order/list',
        name: 'OrderList',
        component: () => import('@/views/order/OrderList.vue'),
        meta: { title: '我的订单', requireAuth: true }
      },
      {
        path: 'order/detail/:orderNo',
        name: 'OrderDetail',
        component: () => import('@/views/order/OrderDetail.vue'),
        meta: { title: '订单详情', requireAuth: true }
      },
      {
        path: 'payment/:orderNo',
        name: 'Payment',
        component: () => import('@/views/order/Payment.vue'),
        meta: { title: '支付', requireAuth: true }
      },
      {
        path: 'user',
        name: 'User',
        component: () => import('@/views/user/UserIndex.vue'),
        meta: { title: '个人中心', requireAuth: true }
      },
      {
        path: 'user/address',
        name: 'AddressManage',
        component: () => import('@/views/user/AddressManage.vue'),
        meta: { title: '收货地址', requireAuth: true }
      }
    ]
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/auth/Register.vue'),
    meta: { title: '注册' }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
    meta: { title: '页面不存在' }
  }
]

// 创建路由实例
const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    } else {
      return { top: 0 }
    }
  }
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const userStore = useUserStore()
  
  // 初始化用户状态
  if (!userStore.accessToken && localStorage.getItem('accessToken')) {
    userStore.init()
  }
  
  // 设置页面标题
  document.title = `${to.meta.title || '商城'} - 电商商城`
  
  // 检查是否需要登录
  if (to.meta.requireAuth) {
    if (!userStore.isLoggedIn) {
      next({ name: 'Login', query: { redirect: to.fullPath } })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
