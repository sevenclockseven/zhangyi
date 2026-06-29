#!/usr/bin/env python3
"""Patch Assets.vue: add transaction tab, import/export, status change"""
import sys

path = '/opt/zhangyi/web/src/views/Assets.vue'
with open(path, 'r') as f:
    lines = f.readlines()

content = ''.join(lines)
original_len = len(content)

# ===== 1. Fix page-header: add import/export buttons =====
# Find the page-header div and add buttons before the primary button
old_header = '''      <h2>设备管理</h2>
      <el-button type="primary" @click="showCardDialog = true">
        <el-icon><Plus /></el-icon>新增资产
      </el-button>'''

new_header = '''      <h2>设备管理</h2>
      <div style="display: flex; gap: 8px">
        <el-button type="success" @click="handleExport">导出</el-button>
        <el-button type="warning" @click="showImportDialog = true">导入</el-button>
        <el-button type="primary" @click="showCardDialog = true">
          <el-icon><Plus /></el-icon>新增资产
        </el-button>
      </div>'''

content = content.replace(old_header, new_header, 1)

# ===== 2. Fix operation column: add "变更" button =====
old_ops = '''          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="editCard(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="deleteCard(row)">删除</el-button>
            </template>
          </el-table-column>'''

new_ops = '''          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="editCard(row)">编辑</el-button>
              <el-button size="small" type="warning" link @click="openChangeStatus(row)">变更</el-button>
              <el-button size="small" type="danger" link @click="deleteCard(row)">删除</el-button>
            </template>
          </el-table-column>'''

content = content.replace(old_ops, new_ops, 1)

# ===== 3. Change el-option label from template literal to static =====
content = content.replace(
    ':label="`${c.code} ${c.name}`"',
    ':label="c.code + \\' \\' + c.name"'
)

# ===== 4. Add transaction tab BEFORE the closing </el-tabs> =====
# Find the end of summary tab (the last tab before </el-tabs>)
trans_tab = '''
      <!-- ========== 资产变动 ========== -->
      <el-tab-pane label="资产变动" name="transactions">
        <el-table :data="transactions" border size="small" style="width: 100%">
          <el-table-column label="资产编号" width="100">
            <template #default="{ row }">{{ row.card_code || row.card_id }}</template>
          </el-table-column>
          <el-table-column label="资产名称" width="120">
            <template #default="{ row }">{{ row.card_name || '-' }}</template>
          </el-table-column>
          <el-table-column label="变动类型" width="90">
            <template #default="{ row }">
              <el-tag size="small">{{ transTypeText(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="变动日期" width="100" prop="date" />
          <el-table-column label="变动前金额" width="110" align="right">
            <template #default="{ row }">{{ fmt(row.amount_before) }}</template>
          </el-table-column>
          <el-table-column label="变动后金额" width="110" align="right">
            <template #default="{ row }">{{ fmt(row.amount_after) }}</template>
          </el-table-column>
          <el-table-column label="备注" min-width="160" prop="note" />
          <el-table-column label="时间" width="140">
            <template #default="{ row }">{{ row.created_at ? row.created_at.replace('T', ' ').slice(0, 16) : '-' }}</template>
          </el-table-column>
        </el-table>
        <el-empty v-if="transactions.length === 0" description="暂无变动记录" />
      </el-tab-pane>'''

# Insert before closing </el-tabs> -- find the </el-tabs> that closes el-tabs
# The summary tab's closing </el-tab-pane> is followed by </el-tabs>
old_tabs_close = '''        </el-table>
      </el-tab-pane>
    </el-tabs>'''

new_tabs_close = '''        </el-table>
      </el-tab-pane>
''' + trans_tab + '''
    </el-tabs>'''

content = content.replace(old_tabs_close, new_tabs_close, 1)

# ===== 5. Add new dialogs before closing </div> (end of template) =====
dialogs = '''
    <!-- 状态变更弹窗 -->
    <el-dialog v-model="showChangeDialog" title="资产状态变更" width="450px">
      <el-form :model="changeForm" label-width="80px">
        <el-form-item label="当前状态">
          <el-tag :type="statusType(changeForm.current_status)" size="small">{{ statusText(changeForm.current_status) }}</el-tag>
        </el-form-item>
        <el-form-item label="新状态" required>
          <el-select v-model="changeForm.status" placeholder="选择新状态" style="width: 100%">
            <el-option label="在用" value="in_use" />
            <el-option label="闲置" value="idle" />
            <el-option label="维修" value="maintenance" />
            <el-option label="报废" value="scrapped" />
          </el-select>
        </el-form-item>
        <el-form-item label="存放地点">
          <el-input v-model="changeForm.location" placeholder="不修改可不填" />
        </el-form-item>
        <el-form-item label="使用部门">
          <el-input v-model="changeForm.department" placeholder="不修改可不填" />
        </el-form-item>
        <el-form-item label="责任人">
          <el-input v-model="changeForm.employee_name" placeholder="不修改可不填" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="changeForm.note" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showChangeDialog = false">取消</el-button>
        <el-button type="primary" @click="saveStatus">确认变更</el-button>
      </template>
    </el-dialog>

    <!-- 导入弹窗 -->
    <el-dialog v-model="showImportDialog" title="批量导入资产" width="600px">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px">
        请粘贴 JSON 数组，每个元素包含 code, name, category_id, original_value 字段
      </el-alert>
      <el-input v-model="importJson" type="textarea" :rows="8"
        placeholder='[{"code":"A001","name":"笔记本电脑","category_id":1,"original_value":8000}]' />
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="handleImport">导入</el-button>
      </template>
    </el-dialog>'''

