<template>
  <div class="reports">
    <div class="page-header">
      <h2>报表中心</h2>
      <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="loadReport">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadReport">
      <el-tab-pane label="资产负债表" name="balance-sheet" />
      <el-tab-pane label="利润表" name="income-statement" />
      <el-tab-pane label="科目余额表" name="account-balance" />

      <div style="margin-bottom: 12px">
        <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadReport" :size="isMobile ? 'small' : 'default'" />
      </div>

      <!-- 资产负债表 -->
      <div v-if="activeTab === 'balance-sheet' && reportData">
        <div :class="isMobile ? 'report-stack' : 'report-row'">
          <el-card shadow="never">
            <template #header><strong>资产</strong></template>
            <div class="table-wrapper">
              <el-table :data="reportData.assets" border size="small" show-summary :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="期末余额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.balance) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
          <el-card shadow="never">
            <template #header><strong>负债及权益</strong></template>
            <div class="table-wrapper">
              <el-table :data="[...(reportData.liabilities || []), ...(reportData.equity || [])]" border size="small" show-summary :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="期末余额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.balance) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
        </div>
      </div>

      <!-- 利润表 -->
      <div v-if="activeTab === 'income-statement' && reportData">
        <div :class="isMobile ? 'report-stack' : 'report-row'">
          <el-card shadow="never">
            <template #header><strong>收入</strong></template>
            <div class="table-wrapper">
              <el-table :data="reportData.revenue" border size="small" :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="本期金额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.amount) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
          <el-card shadow="never">
            <template #header><strong>费用</strong></template>
            <div class="table-wrapper">
              <el-table :data="reportData.expenses" border size="small" :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="本期金额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.amount) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
        </div>
      </div>

      <!-- 科目余额表 -->
      <div v-if="activeTab === 'account-balance' && reportData">
        <div class="table-wrapper">
          <el-table :data="reportData" border size="small" show-summary :max-height="tableMaxHeight">
            <el-table-column prop="account_code" label="编码" width="90" fixed />
            <el-table-column prop="account_name" label="科目" min-width="120" fixed />
            <el-table-column prop="direction" label="向" width="50" align="center" />
            <el-table-column label="期初" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.opening_debit || row.opening_credit) }}</template>
            </el-table-column>
            <el-table-column label="本期借" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
            </el-table-column>
            <el-table-column label="本期贷" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
            </el-table-column>
            <el-table-column label="期末" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.closing_debit || row.closing_credit) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = isMobile.value ? 'calc(100vh - 320px)' : 'calc(100vh - 350px)'

const books = ref([])
const currentBook = ref(null)
const activeTab = ref('balance-sheet')
const period = ref(new Date().toISOString().slice(0, 7))
const reportData = ref(null)

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
}

const loadReport = async () => {
  if (!currentBook.value || !period.value) return
  reportData.value = null
  try {
    if (activeTab.value === 'balance-sheet') {
      const { data } = await axios.get(`/api/books/${currentBook.value}/reports/balance-sheet?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'income-statement') {
      const { data } = await axios.get(`/api/books/${currentBook.value}/reports/income-statement?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'account-balance') {
      const { data } = await axios.get(`/api/books/${currentBook.value}/reports/account-balance?period=${period.value}`)
      reportData.value = data.data || []
    }
  } catch (e) { console.error(e) }
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2 })
}

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }

.report-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.report-stack { display: flex; flex-direction: column; gap: 12px; }

.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
