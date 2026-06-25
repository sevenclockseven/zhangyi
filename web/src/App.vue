<template>
  <el-container class="app-container" v-if="!isLoginPage">
    <!-- Mobile overlay -->
    <div class="sidebar-overlay" v-if="sidebarOpen && isMobile" @click="sidebarOpen = false"></div>

    <!-- Sidebar -->
    <el-aside :width="sidebarWidth" class="app-aside" :class="{ 'sidebar-mobile': isMobile, 'sidebar-open': sidebarOpen }">
      <div class="logo">
        <svg viewBox="0 0 120 120" width="40" height="40">
          <defs>
            <linearGradient id="lg" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style="stop-color:#409EFF"/>
              <stop offset="100%" style="stop-color:#2b5ca6"/>
            </linearGradient>
          </defs>
          <rect width="120" height="120" rx="24" fill="url(#lg)"/>
          <rect x="24" y="16" width="72" height="88" rx="8" fill="white" opacity="0.95"/>
          <rect x="34" y="32" width="52" height="4" rx="2" fill="#409EFF" opacity="0.7"/>
          <rect x="34" y="42" width="40" height="4" rx="2" fill="#409EFF" opacity="0.4"/>
          <rect x="34" y="52" width="52" height="4" rx="2" fill="#409EFF" opacity="0.7"/>
          <rect x="34" y="62" width="36" height="4" rx="2" fill="#409EFF" opacity="0.4"/>
          <rect x="34" y="72" width="52" height="4" rx="2" fill="#409EFF" opacity="0.7"/>
          <circle cx="90" cy="86" r="20" fill="#67C23A"/>
          <polyline points="81,86 87,92 99,78" fill="none" stroke="white" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <div class="logo-text" v-show="!isMobile || sidebarOpen">
          <h2>账易</h2>
          <span>代理记账系统</span>
        </div>
      </div>
      <el-menu
        :default-active="route.path"
        router
        class="aside-menu"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
        @select="onMenuSelect"
      >
        <el-menu-item v-for="item in menuItems" :key="item.index" :index="item.index">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- Main content -->
    <el-container>
      <el-header class="app-header">
        <div class="header-left">
          <el-icon class="menu-toggle" @click="sidebarOpen = !sidebarOpen" v-if="isMobile">
            <Fold v-if="sidebarOpen" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/" v-if="!isMobile">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title && route.meta.title !== '工作台'">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
          <span class="page-title" v-else>{{ route.meta.title || '账易' }}</span>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><User /></el-icon>
              <span v-if="!isMobile">{{ currentUser.real_name || currentUser.username || '用户' }}</span>
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>

    <!-- 修改密码对话框 -->
    <el-dialog v-model="showPasswordDialog" title="修改密码" :width="isMobile ? '90%' : '400px'">
      <el-form :model="passwordForm" label-width="80px">
        <el-form-item label="原密码">
          <el-input v-model="passwordForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPasswordDialog = false">取消</el-button>
        <el-button type="primary" @click="changePassword">确定</el-button>
      </template>
    </el-dialog>
  </el-container>

  <router-view v-if="isLoginPage" />
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting, SwitchButton, Coin } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

const isLoginPage = computed(() => route.path === '/login')
const currentUser = ref({})
const showPasswordDialog = ref(false)
const passwordForm = ref({ old_password: '', new_password: '' })

// Mobile responsive
const isMobile = ref(window.innerWidth < 768)
const sidebarOpen = ref(false)
const sidebarWidth = '220px'

// Default menu config
const defaultMenuConfig = [
  { index: '/', label: '工作台', icon: 'HomeFilled', visible: true },
  { index: '/books', label: '账套管理', icon: 'Notebook', visible: true },
  { index: '/accounts', label: '科目管理', icon: 'Memo', visible: true },
  { index: '/vouchers', label: '凭证管理', icon: 'Document', visible: true },
  { index: '/ledger', label: '账簿查询', icon: 'List', visible: true },
  { index: '/reports', label: '报表中心', icon: 'DataAnalysis', visible: true },
  { index: '/opening-balance', label: '期初余额', icon: 'Coin', visible: true },
  { index: '/closing', label: '期末处理', icon: 'SwitchButton', visible: true },
  { index: '/settings', label: '系统设置', icon: 'Setting', visible: true },
]

