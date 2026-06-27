import { createApp } from 'vue'
import { createPinia } from 'pinia'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import axios from 'axios'

// Element Plus 全量 CSS（按需引入反复踩坑：overlay/teleport组件样式丢失，得不偿失）
import 'element-plus/dist/index.css'

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

// Element Plus 组件按需引入（JS tree-shaking 由 unplugin-vue-components 自动处理）
// CSS 已全量引入，无需额外处理

app.mount('#app')
