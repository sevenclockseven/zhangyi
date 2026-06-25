<template>
  <el-container class="app-container">
    <!-- Sidebar -->
    <el-aside width="220px" class="app-aside">
      <div class="logo">
        <h2>📊 账易</h2>
        <span>代理记账系统</span>
      </div>
      <el-menu
        :default-active="route.path"
        router
        class="aside-menu"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item index="/">
          <el-icon><HomeFilled /></el-icon>
          <span>工作台</span>
        </el-menu-item>
        <el-menu-item index="/books">
          <el-icon><Notebook /></el-icon>
          <span>账套管理</span>
        </el-menu-item>
        <el-menu-item index="/accounts">
          <el-icon><Memo /></el-icon>
          <span>科目管理</span>
        </el-menu-item>
        <el-menu-item index="/vouchers">
          <el-icon><Document /></el-icon>
          <span>凭证管理</span>
        </el-menu-item>
        <el-menu-item index="/ledger">
          <el-icon><List /></el-icon>
          <span>账簿查询</span>
        </el-menu-item>
        <el-menu-item index="/reports">
          <el-icon><DataAnalysis /></el-icon>
          <span>报表中心</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- Main content -->
    <el-container>
      <el-header class="app-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-tag type="success" size="small">v0.1.0</el-tag>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

.app-container {
  height: 100vh;
}

.app-aside {
  background-color: #304156;
  overflow: hidden;
}

.logo {
  padding: 20px;
  text-align: center;
  color: #fff;
  border-bottom: 1px solid #3d4f65;
}

.logo h2 {
  font-size: 24px;
  margin-bottom: 4px;
}

.logo span {
  font-size: 12px;
  color: #bfcbd9;
}

.aside-menu {
  border-right: none;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e6e6e6;
  background: #fff;
}

.app-main {
  background-color: #f5f7fa;
  padding: 20px;
}
</style>