const iconMap = { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting, SwitchButton, Coin }

// Load menu config from localStorage
const menuConfig = ref(defaultMenuConfig)
const menuItems = computed(() =>
  menuConfig.value
    .filter(item => item.visible !== false)
    .map(item => ({ ...item, icon: iconMap[item.icon] || HomeFilled }))
)

const loadMenuConfig = () => {
  try {
    const saved = localStorage.getItem('zhangyi_menu_config')
    if (saved) {
      const parsed = JSON.parse(saved)
      // Use saved order, append any new default items not in saved config
      const savedIndexes = new Set(parsed.map(p => p.index))
      const extras = defaultMenuConfig.filter(d => !savedIndexes.has(d.index))
      menuConfig.value = [...parsed, ...extras].map(item => {
        const def = defaultMenuConfig.find(d => d.index === item.index)
        return def ? { ...def, ...item } : item
      })
    }
  } catch {}
}

// Expose for Settings page
if (typeof window !== 'undefined') {
  window.__menuConfig = menuConfig
  window.__saveMenuConfig = () => {
    localStorage.setItem('zhangyi_menu_config', JSON.stringify(menuConfig.value))
  }
  window.__resetMenuConfig = () => {
    menuConfig.value = [...defaultMenuConfig]
    localStorage.removeItem('zhangyi_menu_config')
  }
}

const handleResize = () => {
  isMobile.value = window.innerWidth < 768
  if (!isMobile.value) sidebarOpen.value = false
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
  window.addEventListener('menu-config-changed', loadMenuConfig)
  const user = localStorage.getItem('user')
  if (user) currentUser.value = JSON.parse(user)
  loadMenuConfig()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('menu-config-changed', loadMenuConfig)
})



const onMenuSelect = () => {
  if (isMobile.value) sidebarOpen.value = false
}

const handleCommand = (cmd) => {
  if (cmd === 'logout') {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    router.push('/login')
  } else if (cmd === 'password') {
    passwordForm.value = { old_password: '', new_password: '' }
    showPasswordDialog.value = true
  }
}

const changePassword = async () => {
  try {
    await axios.put('/api/auth/password', passwordForm.value)
    ElMessage.success('密码修改成功')
    showPasswordDialog.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  }
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; -webkit-font-smoothing: antialiased; }
.app-container { height: 100vh; }
.app-aside { background-color: #304156; overflow: hidden; transition: transform 0.3s ease; }
.sidebar-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); z-index: 999; }
.sidebar-mobile { position: fixed; top: 0; left: 0; bottom: 0; z-index: 1000; transform: translateX(-100%); }
.sidebar-mobile.sidebar-open { transform: translateX(0); }
.logo { padding: 16px 20px; display: flex; align-items: center; gap: 12px; border-bottom: 1px solid #3d4f65; }
.logo-text h2 { font-size: 20px; color: #fff; margin: 0; line-height: 1.2; }
.logo-text span { font-size: 11px; color: #bfcbd9; }
.aside-menu { border-right: none; }
.app-header { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #e6e6e6; background: #fff; padding: 0 16px; }
.header-left { display: flex; align-items: center; gap: 8px; }
.menu-toggle { font-size: 20px; cursor: pointer; color: #606266; }
.page-title { font-size: 16px; font-weight: 500; color: #303133; }
.user-info { display: flex; align-items: center; gap: 4px; cursor: pointer; color: #606266; font-size: 14px; }
.app-main { background-color: #f5f7fa; padding: 16px; overflow-x: hidden; }
@media (max-width: 767px) {
  .app-main { padding: 12px; }
  .el-dialog { width: 90% !important; margin: 10vh auto !important; }
}
</style>
