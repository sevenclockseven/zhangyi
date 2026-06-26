<template>
  <div class="dashboard">
    <h2>工作台</h2>
    <div class="stats-grid">
      <el-card shadow="hover">
        <template #header>账套总数</template>
        <div class="stat-value">{{ stats.totalBooks }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>本月凭证（全部账套）</template>
        <div class="stat-value">{{ stats.monthVouchers }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待审核（全部账套）</template>
        <div class="stat-value warning">{{ stats.pendingReview }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待记账（全部账套）</template>
        <div class="stat-value info">{{ stats.pendingPost }}</div>
      </el-card>
    </div>

    <div class="bottom-grid">
      <el-card>
        <template #header>快捷操作</template>
        <el-space wrap>
          <el-button type="primary" @click="$router.push('/books')">
            <el-icon><Plus /></el-icon>新建账套
          </el-button>
          <el-button @click="$router.push('/vouchers')">
            <el-icon><Document /></el-icon>录入凭证
          </el-button>
          <el-button @click="$router.push('/opening-balance')">
            <el-icon><Coin /></el-icon>期初余额
          </el-button>
          <el-button @click="$router.push('/reports')">
            <el-icon><DataAnalysis /></el-icon>查看报表
          </el-button>
          <el-button @click="$router.push('/closing')">
            <el-icon><SwitchButton /></el-icon>期末处理
          </el-button>
        </el-space>
      </el-card>
      <el-card>
        <template #header>系统信息</template>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="版本">{{ systemInfo.version || '-' }}</el-descriptions-item>
          <el-descriptions-item label="数据库">SQLite</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag type="success" size="small">{{ systemInfo.status || '-' }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { bookApi, voucherApi, healthApi } from '../api'

const stats = ref({
  totalBooks: 0,
  monthVouchers: 0,
  pendingReview: 0,
  pendingPost: 0
})

const systemInfo = ref({
  version: '',
  status: ''
})

onMounted(async () => {
  try {
    // Load books count
    const { data } = await bookApi.list()
    const books = data.data || []
    stats.value.totalBooks = books.length

    // Aggregate stats across ALL books
    const now = new Date()
    const monthPrefix = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
    let allVouchers = []
    for (const book of books) {
      const { data: vData } = await voucherApi.list(book.id)
      allVouchers = allVouchers.concat(vData.data || [])
    }
    stats.value.monthVouchers = allVouchers.filter(v => v.date && v.date.startsWith(monthPrefix)).length
    stats.value.pendingReview = allVouchers.filter(v => v.status === 'draft').length
    stats.value.pendingPost = allVouchers.filter(v => v.status === 'reviewed').length

    // Load system info from health API
    const { data: health } = await healthApi.check()
    systemInfo.value = health
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
})
</script>

<style scoped>
.dashboard h2 { margin-bottom: 20px; color: #303133; font-size: 18px; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.bottom-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-top: 20px;
}

.stat-value {
  font-size: 36px;
  font-weight: bold;
  color: #409EFF;
  text-align: center;
  padding: 10px 0;
}

.stat-value.warning { color: #E6A23C; }
.stat-value.info { color: #909399; }

@media (max-width: 767px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .bottom-grid { grid-template-columns: 1fr; gap: 12px; }
  .stat-value { font-size: 28px; }
}
</style>
