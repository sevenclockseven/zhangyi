<template>
  <div class="ledger">
    <div class="page-header">
      <h2>账簿查询</h2>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadData">
      <el-tab-pane label="科目余额表" name="balance">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadData()" :size="isMobile ? 'small' : 'default'" />
          <el-button size="small" @click="expandAll"><el-icon><Plus /></el-icon>全部展开</el-button>
          <el-button size="small" @click="collapseAll"><el-icon><Minus /></el-icon>全部折叠</el-button>
        </div>
        <div class="table-wrapper">
          <el-table
            ref="tableRef"
            :data="balanceData"
            row-key="account_code"
            :tree-props="{ children: 'children' }"
            :default-expand-all="true"
            :max-height="tableMaxHeight"
            border
            size="small"
            show-summary
            :summary-method="balanceSummary"
            row-class-name="rowClassName"
            :stripe="false"
          >
            <el-table-column prop="account_code" label="编码" width="100" fixed />
            <el-table-column prop="account_name" label="科目名称" min-width="160" fixed>
              <template #default="{ row }">
                <span :style="{ fontWeight: row.level === 1 ? 'bold' : 'normal', paddingLeft: (row.level - 1) * 16 + 'px' }">{{ row.account_name }}</span>
              </template>
            </el-table-column>
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
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'
import { accountApi } from '../api'
import { Plus, Minus } from '@element-plus/icons-vue'

const { isMobile } = useMobile()
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 260px)' : 'calc(100vh - 300px)')

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const activeTab = ref('balance')
const period = ref(new Date().toISOString().slice(0, 7))
const balanceData = ref([])
const accounts = ref([])
const tableRef = ref(null)

const expandAll = () => {
  // el-table 树形模式没有 expandAll API，用 expand-row-keys 模拟
  const allKeys = []
  const walk = (nodes) => {
    for (const n of nodes) {
      allKeys.push(n.account_code)
      if (n.children && n.children.length) walk(n.children)
    }
  }
  walk(balanceData.value)
  tableRef.value?.store?.states && (tableRef.value.store.states.expandRowKeys.value = allKeys)
}

const collapseAll = () => {
  if (tableRef.value?.store?.states) {
    tableRef.value.store.states.expandRowKeys.value = []
  }
}

const rowClassName = ({ row }) => {
  if (row.level === 1) {
    const code = row.account_code || ''
    if (code.startsWith('1')) return 'row-asset'
    if (code.startsWith('2')) return 'row-liability'
    if (code.startsWith('3')) return 'row-equity'
    if (code.startsWith('4')) return 'row-cost'
    if (code.startsWith('5')) return 'row-expense'
  }
  return ''
}

const loadData = async () => {
  if (!currentBook.value) return
  if (activeTab.value === 'balance' && period.value) {
    const { data } = await axios.get(`/api/books/${currentBook.value}/reports/account-balance?period=${period.value}`)
    balanceData.value = data.data || []
  } else if (activeTab.value === 'general') {
    const { data } = await accountApi.list(currentBook.value)
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

watch(currentBook, (val) => { if (val) loadData() })
onMounted(() => {
  if (currentBook.value) loadData()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }

/* 一级科目颜色标识 */
.row-asset td { background-color: #ecf5ff !important; }
.row-liability td { background-color: #fdf6ec !important; }
.row-equity td { background-color: #f0f9eb !important; }
.row-cost td { background-color: #f9f0ff !important; }
.row-expense td { background-color: #fef0f0 !important; }
</style>
