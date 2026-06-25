<template>
  <div class="vouchers">
    <div class="page-header">
      <h2>凭证管理</h2>
      <div>
        <el-select v-model="currentBook" placeholder="选择账套" style="width: 200px; margin-right: 12px" @change="loadVouchers">
          <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
        </el-select>
        <el-button type="primary" @click="showEditor = true" :disabled="!currentBook">
          <el-icon><Plus /></el-icon>新增凭证
        </el-button>
      </div>
    </div>

    <!-- Filter bar -->
    <el-card shadow="never" style="margin-bottom: 16px" v-if="currentBook">
      <el-row :gutter="16">
        <el-col :span="6">
          <el-date-picker v-model="filterDateRange" type="daterange" range-separator="至"
            start-placeholder="开始日期" end-placeholder="结束日期" value-format="YYYY-MM-DD"
            @change="loadVouchers" style="width: 100%" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="filterStatus" placeholder="凭证状态" clearable @change="loadVouchers">
            <el-option label="草稿" value="draft" />
            <el-option label="已审核" value="reviewed" />
            <el-option label="已记账" value="posted" />
            <el-option label="已作废" value="voided" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-input v-model="filterMemo" placeholder="搜索摘要" clearable @change="loadVouchers" />
        </el-col>
      </el-row>
    </el-card>

    <!-- Voucher list -->
    <el-table :data="vouchers" stripe v-if="currentBook" style="width: 100%"
      @selection-change="handleSelectionChange" empty-text="暂无凭证">
      <el-table-column type="selection" width="40" />
      <el-table-column prop="number" label="凭证字号" width="140" />
      <el-table-column prop="date" label="日期" width="110" />
      <el-table-column label="摘要" min-width="200">
        <template #default="{ row }">
          {{ getVoucherMemo(row) }}
        </template>
      </el-table-column>
      <el-table-column prop="total_debit" label="借方金额" width="120" align="right">
        <template #default="{ row }">
          {{ formatMoney(row.total_debit) }}
        </template>
      </el-table-column>
      <el-table-column prop="total_credit" label="贷方金额" width="120" align="right">
        <template #default="{ row }">
          {{ formatMoney(row.total_credit) }}
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="viewVoucher(row)">查看</el-button>
          <el-button size="small" type="warning" link @click="editVoucher(row)" v-if="row.status === 'draft'">编辑</el-button>
          <el-button size="small" type="success" link @click="reviewVoucher(row)" v-if="row.status === 'draft'">审核</el-button>
          <el-button size="small" type="success" link @click="postVoucher(row)" v-if="row.status === 'reviewed' || row.status === 'draft'">记账</el-button>
          <el-button size="small" type="danger" link @click="deleteVoucher(row)" v-if="row.status === 'draft'">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Batch actions -->
    <div class="batch-actions" v-if="selectedVouchers.length > 0">
      <el-button @click="batchReview" :disabled="!canBatchReview">批量审核</el-button>
      <el-button @click="batchPost" :disabled="!canBatchPost">批量记账</el-button>
      <span style="margin-left: 12px; color: #909399">已选 {{ selectedVouchers.length }} 条</span>
    </div>

    <!-- Voucher Editor Dialog -->
    <el-dialog v-model="showEditor" :title="editingVoucher ? '编辑凭证' : '新增凭证'" width="900px" :close-on-click-modal="false">
      <el-form :model="voucherForm" label-width="80px">
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="日期">
              <el-date-picker v-model="voucherForm.date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="凭证类型">
              <el-select v-model="voucherForm.voucher_type" style="width: 100%">
                <el-option label="记账凭证" value="general" />
                <el-option label="收款凭证" value="receipt" />
                <el-option label="付款凭证" value="payment" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="附件">
              <el-input-number v-model="voucherForm.attachments" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <!-- Voucher items table -->
      <el-table :data="voucherForm.items" border size="small" style="margin: 12px 0">
        <el-table-column label="行号" width="50" type="index" />
        <el-table-column label="科目" min-width="200">
          <template #default="{ row, $index }">
            <el-select v-model="row.account_id" filterable placeholder="选择科目" @change="(val) => onAccountChange(val, $index)" style="width: 100%">
              <el-option v-for="a in accounts" :key="a.id" :label="a.code + ' ' + a.name" :value="a.id" :disabled="!a.is_leaf" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="摘要" min-width="160">
          <template #default="{ row }">
            <el-input v-model="row.memo" placeholder="摘要" />
          </template>
        </el-table-column>
        <el-table-column label="借方金额" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.debit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="calcTotal" />
          </template>
        </el-table-column>
        <el-table-column label="贷方金额" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.credit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="calcTotal" />
          </template>
        </el-table-column>
        <el-table-column label="" width="50">
          <template #default="{ $index }">
            <el-button size="small" type="danger" link @click="removeItem($index)" :disabled="voucherForm.items.length <= 2">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-button size="small" @click="addItem" style="margin-bottom: 12px">
        <el-icon><Plus /></el-icon>添加行
      </el-button>

      <!-- Totals -->
      <el-row :gutter="16" style="padding: 12px; background: #f5f7fa; border-radius: 4px">
        <el-col :span="8">
          <strong>借方合计：</strong>
          <span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalDebit) }}</span>
        </el-col>
        <el-col :span="8">
          <strong>贷方合计：</strong>
          <span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalCredit) }}</span>
        </el-col>
        <el-col :span="8">
          <strong>差额：</strong>
          <span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalDebit - totalCredit) }}</span>
          <el-icon v-if="totalDebit === totalCredit && totalDebit > 0" style="color: #67C23A; margin-left: 4px"><Check /></el-icon>
        </el-col>
      </el-row>

      <template #footer>
        <el-button @click="showEditor = false">取消</el-button>
        <el-button type="primary" @click="saveVoucher" :disabled="totalDebit !== totalCredit || totalDebit === 0">
          {{ editingVoucher ? '保存修改' : '保存凭证' }}
        </el-button>
        <el-button type="success" @click="saveAndNew" v-if="!editingVoucher" :disabled="totalDebit !== totalCredit || totalDebit === 0">
          保存并新增
        </el-button>
      </template>
    </el-dialog>

    <!-- View Voucher Dialog -->
    <el-dialog v-model="showViewer" title="凭证详情" width="800px">
      <div v-if="viewingVoucher">
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item label="凭证字号">{{ viewingVoucher.number }}</el-descriptions-item>
          <el-descriptions-item label="日期">{{ viewingVoucher.date }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusType(viewingVoucher.status)" size="small">{{ statusLabel(viewingVoucher.status) }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <el-table :data="viewingVoucher.items" border size="small" style="margin-top: 12px">
          <el-table-column label="行号" width="50" prop="line_no" />
          <el-table-column label="科目" min-width="200">
            <template #default="{ row }">{{ row.account_code }} {{ row.account_name }}</template>
          </el-table-column>
          <el-table-column label="摘要" min-width="150" prop="memo" />
          <el-table-column label="借方" width="120" align="right">
            <template #default="{ row }">{{ row.debit ? formatMoney(row.debit) : '' }}</template>
          </el-table-column>
          <el-table-column label="贷方" width="120" align="right">
            <template #default="{ row }">{{ row.credit ? formatMoney(row.credit) : '' }}</template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const books = ref([])
const currentBook = ref(null)
const vouchers = ref([])
const accounts = ref([])
const selectedVouchers = ref([])

// Filter
const filterDateRange = ref(null)
const filterStatus = ref('')
const filterMemo = ref('')

// Editor
const showEditor = ref(false)
const editingVoucher = ref(null)
const voucherForm = ref({ date: '', voucher_type: 'general', attachments: 0, items: [] })

// Viewer
const showViewer = ref(false)
const viewingVoucher = ref(null)

// Computed
const totalDebit = computed(() => voucherForm.value.items.reduce((s, i) => s + (i.debit || 0), 0))
const totalCredit = computed(() => voucherForm.value.items.reduce((s, i) => s + (i.credit || 0), 0))

const canBatchReview = computed(() => selectedVouchers.value.some(v => v.status === 'draft'))
const canBatchPost = computed(() => selectedVouchers.value.some(v => v.status === 'reviewed' || v.status === 'draft'))

// Load data
const loadBooks = async () => {
  try {
    const { data } = await axios.get('/api/books')
    books.value = data.data || []
    if (books.value.length > 0 && !currentBook.value) {
      currentBook.value = books.value[0].id
    }
  } catch (e) { console.error(e) }
}

const loadVouchers = async () => {
  if (!currentBook.value) return
  try {
    let url = `/api/books/${currentBook.value}/vouchers`
    const params = []
    if (filterStatus.value) params.push(`status=${filterStatus.value}`)
    if (filterDateRange.value) {
      params.push(`date_from=${filterDateRange.value[0]}`)
      params.push(`date_to=${filterDateRange.value[1]}`)
    }
    if (params.length) url += '?' + params.join('&')
    const { data } = await axios.get(url)
    vouchers.value = data.data || []
  } catch (e) { console.error(e) }
}

const loadAccounts = async () => {
  if (!currentBook.value) return
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/accounts`)
    accounts.value = data.data || []
  } catch (e) { console.error(e) }
}

// Voucher operations
const viewVoucher = async (row) => {
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    viewingVoucher.value = data.data
    showViewer.value = true
  } catch (e) { ElMessage.error('加载凭证失败') }
}

const editVoucher = async (row) => {
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    editingVoucher.value = data.data
    voucherForm.value = {
      date: data.data.date,
      voucher_type: data.data.voucher_type || 'general',
      attachments: data.data.attachments || 0,
      items: data.data.items.map(i => ({
        account_id: i.account_id,
        account_code: i.account_code,
        account_name: i.account_name,
        debit: i.debit,
        credit: i.credit,
        memo: i.memo
      }))
    }
    showEditor.value = true
  } catch (e) { ElMessage.error('加载凭证失败') }
}

const reviewVoucher = async (row) => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/review`)
    ElMessage.success('审核成功')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '审核失败') }
}

