<template>
  <div class="closing">
    <div class="page-header">
      <h2>期末处理</h2>
      <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="loadStatus">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <div v-if="currentBook" class="closing-content">
      <!-- 当前状态 -->
      <el-card shadow="never">
        <template #header><strong>结账状态</strong></template>
        <el-descriptions :column="isMobile ? 1 : 2" border size="small">
          <el-descriptions-item label="当前期间">{{ status.current_period || '-' }}</el-descriptions-item>
          <el-descriptions-item label="结账状态">
            <el-tag :type="status.is_closed ? 'danger' : 'success'" size="small">
              {{ status.is_closed ? '已结账' : '未结账' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="未记账凭证">
            <el-tag :type="status.unposted_count > 0 ? 'warning' : 'success'" size="small">
              {{ status.unposted_count || 0 }} 张
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="结账日期">{{ status.closed_at || '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 操作区 -->
      <div class="action-cards">
        <!-- 损益结转 -->
        <el-card shadow="hover" class="action-card">
          <div class="action-icon" style="background: #409EFF">
            <el-icon size="32"><Switch /></el-icon>
          </div>
          <h3>损益结转</h3>
          <p>将所有损益类科目余额结转到"本年利润"，自动生成结转凭证</p>
          <el-button type="primary" @click="doAutoTransfer" :loading="loading" :disabled="status.is_closed">
            执行损益结转
          </el-button>
        </el-card>

        <!-- 期末结账 -->
        <el-card shadow="hover" class="action-card">
          <div class="action-icon" style="background: #67C23A">
            <el-icon size="32"><CircleCheck /></el-icon>
          </div>
          <h3>期末结账</h3>
          <p>锁定本期所有凭证，生成期末余额作为下期期初，开启下一会计期间</p>
          <el-button type="success" @click="doClose" :loading="loading" :disabled="status.is_closed">
            执行期末结账
          </el-button>
        </el-card>

        <!-- 反结账 -->
        <el-card shadow="hover" class="action-card">
          <div class="action-icon" style="background: #E6A23C">
            <el-icon size="32"><RefreshLeft /></el-icon>
          </div>
          <h3>反结账</h3>
          <p>解锁已结账期间，可修改本期数据。需要先删除自动生成的结转凭证</p>
          <el-button type="warning" @click="doUnclose" :loading="loading" :disabled="!status.is_closed">
            执行反结账
          </el-button>
        </el-card>
      </div>

      <!-- 操作日志 -->
      <el-card shadow="never" style="margin-top: 16px">
        <template #header><strong>操作说明</strong></template>
        <el-steps :active="currentStep" direction="vertical" :space="40">
          <el-step title="第一步：检查凭证" description="确保本期所有凭证已审核并记账" />
          <el-step title="第二步：损益结转" description="自动生成损益结转凭证，将收入/费用结转到本年利润" />
          <el-step title="第三步：审核结转凭证" description="到凭证管理中审核并记账自动生成的结转凭证" />
          <el-step title="第四步：期末结账" description="锁定本期，生成下期期初余额" />
        </el-steps>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const books = ref([])
const currentBook = ref(null)
const loading = ref(false)
const status = ref({
  current_period: '',
  is_closed: false,
  unposted_count: 0,
  closed_at: ''
})

const currentStep = computed(() => {
  if (status.value.is_closed) return 4
  if (status.value.unposted_count === 0) return 2
  return 1
})

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
}

const loadStatus = async () => {
  if (!currentBook.value) return
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/closing/status`)
    status.value = data.data || status.value
  } catch (e) {
    console.error(e)
  }
}

const doAutoTransfer = async () => {
  await ElMessageBox.confirm('执行损益结转将自动生成结转凭证，确定继续？', '损益结转', { type: 'info' })
  loading.value = true
  try {
    const { data } = await axios.post(`/api/books/${currentBook.value}/closing/auto-transfer`)
    ElMessage.success(data.message || '损益结转完成')
    loadStatus()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '损益结转失败')
  } finally {
    loading.value = false
  }
}

const doClose = async () => {
  if (status.value.unposted_count > 0) {
    await ElMessageBox.confirm(`当前有 ${status.value.unposted_count} 张未记账凭证，结账后将无法修改。确定继续？`, '期末结账', { type: 'warning' })
  } else {
    await ElMessageBox.confirm('确定执行期末结账？结账后本期凭证将被锁定。', '期末结账', { type: 'warning' })
  }
  loading.value = true
  try {
    const { data } = await axios.post(`/api/books/${currentBook.value}/closing/close`)
    ElMessage.success(data.message || '期末结账完成')
    loadStatus()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '期末结账失败')
  } finally {
    loading.value = false
  }
}

const doUnclose = async () => {
  await ElMessageBox.confirm('反结账将解锁本期，可以修改数据。确定继续？', '反结账', { type: 'warning' })
  loading.value = true
  try {
    const { data } = await axios.post(`/api/books/${currentBook.value}/closing/unclose`)
    ElMessage.success(data.message || '反结账完成')
    loadStatus()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '反结账失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadBooks()
  watch(currentBook, (newVal) => {
    if (newVal) loadStatus()
  })
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }

.closing-content { display: flex; flex-direction: column; gap: 16px; }

.action-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.action-card {
  text-align: center;
  padding: 20px 16px;
}

.action-icon {
  width: 60px; height: 60px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 12px; color: white;
}

.action-card h3 { margin: 0 0 8px; font-size: 16px; color: #303133; }
.action-card p { margin: 0 0 16px; font-size: 13px; color: #909399; line-height: 1.6; }

@media (max-width: 767px) {
  .action-cards { grid-template-columns: 1fr; }
}
</style>
