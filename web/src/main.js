import { createApp } from 'vue'
import { createPinia } from 'pinia'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import axios from 'axios'

import App from './App.vue'
import router from './router'

// Axios 请求拦截器 - 自动添加 token
axios.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Axios 响应拦截器 - 401 自动跳转登录
axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      // 避免在登录页死循环跳转
      const currentPath = window.location.pathname
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (currentPath !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

const app = createApp(App)

app.use(createPinia())
app.use(router)

// Element Plus 按需引入（由 unplugin-vue-components + ElementPlusResolver 自动处理）
// 中文 locale 通过全局配置注入
app.provide('elLocale', zhCn)

app.mount('#app')
