import { createRouter, createWebHistory } from 'vue-router'

const routes = [
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

export default router
