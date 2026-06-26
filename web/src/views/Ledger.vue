<template>
  <div class="ledger">
    <div class="page-header">
      <h2>账簿查询</h2>
      <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="setCurrentBook($event); loadData()">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadData">
      <el-tab-pane label="科目余额表" name="balance">
        <div style="margin-bottom: 12px">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadData()" :size="isMobile ? 'small' : 'default'" />
        </div>
        <div class="table-wrapper">
          <el-table :data="balanceData" stripe border size="small" show-summary :summary-method="balanceSummary" :max-height="tableMaxHeight">
            <el-table-column prop="account_code" label="编码" width="100" fixed />
            <el-table-column prop="account_name" label="科目名称" min-width="120" fixed />
            <el-table-column prop="direction" label="向" width="50" align="center" />
            <el-table-column label="期初借" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.opening_debit) }}</template>
            </el-table-column>
            <el-table-column label="期初贷" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.opening_credit) }}</template>
            </el-table-column>
            <el-table-column label="本期借" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
            </el-table-column>
            <el-table-column label="本期贷" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
            </el-table-column>
            <el-table-column label="期末借" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.closing_debit) }}</template>
            </el-table-column>
            <el-table-column label="期末贷" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.closing_credit) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="总账" name="general">
        <div class="table-wrapper">
          <el-table :data="accounts" stripe border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="科目名称" min-width="150" />
            <el-table-column prop="direction" label="向" width="50" align="center" />
            <el-table-column label="余额" width="120" align="right">
              <template #default="{ row }">
                <span :style="{ color: row.is_active ? '' : '#c0c4cc' }">-</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 260px)' : 'calc(100vh - 300px)')

const books = ref([])
const { currentBookId: currentBook, setCurrentBook } = useBookStore()
const activeTab = ref('balance')
const period = ref(new Date().toISOString().slice(0, 7))
const balanceData = ref([])
const accounts = ref([])

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) { currentBook.value = books.value[0].id; await loadData() }
}

const loadData = async () => {
  if (!currentBook.value) return
  if (activeTab.value === 'balance' && period.value) {
    const { data } = await axios.get(`/api/books/${currentBook.value}/reports/account-balance?period=${period.value}`)
    balanceData.value = data.data || []
  } else if (activeTab.value === 'general') {
    const { data } = await axios.get(`/api/books/${currentBook.value}/accounts`)
    accounts.value = data.data || []
  }
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2 })
}

const balanceSummary = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i === 1 || i === 2) { sums[i] = ''; return }
    const key = ['opening_debit', 'opening_credit', 'period_debit', 'period_credit', 'closing_debit', 'closing_credit'][i - 3]
    if (key) sums[i] = fmt(data.reduce((s, r) => s + (r[key] || 0), 0))
    else sums[i] = ''
  })
  return sums
}

watch(currentBook, loadData)
onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
