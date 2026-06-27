<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">
        <svg viewBox="0 0 120 120" width="64" height="64">
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
        <h1>易记</h1>
        <p>代理记账系统</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin" label-width="0">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" placeholder="密码" prefix-icon="Lock" type="password" size="large" show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="handleLogin">登 录</el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">默认账号: admin / admin123</div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const form = ref({ username: '', password: '' })

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const { data } = await authApi.login(form.value)
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    localStorage.setItem('book_permissions', JSON.stringify(data.book_permissions || []))
    ElMessage.success('登录成功')
    router.push('/')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  padding: 40px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}

.login-logo {
  text-align: center;
  margin-bottom: 32px;
}

.login-logo h1 {
  margin: 12px 0 4px;
  font-size: 28px;
  color: #303133;
}

.login-logo p {
  color: #909399;
  font-size: 14px;
}

.login-footer {
  text-align: center;
  color: #c0c4cc;
  font-size: 12px;
  margin-top: 16px;
}
</style>
