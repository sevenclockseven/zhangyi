<template>
  <div class="vouchers">
    <div class="page-header">
      <h2>凭证管理</h2>
      <div class="header-actions">
        <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px', marginRight: isMobile ? '0' : '12px', marginBottom: isMobile ? '8px' : '0' }" @change="loadVouchers">
          <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
        </el-select>
        <el-button type="primary" @click="showEditor = true" :disabled="!currentBook">
          <el-icon><Plus /></el-icon>新增凭证
        </el-button>
      </div>
    </div>

    <!-- Filter bar -->
    <el-card shadow="never" style="margin-bottom: 12px" v-if="currentBook">
      <div :class="isMobile ? 'filter-mobile' : ''">
        <el-date-picker v-model="filterDateRange" type="daterange" range-separator="至"
          start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD"
          @change="loadVouchers" :style="{ width: isMobile ? '100%' : '300px', marginBottom: isMobile ? '8px' : '0' }" />
        <el-select v-model="filterStatus" placeholder="状态" clearable @change="loadVouchers" :style="{ width: isMobile ? '100%' : '120px', marginBottom: isMobile ? '8px' : '0' }">
          <el-option label="草稿" value="draft" />
          <el-option label="已审核" value="reviewed" />
          <el-option label="已记账" value="posted" />
          <el-option label="已作废" value="voided" />
        </el-select>
        <el-input v-if="!isMobile" v-model="filterKeyword" placeholder="搜索凭证号/摘要" clearable @change="loadVouchers" style="width: 200px" />
      </div>
    </el-card>

    <!-- Voucher list - scrollable table -->
    <div class="table-wrapper">
      <el-table :data="vouchers" stripe v-if="currentBook" style="width: 100%"
        @selection-change="handleSelectionChange" empty-text="暂无凭证" :max-height="tableMaxHeight">
        <el-table-column type="selection" width="40" />
        <el-table-column prop="number" label="凭证字号" :width="isMobile ? 100 : 140" />
        <el-table-column prop="date" label="日期" :width="isMobile ? 90 : 110" />
        <el-table-column label="摘要" min-width="150">
          <template #default="{ row }">
            {{ getVoucherMemo(row) }}
          </template>
        </el-table-column>
        <el-table-column prop="total_debit" label="借方" :width="isMobile ? 100 : 120" align="right">
          <template #default="{ row }">
            {{ formatMoney(row.total_debit) }}
          </template>
        </el-table-column>
        <el-table-column prop="total_credit" label="贷方" :width="isMobile ? 100 : 120" align="right">
          <template #default="{ row }">
            {{ formatMoney(row.total_credit) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" :width="isMobile ? 70 : 90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" :width="isMobile ? 150 : 220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="viewVoucher(row)">查看</el-button>
            <el-button size="small" type="warning" link @click="editVoucher(row)" v-if="row.status === 'draft'">编辑</el-button>
            <el-button size="small" type="success" link @click="reviewVoucher(row)" v-if="row.status === 'draft'">审核</el-button>
            <el-button size="small" type="success" link @click="postVoucher(row)" v-if="row.status === 'reviewed' || row.status === 'draft'">记账</el-button>
            <el-button size="small" type="warning" link @click="unreviewVoucher(row)" v-if="row.status === 'reviewed'">反审核</el-button>
            <el-button size="small" type="warning" link @click="unpostVoucher(row)" v-if="row.status === 'posted'">反记账</el-button>
            <el-button size="small" type="danger" link @click="voidVoucher(row)" v-if="row.status === 'draft' || row.status === 'reviewed'">作废</el-button>
            <el-button size="small" type="info" link @click="restoreVoucher(row)" v-if="row.status === 'voided'">恢复</el-button>
            <el-button size="small" type="danger" link @click="deleteVoucher(row)" v-if="row.status === 'draft'">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Batch actions -->
    <div class="batch-actions" v-if="selectedVouchers.length > 0">
      <el-button @click="batchReview" :disabled="!canBatchReview">批量审核</el-button>
      <el-button @click="batchPost" :disabled="!canBatchPost">批量记账</el-button>
      <span style="margin-left: 12px; color: #909399">已选 {{ selectedVouchers.length }} 条</span>
    </div>

    <!-- Voucher Editor Dialog -->
    <el-dialog v-model="showEditor" :title="editingVoucher ? '编辑凭证' : '新增凭证'" :width="isMobile ? '98%' : '900px'" :close-on-click-modal="false" fullscreen>
      <el-form :model="voucherForm" :label-width="isMobile ? '60px' : '80px'" size="small">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="8">
            <el-form-item label="日期">
              <el-date-picker v-model="voucherForm.date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-form-item label="类型">
              <el-select v-model="voucherForm.voucher_type" style="width: 100%">
                <el-option label="记账凭证" value="general" />
                <el-option label="收款凭证" value="receipt" />
                <el-option label="付款凭证" value="payment" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-form-item label="附件">
              <el-input-number v-model="voucherForm.attachments" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <!-- Voucher items - scrollable -->
      <div class="table-wrapper">
        <el-table :data="voucherForm.items" border size="small" style="margin: 8px 0">
          <el-table-column label="#" width="40" type="index" />
          <el-table-column label="摘要" min-width="150">
            <template #default="{ row, $index }">
              <el-input v-model="row.memo" placeholder="摘要" size="small" @keydown.enter.prevent="focusNext($index, 'account')" :ref="el => setFieldRef($index, 'memo', el)" />
            </template>
          </el-table-column>
          <el-table-column label="科目" min-width="180">
            <template #default="{ row, $index }">
              <el-select v-model="row.account_id" filterable placeholder="选择科目" @change="(val) => onAccountChange(val, $index)" @keydown.enter.prevent="focusNext($index, 'debit')" :ref="el => setFieldRef($index, 'account', el)" style="width: 100%">
                <el-option v-for="a in accounts" :key="a.id" :label="a.code + ' ' + a.name" :value="a.id" :disabled="!a.is_leaf" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="借方" width="120">
            <template #default="{ row }">
              <el-input-number v-model="row.debit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="calcTotal" @keydown.enter.prevent="focusNext($index, 'credit')" :ref="el => setFieldRef($index, 'debit', el)" />
            </template>
          </el-table-column>
          <el-table-column label="贷方" width="120">
            <template #default="{ row }">
              <el-input-number v-model="row.credit" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" @change="calcTotal" @keydown.enter.prevent="focusNext($index, 'next-row')" :ref="el => setFieldRef($index, 'credit', el)" />
            </template>
          </el-table-column>
          <el-table-column label="" width="40">
            <template #default="{ $index }">
              <el-button size="small" type="danger" link @click="removeItem($index)" :disabled="voucherForm.items.length <= 2">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <el-button size="small" @click="addItem" style="margin-bottom: 8px">
        <el-icon><Plus /></el-icon>添加行
      </el-button>

      <!-- Totals -->
      <div class="totals-bar">
        <div><strong>借方：</strong><span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalDebit) }}</span></div>
        <div><strong>贷方：</strong><span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalCredit) }}</span></div>
        <div>
          <strong>差额：</strong>
          <span :class="{ 'text-danger': totalDebit !== totalCredit }">{{ formatMoney(totalDebit - totalCredit) }}</span>
          <el-icon v-if="totalDebit === totalCredit && totalDebit > 0" style="color: #67C23A; margin-left: 4px"><Check /></el-icon>
        </div>
      </div>

      <template #footer>
        <div class="editor-footer">
          <el-button @click="showEditor = false">取消</el-button>
          <el-button type="primary" @click="saveVoucher" :disabled="totalDebit !== totalCredit || totalDebit === 0">
            {{ editingVoucher ? '保存' : '保存' }}
          </el-button>
          <el-button type="success" @click="saveAndNew" v-if="!editingVoucher" :disabled="totalDebit !== totalCredit || totalDebit === 0">
            保存并新增
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- View Voucher Dialog -->
    <el-dialog v-model="showViewer" title="凭证详情" :width="isMobile ? '98%' : '800px'">
      <div v-if="viewingVoucher">
        <el-descriptions :column="isMobile ? 1 : 3" border size="small">
          <el-descriptions-item label="凭证字号">{{ viewingVoucher.number }}</el-descriptions-item>
          <el-descriptions-item label="日期">{{ viewingVoucher.date }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusType(viewingVoucher.status)" size="small">{{ statusLabel(viewingVoucher.status) }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <div class="table-wrapper" style="margin-top: 12px">
          <el-table :data="viewingVoucher.items" border size="small">
            <el-table-column label="#" width="40" prop="line_no" />
            <el-table-column label="科目" min-width="150">
              <template #default="{ row }">{{ row.account_code }} {{ row.account_name }}</template>
            </el-table-column>
            <el-table-column label="摘要" min-width="100" prop="memo" />
            <el-table-column label="借方" width="100" align="right">
              <template #default="{ row }">{{ row.debit ? formatMoney(row.debit) : '' }}</template>
            </el-table-column>
            <el-table-column label="贷方" width="100" align="right">
              <template #default="{ row }">{{ row.credit ? formatMoney(row.credit) : '' }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 300px)' : 'calc(100vh - 350px)')

const books = ref([])
const currentBook = ref(null)
const vouchers = ref([])
const accounts = ref([])
const selectedVouchers = ref([])

const filterDateRange = ref(null)
const filterStatus = ref('')
const filterKeyword = ref('')

const showEditor = ref(false)
const editingVoucher = ref(null)
const voucherForm = ref({ date: '', voucher_type: 'general', attachments: 0, items: [] })

const showViewer = ref(false)
const viewingVoucher = ref(null)

const totalDebit = computed(() => voucherForm.value.items.reduce((s, i) => s + (i.debit || 0), 0))
const totalCredit = computed(() => voucherForm.value.items.reduce((s, i) => s + (i.credit || 0), 0))

const canBatchReview = computed(() => selectedVouchers.value.some(v => v.status === 'draft'))
const canBatchPost = computed(() => selectedVouchers.value.some(v => v.status === 'reviewed' || v.status === 'draft'))

const loadBooks = async () => {
  try {
    const { data } = await axios.get('/api/books')
    books.value = data.data || []
    if (books.value.length > 0 && !currentBook.value) currentBook.value = books.value[0].id
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

const viewVoucher = async (row) => {
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    viewingVoucher.value = data.data
    showViewer.value = true
  } catch (e) { ElMessage.error('加载失败') }
}

const editVoucher = async (row) => {
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    editingVoucher.value = data.data
    voucherForm.value = {
      date: data.data.date,
      voucher_type: data.data.voucher_type || 'general',
      attachments: data.data.attachments || 0,
      items: data.data.items.map(i => ({ account_id: i.account_id, account_code: i.account_code, account_name: i.account_name, debit: i.debit, credit: i.credit, memo: i.memo }))
    }
    showEditor.value = true
  } catch (e) { ElMessage.error('加载失败') }
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
  await ElMessageBox.confirm('确定删除？', '确认')
  try {
    await axios.delete(`/api/books/${currentBook.value}/vouchers/${row.id}`)
    ElMessage.success('已删除')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '删除失败') }
}

