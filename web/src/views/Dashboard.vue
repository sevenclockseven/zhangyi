<template>
  <div class="dashboard">
    <h2>工作台</h2>
    <div class="stats-grid">
      <el-card shadow="hover">
        <template #header>账套总数</template>
        <div class="stat-value">{{ stats.totalBooks }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>本月凭证</template>
        <div class="stat-value">{{ stats.monthVouchers }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待审核</template>
        <div class="stat-value warning">{{ stats.pendingReview }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待记账</template>
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
          <el-button @click="$router.push('/reports')">
            <el-icon><DataAnalysis /></el-icon>查看报表
          </el-button>
        </el-space>
      </el-card>
      <el-card>
        <template #header>系统信息</template>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="版本">v0.3.0</el-descriptions-item>
          <el-descriptions-item label="数据库">SQLite</el-descriptions-item>
          <el-descriptions-item label="数据目录">./data/</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const stats = ref({
  totalBooks: 0,
  monthVouchers: 0,
  pendingReview: 0,
  pendingPost: 0
})

onMounted(async () => {
  try {
    const { data } = await axios.get('/api/books')
    stats.value.totalBooks = data.data?.length || 0
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
})
</script>

<style scoped>
.dashboard h2 {
  margin-bottom: 20px;
  color: #303133;
  font-size: 18px;
}

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

.stat-value.warning {
  color: #E6A23C;
}

.stat-value.info {
  color: #909399;
}

@media (max-width: 767px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .bottom-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .stat-value {
    font-size: 28px;
  }
}
</style>