# Insert before the last </div> (which closes <div class="assets">)
last_div = content.rfind('</div>')
content = content[:last_div] + dialogs + '\n  ' + content[last_div:]

# ===== 6. Add new reactive state and methods to script setup =====
# Find where to insert new reactive vars -- after catForm
old_script = '''const catForm = reactive({
  id: null, name: '', code: '', useful_life_months: 60,
  residual_value_rate: 0.05, memo: ''
})'''

new_script = '''const catForm = reactive({
  id: null, name: '', code: '', useful_life_months: 60,
  residual_value_rate: 0.05, memo: ''
})

const transactions = ref([])
const showImportDialog = ref(false)
const importJson = ref('')
const showChangeDialog = ref(false)
const changeForm = reactive({
  card_id: null,
  current_status: '',
  status: '',
  location: '',
  department: '',
  employee_name: '',
  note: ''
})'''

content = content.replace(old_script, new_script, 1)

# Add helper function after statusType
old_helpers = '''function statusType(s) {
  return { in_use: 'success', idle: 'info', maintenance: 'warning', scrapped: 'danger' }[s] || ''
}'''

new_helpers = '''function statusType(s) {
  return { in_use: 'success', idle: 'info', maintenance: 'warning', scrapped: 'danger' }[s] || ''
}
function transTypeText(t) {
  return { acquire: '购入', transfer: '调拨', depreciate: '折旧', maintenance: '维修', scrap: '报废', idle: '闲置' }[t] || t
}'''

content = content.replace(old_helpers, new_helpers, 1)

# Add loadTransactions function after loadSummary
old_load_sum = '''async function loadSummary() {
  const { data } = await assetApi.summary(bookId)
  summaryData.value = data
}'''

new_load_sum = '''async function loadSummary() {
  const { data } = await assetApi.summary(bookId)
  summaryData.value = data
}

async function loadTransactions() {
  const { data } = await assetApi.getAllTransactions(bookId)
  transactions.value = data.data || []
}'''

content = content.replace(old_load_sum, new_load_sum, 1)

# Add openChangeStatus, saveStatus, handleExport, handleImport before onMounted
old_onmounted = '''onMounted(() => {
  loadCards()
  loadCategories()
  loadSummary()
})'''

new_onmounted = '''function openChangeStatus(row) {
  Object.assign(changeForm, {
    card_id: row.id, current_status: row.status,
    status: '', location: '', department: '', employee_name: '', note: ''
  })
  showChangeDialog.value = true
}

async function saveStatus() {
  if (!changeForm.status) { ElMessage.warning('请选择新状态'); return }
  if (changeForm.status === changeForm.current_status) { ElMessage.warning('新状态与当前状态相同'); return }
  const payload = { status: changeForm.status }
  if (changeForm.location) payload.location = changeForm.location
  if (changeForm.department) payload.department = changeForm.department
  if (changeForm.employee_name) payload.employee_name = changeForm.employee_name
  if (changeForm.note) payload.note = changeForm.note
  await assetApi.changeStatus(bookId, changeForm.card_id, payload)
  ElMessage.success('状态变更成功')
  showChangeDialog.value = false
  loadCards()
  loadTransactions()
}

async function handleExport() {
  const { data } = await assetApi.exportAssets(bookId)
  const items = data.data || []
  if (items.length === 0) { ElMessage.warning('暂无资产可导出'); return }
  const headers = ['编号','名称','规格','分类','原值','累计折旧','净值','月折旧','状态','部门','责任人','存放地点','取得日期','折旧起始月','使用年限','净残值率','来源','备注']
  const fields = ['code','name','spec_model','category_name','original_value','accumulated_depreciation','net_value','monthly_depreciation','status','department','employee_name','location','acquisition_date','depreciation_start_month','useful_life_months','residual_value_rate','source','remark']
  let csv = headers.join('\\t') + '\\n'
  for (const item of items) {
    csv += fields.map(f => item[f] ?? '').join('\\t') + '\\n'
  }
  const blob = new Blob(['\\uFEFF' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = '资产清单_' + new Date().toISOString().slice(0,10) + '.csv'
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('已导出 ' + items.length + ' 条')
}

async function handleImport() {
  if (!importJson.value.trim()) { ElMessage.warning('请输入JSON数据'); return }
  let items
  try {
    items = JSON.parse(importJson.value)
    if (!Array.isArray(items)) throw new Error('数据必须是数组')
  } catch (e) {
    ElMessage.error('JSON格式错误：' + e.message)
    return
  }
  if (items.length === 0) { ElMessage.warning('数据为空'); return }
  try {
    const { data } = await assetApi.importAssets(bookId, { items })
    if (data.errors && data.errors.length > 0) {
      ElMessage.warning('导入 ' + data.imported + '/' + data.total + '，失败 ' + data.errors.length + ' 条')
    } else {
      ElMessage.success('成功导入 ' + data.imported + ' 条')
    }
    showImportDialog.value = false
    importJson.value = ''
    loadCards()
    loadSummary()
    loadTransactions()
  } catch (e) {
    ElMessage.error('导入失败：' + e.message)
  }
}

onMounted(() => {
  loadCards()
  loadCategories()
  loadSummary()
  loadTransactions()
})'''

content = content.replace(old_onmounted, new_onmounted, 1)

# Write back
with open(path, 'w') as f:
    f.write(content)

print(f"OK: Assets.vue patched ({original_len} -> {len(content)} chars)")

