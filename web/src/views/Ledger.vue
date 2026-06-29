<template>
  <div class="ledger">
    <div class="page-header">
      <h2>账簿查询</h2>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadData">
      <el-tab-pane label="总账" name="general">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadData()" :size="isMobile ? 'small' : 'default'" />
        </div>
        <div class="table-wrapper">
          <el-table :data="generalData" stripe border size="small" :max-height="tableMaxHeight" show-summary :summary-method="generalSummary">
            <el-table-column prop="code" label="科目编码" width="100" fixed />
            <el-table-column prop="name" label="科目名称" min-width="150" fixed />
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

      <el-tab-pane label="现金日记账" name="cash-journal">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadData()" :size="isMobile ? 'small' : 'default'" />
        </div>
        <div class="table-wrapper">
          <el-table :data="journalData" border size="small" :max-height="tableMaxHeight" show-summary :summary-method="journalSummary">
            <el-table-column prop="date" label="日期" width="100" fixed />
            <el-table-column prop="voucher_number" label="凭证号" width="80" />
            <el-table-column prop="memo" label="摘要" min-width="180" />
            <el-table-column label="借方" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.debit) }}</template>
            </el-table-column>
            <el-table-column label="贷方" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.credit) }}</template>
            </el-table-column>
            <el-table-column label="余额" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.balance) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="银行日记账" name="bank-journal">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadData()" :size="isMobile ? 'small' : 'default'" />
        </div>
        <div class="table-wrapper">
          <el-table :data="journalData" border size="small" :max-height="tableMaxHeight" show-summary :summary-method="journalSummary">
            <el-table-column prop="date" label="日期" width="100" fixed />
            <el-table-column prop="voucher_number" label="凭证号" width="80" />
            <el-table-column prop="memo" label="摘要" min-width="180" />
            <el-table-column label="借方" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.debit) }}</template>
            </el-table-column>
            <el-table-column label="贷方" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.credit) }}</template>
            </el-table-column>
            <el-table-column label="余额" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.balance) }}</template>
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
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'

const { isMobile } = useMobile()
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 280px)' : 'calc(100vh - 320px)')

const { currentBookId: currentBook } = useBookStore()
const activeTab = ref('general')
const period = ref(new Date().toISOString().slice(0, 7))
const generalData = ref([])
const journalData = ref([])

const loadData = async () => {
  if (!currentBook.value || !period.value) return
  if (activeTab.value === 'general') {
    const { data } = await axios.get(`/api/books/${currentBook.value}/reports/general-ledger?period=${period.value}`)
    generalData.value = data.data || []
  } else if (activeTab.value === 'cash-journal') {
    const { data } = await axios.get(`/api/books/${currentBook.value}/ledger/journal?type=cash&period=${period.value}`)
    journalData.value = data.data || []
  } else if (activeTab.value === 'bank-journal') {
    const { data } = await axios.get(`/api/books/${currentBook.value}/ledger/journal?type=bank&period=${period.value}`)
    journalData.value = data.data || []
  }
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return Number(v).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const generalSummary = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i === 1 || i === 2) { sums[i] = ''; return }
    const key = ['opening_debit', 'opening_credit', 'period_debit', 'period_credit', 'closing_debit', 'closing_credit'][i - 3]
    if (key) sums[i] = fmt(data.reduce((s, r) => s + (Number(r[key]) || 0), 0))
    else sums[i] = ''
  })
  return sums
}

const journalSummary = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i === 1 || i === 2) { sums[i] = ''; return }
    const key = ['debit', 'credit', 'balance'][i - 3]
    if (key && key !== 'balance') sums[i] = fmt(data.reduce((s, r) => s + (Number(r[key]) || 0), 0))
    else sums[i] = ''
  })
  return sums
}

watch(currentBook, (val) => { if (val) loadData() })
onMounted(() => {
  if (currentBook.value) loadData()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