const postVoucher = async (row) => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/post`)
    ElMessage.success('记账成功')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '记账失败') }
}

const deleteVoucher = async (row) => {
  await ElMessageBox.confirm('确定删除该凭证？', '确认')
  try {
    await axios.delete(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    ElMessage.success('删除成功')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '删除失败') }
}

// Batch
const handleSelectionChange = (rows) => { selectedVouchers.value = rows }

const batchReview = async () => {
  const ids = selectedVouchers.value.filter(v => v.status === 'draft').map(v => v.id)
  if (!ids.length) return
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/batch-review`, { ids })
    ElMessage.success('批量审核成功')
    loadVouchers()
  } catch (e) { ElMessage.error('批量审核失败') }
}

const batchPost = async () => {
  const ids = selectedVouchers.value.filter(v => v.status === 'reviewed' || v.status === 'draft').map(v => v.id)
  if (!ids.length) return
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/batch-post`, { ids })
    ElMessage.success('批量记账成功')
    loadVouchers()
  } catch (e) { ElMessage.error('批量记账失败') }
}

// Editor helpers
const newItem = () => ({ account_id: null, account_code: '', account_name: '', debit: 0, credit: 0, memo: '' })

const addItem = () => { voucherForm.value.items.push(newItem()) }
const removeItem = (index) => { voucherForm.value.items.splice(index, 1) }

const onAccountChange = (val, index) => {
  const acct = accounts.value.find(a => a.id === val)
  if (acct) {
    voucherForm.value.items[index].account_code = acct.code
    voucherForm.value.items[index].account_name = acct.name
  }
}

const calcTotal = () => {}

const saveVoucher = async () => {
  try {
    const payload = { ...voucherForm.value }
    if (editingVoucher.value) {
      await axios.put(`/api/books/${currentBook.value}/vouchers/${editingVoucher.value.id}`, payload)
      ElMessage.success('修改成功')
    } else {
      await axios.post(`/api/books/${currentBook.value}/vouchers`, payload)
      ElMessage.success('保存成功')
    }
    showEditor.value = false
    editingVoucher.value = null
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const saveAndNew = async () => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers`, voucherForm.value)
    ElMessage.success('保存成功')
    resetForm()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const resetForm = () => {
  editingVoucher.value = null
  voucherForm.value = {
    date: new Date().toISOString().slice(0, 10),
    voucher_type: 'general',
    attachments: 0,
    items: [newItem(), newItem()]
  }
}

// Helpers
const formatMoney = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const statusType = (s) => ({ draft: 'info', reviewed: 'warning', posted: 'success', voided: 'danger' }[s] || 'info')
const statusLabel = (s) => ({ draft: '草稿', reviewed: '已审核', posted: '已记账', voided: '已作废' }[s] || s)

const getVoucherMemo = (row) => {
  // Try to get memo from first item (if loaded)
  return row.memo || ''
}

// Watch
watch(currentBook, () => {
  loadVouchers()
  loadAccounts()
})

watch(showEditor, (val) => {
  if (val && !editingVoucher.value) resetForm()
})

onMounted(() => {
  loadBooks()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-header h2 {
  color: #303133;
}

.batch-actions {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.text-danger {
  color: #f56c6c;
  font-weight: bold;
}
</style>
