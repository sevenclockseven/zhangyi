<template>
  <div class="ledger">
    <div class="page-header">
      <h2>账簿查询</h2>
      <el-select v-model="currentBook" placeholder="选择账套" style="width: 200px" @change="loadData">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadData">
      <!-- 科目余额表 -->
      <el-tab-pane label="科目余额表" name="balance">
        <el-row :gutter="12" style="margin-bottom: 12px">
          <el-col :span="6">
            <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="选择期间" @change="loadData" />
          </el-col>
        </el-row>
        <el-table :data="balanceData" stripe border size="small" show-summary :summary-method="balanceSummary">
          <el-table-column prop="account_code" label="科目编码" width="120" />
          <el-table-column prop="account_name" label="科目名称" min-width="180" />
          <el-table-column prop="direction" label="方向" width="60" align="center" />
          <el-table-column label="期初借方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.opening_debit) }}</template>
          </el-table-column>
          <el-table-column label="期初贷方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.opening_credit) }}</template>
          </el-table-column>
          <el-table-column label="本期借方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
          </el-table-column>
          <el-table-column label="本期贷方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
          </el-table-column>
          <el-table-column label="期末借方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.closing_debit) }}</template>
          </el-table-column>
          <el-table-column label="期末贷方" width="120" align="right">
            <template #default="{ row }">{{ fmt(row.closing_credit) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 总账 -->
      <el-tab-pane label="总账" name="general">
        <el-table :data="accounts" stripe border size="small">
          <el-table-column prop="code" label="科目编码" width="120" />
          <el-table-column prop="name" label="科目名称" min-width="180" />
          <el-table-column prop="direction" label="方向" width="60" align="center" />
          <el-table-column label="余额" width="140" align="right">
            <template #default="{ row }">
              <span :style="{ color: row.is_active ? '' : '#c0c4cc' }">-</span>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import axios from 'axios'

const books = ref([])
const currentBook = ref(null)
const activeTab = ref('balance')
const period = ref(new Date().toISOString().slice(0, 7))
const balanceData = ref([])
const accounts = ref([])

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
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
    if (key) {
      sums[i] = fmt(data.reduce((s, r) => s + (r[key] || 0), 0))
    } else {
      sums[i] = ''
    }
  })
  return sums
}

watch(currentBook, loadData)
onMounted(loadBooks)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { color: #303133; }
</style>
