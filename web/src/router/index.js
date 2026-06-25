import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '登录', public: true }
  },
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { title: '工作台' }
  },
  {
    path: '/books',
    name: 'Books',
    component: () => import('../views/Books.vue'),
    meta: { title: '账套管理' }
  },
  {
    path: '/accounts',
    name: 'Accounts',
    component: () => import('../views/Accounts.vue'),
    meta: { title: '科目管理' }
  },
  {
    path: '/vouchers',
    name: 'Vouchers',
    component: () => import('../views/Vouchers.vue'),
    meta: { title: '凭证管理' }
  },
  {
    path: '/ledger',
    name: 'Ledger',
    component: () => import('../views/Ledger.vue'),
    meta: { title: '账簿查询' }
  },
  {
    path: '/reports',
    name: 'Reports',
    component: () => import('../views/Reports.vue'),
    meta: { title: '报表中心' }
  },
  {
    path: '/opening-balance',
    name: 'OpeningBalance',
    component: () => import('../views/OpeningBalance.vue'),
    meta: { title: '期初余额' }
  },
  {
    path: '/closing',
    name: 'Closing',
    component: () => import('../views/Closing.vue'),
    meta: { title: '期末处理' }
  },
  {
    path: '/custom-reports',
    name: 'CustomReports',
    component: () => import('../views/CustomReports.vue'),
    meta: { title: '自定义报表' }
  },
  {
    path: '/voucher-templates',
    name: 'VoucherTemplates',
    component: () => import('../views/VoucherTemplates.vue'),
    meta: { title: '凭证模板' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue'),
    meta: { title: '系统设置' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router