const unreviewVoucher = async (row) => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/unreview`)
    ElMessage.success('反审核成功')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '反审核失败') }
}

const unpostVoucher = async (row) => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/unpost`)
    ElMessage.success('反记账成功')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '反记账失败') }
}

const voidVoucher = async (row) => {
  await ElMessageBox.confirm('确定作废该凭证？', '确认', { type: 'warning' })
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/void`)
    ElMessage.success('已作废')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '作废失败') }
}

const restoreVoucher = async (row) => {
  try {
    await axios.post(`/api/books/${currentBook.value}/vouchers/${row.id}/restore`)
    ElMessage.success('已恢复')
    loadVouchers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '恢复失败') }
}

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

// Field refs for Enter navigation
const fieldRefs = {}
const setFieldRef = (rowIndex, field, el) => {
  const key = `${rowIndex}_${field}`
  if (el) fieldRefs[key] = el
}

const focusNext = (rowIndex, field) => {
  if (field === 'next-row') {
    // Jump to next row's memo
    const nextKey = `${rowIndex + 1}_memo`
    if (fieldRefs[nextKey]) {
      fieldRefs[nextKey].focus?.() || fieldRefs[nextKey].$el?.querySelector('input')?.focus()
    } else {
      // Add new row if at the end
      addItem()
      nextTick(() => {
        setTimeout(() => {
          const newKey = `${voucherForm.value.items.length - 1}_account`
          fieldRefs[newKey]?.focus?.() || fieldRefs[newKey]?.$el?.querySelector('input')?.focus()
        }, 100)
      })
    }
  } else {
    const key = `${rowIndex}_${field}`
    const ref = fieldRefs[key]
    if (ref) {
      ref.focus?.() || ref.$el?.querySelector('input')?.focus()
    }
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
  voucherForm.value = { date: new Date().toISOString().slice(0, 10), voucher_type: 'general', attachments: 0, items: [newItem(), newItem()] }
}

const formatMoney = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const statusType = (s) => ({ draft: 'info', reviewed: 'warning', posted: 'success', voided: 'danger' }[s] || 'info')
const statusLabel = (s) => ({ draft: '草稿', reviewed: '已审核', posted: '已记账', voided: '已作废' }[s] || s)
const getVoucherMemo = (row) => {
  if (row.memo) return row.memo
  if (row.items && row.items.length > 0) return row.items[0].memo || ''
  return ''
}

watch(currentBook, () => { loadVouchers(); loadAccounts() })
watch(showEditor, (val) => { if (val && !editingVoucher.value) resetForm() })

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}

.page-header h2 { color: #303133; font-size: 18px; }

.header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-mobile {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.batch-actions {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.totals-bar {
  display: flex;
  gap: 16px;
  padding: 10px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 14px;
  flex-wrap: wrap;
}

.editor-footer {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.text-danger { color: #f56c6c; font-weight: bold; }

@media (max-width: 767px) {
  .page-header {
    flex-direction: column;
  }
  .header-actions {
    width: 100%;
  }
  .header-actions .el-select {
    flex: 1;
  }
}
</style>
