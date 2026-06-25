<template>
  <div class="opening-balance">
    <div class="page-header">
      <h2>期初余额</h2>
      <div class="header-actions">
        <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="loadData">
          <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
        </el-select>
      </div>
    </div>

    <div v-if="currentBook">
      <!-- 试算平衡提示 -->
      <el-alert
        :title="isBalanced ? '试算平衡 ✓' : '试算不平衡 ✗'"
        :description="`借方合计：${fmt(totalDebit)}　｜　贷方合计：${fmt(totalCredit)}　｜　差额：${fmt(Math.abs(totalDebit - totalCredit))}`"
        :type="isBalanced ? 'success' : 'error'"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />

      <div class="toolbar">
        <el-button type="primary" size="small" @click="saveAll" :loading="saving" :disabled="!isBalanced">
          <el-icon><Check /></el-icon>保存期初余额
        </el-button>
        <el-button size="small" @click="loadData">
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
              <el-input-number
                v-if="row.is_leaf"
                v-model="row.opening_debit"
                :min="0"
                :precision="2"
                :controls="false"
                size="small"
                style="width: 100%"
                @change="onBalanceChange(row)"
              />
              <span v-else>{{ fmt(row.opening_debit) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="期初贷方" width="140" align="right">
            <template #default="{ row }">
              <el-input-number
                v-if="row.is_leaf"
                v-model="row.opening_credit"
                :min="0"
                :precision="2"
                :controls="false"
                size="small"
                style="width: 100%"
                @change="onBalanceChange(row)"
              />
              <span v-else>{{ fmt(row.opening_credit) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 260px)' : 'calc(100vh - 300px)')

const books = ref([])
const currentBook = ref(null)
const balances = ref([])
const saving = ref(false)

const totalDebit = computed(() => {
  return balances.value
    .filter(b => b.is_leaf)
    .reduce((sum, b) => sum + (b.opening_debit || 0), 0)
})

const totalCredit = computed(() => {
  return balances.value
    .filter(b => b.is_leaf)
    .reduce((sum, b) => sum + (b.opening_credit || 0), 0)
})

const isBalanced = computed(() => {
  return Math.abs(totalDebit.value - totalCredit.value) < 0.01
})

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
}

const loadData = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/opening-balances`)
  balances.value = data.data || []
}

// Propagate parent account totals
const onBalanceChange = (row) => {
  // Find parent and recalculate
  recalcParents()
}

const recalcParents = () => {
  // Group by parent_code and sum children
  const parentMap = {}
  balances.value.forEach(b => {
    if (b.account_code.includes('.')) {
      const parentCode = b.account_code.split('.').slice(0, -1).join('.')
      if (!parentMap[parentCode]) parentMap[parentCode] = { debit: 0, credit: 0 }
      parentMap[parentCode].debit += b.opening_debit || 0
      parentMap[parentCode].credit += b.opening_credit || 0
    }
  })
  balances.value.forEach(b => {
    if (parentMap[b.account_code]) {
      b.opening_debit = parentMap[b.account_code].debit
      b.opening_credit = parentMap[b.account_code].credit
    }
  })
}

const saveAll = async () => {
  if (!isBalanced.value) {
    ElMessage.warning('借方与贷方不相等，无法保存')
    return
  }
  saving.value = true
  try {
    const payload = {
      balances: balances.value
        .filter(b => b.is_leaf && (b.opening_debit > 0 || b.opening_credit > 0))
        .map(b => ({
          account_id: b.account_id,
          opening_debit: b.opening_debit || 0,
          opening_credit: b.opening_credit || 0
        }))
    }
    await axios.post(`/api/books/${currentBook.value}/opening-balances`, payload)
    ElMessage.success('期初余额保存成功')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
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
    if (i === 3) {
      sums[i] = fmt(data.filter(r => r.is_leaf).reduce((s, r) => s + (r.opening_debit || 0), 0))
    } else if (i === 4) {
      sums[i] = fmt(data.filter(r => r.is_leaf).reduce((s, r) => s + (r.opening_credit || 0), 0))
    }
  })
  return sums
}

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
