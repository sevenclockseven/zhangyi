<template>
  <div class="assets">
    <div class="page-header">
      <h2>设备管理</h2>
      <div style="display:flex;gap:8px">
        <el-button type="success" @click="handleExport">导出</el-button>
        <el-button type="warning" @click="showImportDialog=true">导入</el-button>
        <el-button type="primary" @click="showCardDialog=true">
        <el-icon><Plus /></el-icon>新增资产
        </el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <!-- ========== 资产卡片 ========== -->
      <el-tab-pane label="资产卡片" name="cards">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-input v-model="keyword" placeholder="搜索名称/编号" style="width: 200px" clearable @keyup.enter="loadCards" />
          <el-select v-model="filterStatus" placeholder="状态" clearable style="width: 120px" @change="loadCards">
            <el-option label="在用" value="in_use" />
            <el-option label="闲置" value="idle" />
            <el-option label="维修" value="maintenance" />
            <el-option label="报废" value="scrapped" />
          </el-select>
          <el-button type="primary" @click="loadCards">查询</el-button>
        </div>
        <el-table :data="cards" border size="small" style="width: 100%">
          <el-table-column prop="code" label="编号" width="100" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="spec_model" label="规格" width="120" />
          <el-table-column label="原值" width="100" align="right">
            <template #default="{ row }">{{ fmt(row.original_value) }}</template>
          </el-table-column>
          <el-table-column label="累计折旧" width="100" align="right">
            <template #default="{ row }">{{ fmt(row.accumulated_depreciation) }}</template>
          </el-table-column>
          <el-table-column label="净值" width="100" align="right">
            <template #default="{ row }">{{ fmt(row.net_value) }}</template>
          </el-table-column>
          <el-table-column label="月折旧" width="90" align="right">
            <template #default="{ row }">{{ fmt(row.monthly_depreciation) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="70">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="部门" width="90" prop="department" />
          <el-table-column label="责任人" width="80" prop="employee_name" />
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="editCard(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="deleteCard(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="cards.length === 0" description="暂无资产卡片" />
      </el-tab-pane>

      <!-- ========== 分类管理 ========== -->
      <el-tab-pane label="资产分类" name="categories">
        <div style="margin-bottom: 12px">
          <el-button type="primary" @click="showCatDialog = true">新增分类</el-button>
        </div>
        <el-table :data="categories" border size="small" style="width: 100%">
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="name" label="名称" min-width="150" />
          <el-table-column label="折旧方法" width="100">
            <template #default="{ row }">{{ row.method === 'straight_line' ? '直线法' : row.method }}</template>
          </el-table-column>
          <el-table-column prop="useful_life_months" label="使用年限(月)" width="120" align="center" />
          <el-table-column label="净残值率" width="100" align="center">
            <template #default="{ row }">{{ (row.residual_value_rate * 100).toFixed(0) }}%</template>
          </el-table-column>
          <el-table-column prop="memo" label="备注" min-width="150" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="editCategory(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="deleteCategory(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- ========== 折旧计提 ========== -->
      <el-tab-pane label="折旧计提" name="depreciation">
        <div style="display: flex; gap: 12px; align-items: center; margin-bottom: 16px">
          <span>计提月份：</span>
          <el-date-picker v-model="depPeriod" type="month" value-format="YYYY-MM" placeholder="选择月份" />
          <el-button type="primary" @click="calcDep">试算折旧</el-button>
          <el-button type="success" @click="runDep" :disabled="depResults.length === 0">执行计提</el-button>
        </div>
        <el-table :data="depResults" border size="small" style="width: 100%">
          <el-table-column prop="card_name" label="资产名称" min-width="150" />
          <el-table-column label="月折旧额" width="120" align="right">
            <template #default="{ row }">
              <span v-if="!row.skipped">{{ fmt(row.amount) }}</span>
              <span v-else style="color: #999">-</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="150">
            <template #default="{ row }">
              <el-tag v-if="!row.skipped" type="success" size="small">可计提</el-tag>
              <el-tag v-else type="info" size="small">{{ row.reason }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="depTotal > 0" style="margin-top: 12px; font-weight: bold">
          本月应计提合计：{{ fmt(depTotal) }} 元
        </div>
      </el-tab-pane>

      <!-- ========== 台账汇总 ========== -->
      <el-tab-pane label="台账汇总" name="summary">
        <div class="summary-cards">
          <el-card shadow="hover" class="summary-card">
            <div class="summary-label">资产总数</div>
            <div class="summary-value">{{ summaryData.total_count || 0 }}</div>
          </el-card>
          <el-card shadow="hover" class="summary-card">
            <div class="summary-label">资产原值合计</div>
            <div class="summary-value">{{ fmt(summaryData.total_original_value) }}</div>
          </el-card>
          <el-card shadow="hover" class="summary-card">
            <div class="summary-label">资产净值合计</div>
            <div class="summary-value">{{ fmt(summaryData.total_net_value) }}</div>
          </el-card>
        </div>
        <el-table :data="summaryData.summary || []" border size="small" style="width: 100%; margin-top: 16px" show-summary>
          <el-table-column prop="category_name" label="分类" min-width="150" />
          <el-table-column prop="count" label="数量" width="80" align="center" />
          <el-table-column label="原值合计" width="140" align="right">
            <template #default="{ row }">{{ fmt(row.total_original_value) }}</template>
          </el-table-column>
          <el-table-column label="净值合计" width="140" align="right">
            <template #default="{ row }">{{ fmt(row.total_net_value) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="资产变动" name="transactions">
        <el-table :data="transactions" border size="small" style="width:100%">
          <el-table-column label="编号" width="100"><template #default="{ row }">{{ row.card_code||row.card_id }}</template></el-table-column>
          <el-table-column label="名称" width="120"><template #default="{ row }">{{ row.card_name||'-' }}</template></el-table-column>
          <el-table-column label="类型" width="90"><template #default="{ row }"><el-tag size="small">{{ transType(row.type) }}</el-tag></template></el-table-column>
          <el-table-column label="日期" width="100" prop="date" />
          <el-table-column label="变动前" width="110" align="right"><template #default="{ row }">{{ fmt(row.amount_before) }}</template></el-table-column>
          <el-table-column label="变动后" width="110" align="right"><template #default="{ row }">{{ fmt(row.amount_after) }}</template></el-table-column>
          <el-table-column label="备注" min-width="160" prop="note" />
        </el-table>
        <el-empty v-if="transactions.length===0" description="暂无变动记录" />
      </el-tab-pane>
    </el-tabs>

    <!-- 资产卡片 新增/编辑 弹窗 -->
    <el-dialog v-model="showCardDialog" :title="cardForm.id ? '编辑资产' : '新增资产'" width="600px">
      <el-form :model="cardForm" label-width="100px">
        <el-form-item label="资产编号" required>
          <el-input v-model="cardForm.code" placeholder="如 MBP001" />
        </el-form-item>
        <el-form-item label="资产名称" required>
          <el-input v-model="cardForm.name" />
        </el-form-item>
        <el-form-item label="规格型号">
          <el-input v-model="cardForm.spec_model" />
        </el-form-item>
        <el-form-item label="资产分类" required>
          <el-select v-model="cardForm.category_id" placeholder="选择分类" style="width: 100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.code + ' ' + c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="原值(元)" required>
          <el-input-number v-model="cardForm.original_value" :min="0" :precision="2" style="width: 100%" />
        </el-form-item>
        <el-form-item label="取得日期">
          <el-date-picker v-model="cardForm.acquisition_date" value-format="YYYY-MM-DD" format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="折旧起始月">
          <el-date-picker v-model="cardForm.depreciation_start_month" type="month" value-format="YYYY-MM" format="YYYY-MM" style="width: 100%" />
        </el-form-item>
        <el-form-item label="使用年限(月)">
          <el-input-number v-model="cardForm.useful_life_months" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="净残值率">
          <el-input-number v-model="cardForm.residual_value_rate" :min="0" :max="1" :precision="4" :step="0.01" style="width: 100%" />
          <span style="margin-left: 8px; color: #999">{{ ((cardForm.residual_value_rate || 0) * 100).toFixed(0) }}%</span>
        </el-form-item>
        <el-form-item label="使用部门">
          <el-input v-model="cardForm.department" />
        </el-form-item>
        <el-form-item label="责任人">
          <el-input v-model="cardForm.employee_name" />
        </el-form-item>
        <el-form-item label="存放地点">
          <el-input v-model="cardForm.location" />
        </el-form-item>
        <el-form-item label="来源">
          <el-select v-model="cardForm.source" placeholder="选择来源" clearable style="width: 100%">
            <el-option label="购入" value="purchase" />
            <el-option label="自建" value="self_made" />
            <el-option label="捐赠" value="donate" />
            <el-option label="调拨" value="transfer" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="cardForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCardDialog = false">取消</el-button>
        <el-button type="primary" @click="saveCard">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分类 新增/编辑 弹窗 -->
    <el-dialog v-model="showCatDialog" :title="catForm.id ? '编辑分类' : '新增分类'" width="500px">
      <el-form :model="catForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="catForm.name" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="catForm.code" placeholder="如 EQUIP" />
        </el-form-item>
        <el-form-item label="使用年限(月)">
          <el-input-number v-model="catForm.useful_life_months" :min="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="净残值率">
          <el-input-number v-model="catForm.residual_value_rate" :min="0" :max="1" :precision="4" :step="0.01" style="width: 100%" />
          <span style="margin-left: 8px; color: #999">{{ ((catForm.residual_value_rate || 0.05) * 100).toFixed(0) }}%</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="catForm.memo" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCatDialog = false">取消</el-button>
        <el-button type="primary" @click="saveCategory">保存</el-button>
      </template>
    </el-dialog>
  
    <el-dialog v-model="showChangeDialog" title="资产状态变更" width="450px">
      <el-form :model="changeForm" label-width="80px">
        <el-form-item label="当前状态"><el-tag :type="statusType(changeForm.current_status)" size="small">{{ statusText(changeForm.current_status) }}</el-tag></el-form-item>
        <el-form-item label="新状态" required><el-select v-model="changeForm.status" style="width:100%"><el-option label="在用" value="in_use"/><el-option label="闲置" value="idle"/><el-option label="维修" value="maintenance"/><el-option label="报废" value="scrapped"/></el-select></el-form-item>
        <el-form-item label="地点"><el-input v-model="changeForm.location" /></el-form-item>
        <el-form-item label="部门"><el-input v-model="changeForm.department" /></el-form-item>
        <el-form-item label="责任人"><el-input v-model="changeForm.employee_name" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="changeForm.note" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showChangeDialog=false">取消</el-button><el-button type="primary" @click="saveStatus">确认</el-button></template>
    </el-dialog>
    <el-dialog v-model="showImportDialog" title="批量导入" width="600px">
      <el-alert type="info" :closable="false" style="margin-bottom:12px">粘贴JSON数组，每项含 code,name,category_id,original_value</el-alert>
      <el-input v-model="importJson" type="textarea" :rows="8" placeholder='[{"code":"A001","name":"笔记本","category_id":1,"original_value":8000}]' />
      <template #footer><el-button @click="showImportDialog=false">取消</el-button><el-button type="primary" @click="handleImport">导入</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { assetApi } from '../api'

const route = useRoute()
const bookId = route.query.book || route.params.id || 1

const activeTab = ref('cards')
const loading = ref(false)
const keyword = ref('')
const filterStatus = ref('')

const cards = ref([])
const categories = ref([])
const depResults = ref([])
const depPeriod = ref(new Date().toISOString().slice(0, 7))
const depTotal = ref(0)
const summaryData = ref({})

const showCardDialog = ref(false)
const cardForm = reactive({
  id: null, code: '', name: '', spec_model: '', category_id: null,
  original_value: 0, acquisition_date: '', depreciation_start_month: '',
  useful_life_months: 36, residual_value_rate: 0.05,
  department: '', employee_name: '', location: '',
  source: 'purchase', remark: ''
})

const showCatDialog = ref(false)
const catForm = reactive({
  id: null, name: '', code: '', useful_life_months: 60,
  residual_value_rate: 0.05, memo: ''
})

const transactions = ref([])
const showImportDialog = ref(false)
const importJson = ref('')
const showChangeDialog = ref(false)
const changeForm = reactive({ card_id:null, current_status:'', status:'', location:'', department:'', employee_name:'', note:'' })

function fmt(v) {
  return v == null ? '-' : Number(v).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function statusText(s) {
  return { in_use: '在用', idle: '闲置', maintenance: '维修', scrapped: '报废' }[s] || s
}
function statusType(s) {
  return { in_use: 'success', idle: 'info', maintenance: 'warning', scrapped: 'danger' }[s] || ''
}
function transType(t) {
  return { acquire: '购入', transfer: '调拨', depreciate: '折旧', maintenance: '维修', scrap: '报废', idle: '闲置' }[t] || t
}

async function loadCards() {
  loading.value = true
  try {
    const params = {}
    if (keyword.value) params.keyword = keyword.value
    if (filterStatus.value) params.status = filterStatus.value
    const { data } = await assetApi.listCards(bookId, params)
    cards.value = data.data || []
  } finally { loading.value = false }
}

async function loadCategories() {
  const { data } = await assetApi.listCategories(bookId)
  categories.value = data.data || []
}

async function loadSummary() {
  const { data } = await assetApi.summary(bookId)
  summaryData.value = data
}
async function loadTransactions() {
  const { data } = await assetApi.getAllTransactions(bookId)
  transactions.value = data.data || []
}

function editCard(row) {
  Object.assign(cardForm, row)
  showCardDialog.value = true
}

async function deleteCard(row) {
  await ElMessageBox.confirm(`确定删除资产"${row.name}"？`, '确认', { type: 'warning' })
  await assetApi.deleteCard(bookId, row.id)
  ElMessage.success('已删除')
  loadCards()
}

async function saveCard() {
  if (!cardForm.code || !cardForm.name || !cardForm.category_id || !cardForm.original_value) {
    ElMessage.warning('请填写必填项')
    return
  }
  if (cardForm.id) {
    await assetApi.updateCard(bookId, cardForm.id, cardForm)
    ElMessage.success('已更新')
  } else {
    await assetApi.createCard(bookId, cardForm)
    ElMessage.success('已创建')
  }
  showCardDialog.value = false
  loadCards()
}

function editCategory(row) {
  Object.assign(catForm, row)
  showCatDialog.value = true
}

async function deleteCategory(row) {
  await ElMessageBox.confirm(`确定删除分类"${row.name}"？`, '确认', { type: 'warning' })
  await assetApi.deleteCategory(bookId, row.id)
  ElMessage.success('已删除')
  loadCategories()
}

async function saveCategory() {
  if (!catForm.name) { ElMessage.warning('请填写名称'); return }
  if (catForm.id) {
    await assetApi.updateCategory(bookId, catForm.id, catForm)
    ElMessage.success('已更新')
  } else {
    await assetApi.createCategory(bookId, catForm)
    ElMessage.success('已创建')
  }
  showCatDialog.value = false
  loadCategories()
}

async function calcDep() {
  const { data } = await assetApi.calcDepreciation(bookId, depPeriod.value)
  depResults.value = data.data || []
  depTotal.value = depResults.value.reduce((sum, r) => sum + (r.amount || 0), 0)
}

async function runDep() {
  await ElMessageBox.confirm(`确定对 ${depPeriod.value} 执行折旧计提？此操作不可撤销。`, '确认', { type: 'warning' })
  const { data } = await assetApi.runDepreciation(bookId, depPeriod.value)
  ElMessage.success(`计提完成，共 ${data.count} 项资产，合计 ${fmt(data.total)} 元`)
  depResults.value = []
  depTotal.value = 0
  loadCards()
  loadSummary()
}

function openChangeStatus(row) {
  Object.assign(changeForm, { card_id: row.id, current_status: row.status, status: '', location:'', department:'', employee_name:'', note:'' })
  showChangeDialog.value = true
}
async function saveStatus() {
  if (!changeForm.status) { ElMessage.warning('请选择新状态'); return }
  if (changeForm.status === changeForm.current_status) { ElMessage.warning('状态未变化'); return }
  const p = { status: changeForm.status }
  if (changeForm.location) p.location = changeForm.location
  if (changeForm.department) p.department = changeForm.department
  if (changeForm.employee_name) p.employee_name = changeForm.employee_name
  if (changeForm.note) p.note = changeForm.note
  await assetApi.changeStatus(bookId, changeForm.card_id, p)
  ElMessage.success('状态变更成功')
  showChangeDialog.value = false
  loadCards(); loadTransactions()
}
async function handleExport() {
  const { data } = await assetApi.exportAssets(bookId)
  const items = data.data || []
  if (!items.length) { ElMessage.warning('暂无资产'); return }
  const H = ['编号','名称','规格','分类','原值','累计折旧','净值','月折旧','状态','部门','责任人','存放地点','取得日期','折旧起始月','使用年限','净残值率','来源','备注']
  const F = ['code','name','spec_model','category_name','original_value','accumulated_depreciation','net_value','monthly_depreciation','status','department','employee_name','location','acquisition_date','depreciation_start_month','useful_life_months','residual_value_rate','source','remark']
  let csv = H.join('	') + '\n'
  for (const it of items) csv += F.map(f => it[f] ?? '').join('	') + '\n'
  const blob = new Blob(['﻿'+csv], {type:'text/csv;charset=utf-8'})
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = '资产清单_' + new Date().toISOString().slice(0,10) + '.csv'
  a.click(); URL.revokeObjectURL(a.href)
  ElMessage.success('已导出 ' + items.length + ' 条')
}
async function handleImport() {
  if (!importJson.value.trim()) { ElMessage.warning('请输入JSON'); return }
  let items
  try { items = JSON.parse(importJson.value); if (!Array.isArray(items)) throw new Error('必须是数组') }
  catch(e) { ElMessage.error('JSON错误: ' + e.message); return }
  if (!items.length) { ElMessage.warning('数据为空'); return }
  const { data } = await assetApi.importAssets(bookId, { items })
  if (data.errors && data.errors.length) ElMessage.warning('导入 ' + data.imported + '/' + data.total + '，失败 ' + data.errors.length + ' 条')
  else ElMessage.success('导入成功 ' + data.imported + ' 条')
  showImportDialog.value = false; importJson.value = ''
  loadCards(); loadSummary(); loadTransactions()
}

onMounted(() => {
  loadCards()
  loadCategories()
  loadSummary()
  loadTransactions()
})
</script>

<style scoped>
.assets { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.page-header h2 { color: #303133; font-size: 18px; }

.summary-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 16px; }
.summary-card :deep(.el-card__body) { text-align: center; padding: 20px; }
.summary-label { color: #909399; font-size: 13px; margin-bottom: 8px; }
.summary-value { font-size: 24px; font-weight: bold; color: #303133; }
</style>
