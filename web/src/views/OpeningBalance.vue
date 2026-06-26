<template>
  <div class="opening-balance">
    <div class="page-header">
      <h2>期初余额</h2>
    </div>

    <div v-if="currentBook">
      <el-alert
        :title="isBalanced ? '试算平衡 ✓' : '试算不平衡 ✗'"
        :description="balanceDesc"
        :type="isBalanced ? 'success' : 'error'"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />

      <div class="toolbar">
        <el-button type="primary" size="small" @click="saveAll" :loading="saving" :disabled="!isBalanced">
          <el-icon><Check /></el-icon>保存
        </el-button>
        <el-button size="small" @click="exportData">
          <el-icon><Download /></el-icon>导出
        </el-button>
        <el-upload
          :action="importUrl"
          :headers="uploadHeaders"
          :show-file-list="false"
          :on-success="onImportSuccess"
          :on-error="onImportError"
          accept=".csv"
          style="display: inline-block; margin-left: 8px"
        >
          <el-button size="small"><el-icon><Upload /></el-icon>导入CSV</el-button>
        </el-upload>
        <el-button size="small" @click="loadData" style="margin-left: 8px">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
      </div>

      <div class="table-wrapper">
        <el-table :data="balances" border size="small" show-summary :summary-method="summaryMethod" :max-height="tableMaxHeight" row-key="account_id">
          <el-table-column prop="account_code" label="科目编码" width="110" fixed />
          <el-table-column prop="account_name" label="科目名称" min-width="140" fixed />
          <el-table-column prop="direction" label="方向" width="50" align="center" />
          <el-table-column label="期初借方" width="140" align="right">
            <template #default="{ row }">
              <el-input-number v-if="row.is_leaf" v-model="row.opening_debit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="onBalanceChange(row)" />
              <span v-else>{{ fmt(row.opening_debit) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="期初贷方" width="140" align="right">
            <template #default="{ row }">
              <el-input-number v-if="row.is_leaf" v-model="row.opening_credit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="onBalanceChange(row)" />
              <span v-else>{{ fmt(row.opening_credit) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'

const { isMobile } = useMobile()
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 260px)' : 'calc(100vh - 300px)')

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const balances = ref([])
const saving = ref(false)

const totalDebit = computed(() => balances.value.filter(b => b.is_leaf).reduce((s, b) => s + (b.opening_debit || 0), 0))
const totalCredit = computed(() => balances.value.filter(b => b.is_leaf).reduce((s, b) => s + (b.opening_credit || 0), 0))
const isBalanced = computed(() => Math.abs(totalDebit.value - totalCredit.value) < 0.01)
const balanceDesc = computed(() => `借方合计：${fmt(totalDebit.value)}　｜　贷方合计：${fmt(totalCredit.value)}　｜　差额：${fmt(Math.abs(totalDebit.value - totalCredit.value))}`)

const importUrl = computed(() => currentBook.value ? `/api/books/${currentBook.value}/opening-balances/import` : '')
const uploadHeaders = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('token')}` }))

const loadData = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/opening-balances`)
  balances.value = data.data || []
}

const onBalanceChange = () => {}

const exportData = () => {
  const token = localStorage.getItem('token')
  window.open(`/api/books/${currentBook.value}/opening-balances/export?token=${token}`, '_blank')
}

const onImportSuccess = (resp) => {
  ElMessage.success(resp.message || '导入成功')
  loadData()
}

const onImportError = () => { ElMessage.error('导入失败') }

const saveAll = async () => {
  if (!isBalanced.value) { ElMessage.warning('借方与贷方不相等'); return }
  saving.value = true
  try {
    const payload = {
      balances: balances.value
        .filter(b => b.is_leaf && (b.opening_debit > 0 || b.opening_credit > 0))
        .map(b => ({ account_id: b.account_id, opening_debit: b.opening_debit || 0, opening_credit: b.opening_credit || 0 }))
    }
    await axios.post(`/api/books/${currentBook.value}/opening-balances`, payload)
    ElMessage.success('保存成功')
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
  finally { saving.value = false }
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const summaryMethod = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i <= 2) { sums[i] = ''; return }
    if (i === 3) sums[i] = fmt(data.filter(r => r.is_leaf).reduce((s, r) => s + (r.opening_debit || 0), 0))
    else if (i === 4) sums[i] = fmt(data.filter(r => r.is_leaf).reduce((s, r) => s + (r.opening_credit || 0), 0))
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
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
