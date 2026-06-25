<template>
  <div class="settings">
    <div class="page-header">
      <h2>系统设置</h2>
      <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="loadAux">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="onTabChange">
      <el-tab-pane label="辅助核算" name="aux">
        <el-tabs v-model="auxType" @tab-change="loadAux" type="card" class="inner-tabs">
          <el-tab-pane label="客户" name="customer" />
          <el-tab-pane label="供应商" name="supplier" />
          <el-tab-pane label="部门" name="department" />
          <el-tab-pane label="项目" name="project" />
          <el-tab-pane label="员工" name="employee" />
          <el-tab-pane label="仓库" name="warehouse" />
          <el-tab-pane label="银行" name="bank_account" />
        </el-tabs>

        <div class="toolbar">
          <el-button type="primary" size="small" @click="openAdd">
            <el-icon><Plus /></el-icon>新增{{ auxLabel }}
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
          <el-button size="small" type="danger" :disabled="selectedItems.length === 0" @click="batchDelete" style="margin-left: 8px">
            删除选中({{ selectedItems.length }})
          </el-button>
        </div>

        <div class="table-wrapper">
          <el-table :data="auxItems" border size="small" @selection-change="onSelectionChange" :max-height="tableMaxHeight">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" :label="auxType === 'employee' ? '姓名' : '名称'" min-width="120" />
            <!-- Dynamic columns based on type -->
            <template v-if="auxType === 'customer' || auxType === 'supplier'">
              <el-table-column label="联系人" width="100">
                <template #default="{ row }">{{ getExtra(row, 'contact') }}</template>
              </el-table-column>
              <el-table-column label="电话" width="120">
                <template #default="{ row }">{{ getExtra(row, 'phone') }}</template>
              </el-table-column>
              <el-table-column label="地址" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'department'">
              <el-table-column label="上级部门" width="120">
                <template #default="{ row }">{{ getExtra(row, 'parent') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'project'">
              <el-table-column label="状态" width="80">
                <template #default="{ row }">{{ getExtra(row, 'status') }}</template>
              </el-table-column>
              <el-table-column label="开始日期" width="100">
                <template #default="{ row }">{{ getExtra(row, 'start_date') }}</template>
              </el-table-column>
              <el-table-column label="结束日期" width="100">
                <template #default="{ row }">{{ getExtra(row, 'end_date') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'employee'">
              <el-table-column label="部门" width="100">
                <template #default="{ row }">{{ getExtra(row, 'department') }}</template>
              </el-table-column>
              <el-table-column label="电话" width="120">
                <template #default="{ row }">{{ getExtra(row, 'phone') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'warehouse'">
              <el-table-column label="地址" min-width="180">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'bank_account'">
              <el-table-column label="银行账号" width="150">
                <template #default="{ row }">{{ getExtra(row, 'account_number') }}</template>
              </el-table-column>
              <el-table-column label="开户行" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'bank_name') }}</template>
              </el-table-column>
              <el-table-column label="户名" width="120">
                <template #default="{ row }">{{ getExtra(row, 'account_holder') }}</template>
              </el-table-column>
              <el-table-column label="地址" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <el-table-column label="备注" min-width="100">
              <template #default="{ row }">{{ getExtra(row, 'memo') }}</template>
            </el-table-column>
            <el-table-column prop="is_active" label="状态" width="70" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '启用' : '停' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="editAux(row)">编辑</el-button>
                <el-button size="small" type="danger" link @click="deleteAux(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="账套信息" name="book">
        <el-card shadow="never" v-if="bookInfo">
          <el-descriptions :column="isMobile ? 1 : 2" border size="small">
            <el-descriptions-item label="名称">{{ bookInfo.name }}</el-descriptions-item>
            <el-descriptions-item label="编码">{{ bookInfo.code }}</el-descriptions-item>
            <el-descriptions-item label="行业">{{ bookInfo.industry }}</el-descriptions-item>
            <el-descriptions-item label="纳税人">{{ bookInfo.taxpayer_type === 'general' ? '一般纳税人' : '小规模' }}</el-descriptions-item>
            <el-descriptions-item label="启用期间">{{ bookInfo.start_date }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ bookInfo.status }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="菜单排序" name="menu">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>菜单排序（拖拽调整顺序，可控制显示/隐藏）</span>
              <el-button size="small" @click="resetMenu">恢复默认</el-button>
            </div>
          </template>
          <div class="menu-sort-list">
            <div v-for="(item, index) in menuConfig" :key="item.index" class="menu-sort-item">
              <div class="menu-sort-left">
                <el-icon class="sort-handle"><Rank /></el-icon>
                <el-icon><component :is="iconMap[item.icon] || HomeFilled" /></el-icon>
                <span>{{ item.label }}</span>
              </div>
              <div class="menu-sort-right">
                <el-button size="small" :disabled="index === 0" @click="moveMenu(index, -1)">
                  <el-icon><Top /></el-icon>
                </el-button>
                <el-button size="small" :disabled="index === menuConfig.length - 1" @click="moveMenu(index, 1)">
                  <el-icon><Bottom /></el-icon>
                </el-button>
                <el-switch v-model="item.visible" @change="saveMenu" style="margin-left: 12px" />
              </div>
            </div>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- Add/Edit dialog -->
    <el-dialog v-model="showEdit" :title="editingItem ? '编辑' : '新增' + auxLabel" :width="isMobile ? '95%' : '550px'">
      <el-form :model="editForm" label-width="80px" size="small">
        <el-form-item label="编码" required>
          <el-input v-model="editForm.code" placeholder="唯一编码" />
        </el-form-item>
        <el-form-item :label="auxType === 'employee' ? '姓名' : '名称'" required>
          <el-input v-model="editForm.name" :placeholder="auxType === 'employee' ? '姓名' : '名称'" />
        </el-form-item>

        <!-- Customer/Supplier extra fields -->
        <template v-if="auxType === 'customer' || auxType === 'supplier'">
          <el-form-item label="联系人">
            <el-input v-model="editForm.extra.contact" />
          </el-form-item>
          <el-form-item label="电话">
            <el-input v-model="editForm.extra.phone" />
          </el-form-item>
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <!-- Department -->
        <template v-if="auxType === 'department'">
          <el-form-item label="上级部门">
            <el-input v-model="editForm.extra.parent" />
          </el-form-item>
        </template>

        <!-- Project -->
        <template v-if="auxType === 'project'">
          <el-form-item label="状态">
            <el-select v-model="editForm.extra.status" style="width: 100%">
              <el-option label="进行中" value="进行中" />
              <el-option label="已完成" value="已完成" />
              <el-option label="已暂停" value="已暂停" />
            </el-select>
          </el-form-item>
          <el-form-item label="开始日期">
            <el-date-picker v-model="editForm.extra.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束日期">
            <el-date-picker v-model="editForm.extra.end_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          </el-form-item>
        </template>

        <!-- Employee -->
        <template v-if="auxType === 'employee'">
          <el-form-item label="部门">
            <el-input v-model="editForm.extra.department" />
          </el-form-item>
          <el-form-item label="电话">
            <el-input v-model="editForm.extra.phone" />
          </el-form-item>
        </template>

        <!-- Warehouse -->
        <template v-if="auxType === 'warehouse'">
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <!-- Bank Account -->
        <template v-if="auxType === 'bank_account'">
          <el-form-item label="银行账号">
            <el-input v-model="editForm.extra.account_number" />
          </el-form-item>
          <el-form-item label="开户行">
            <el-input v-model="editForm.extra.bank_name" />
          </el-form-item>
          <el-form-item label="户名">
            <el-input v-model="editForm.extra.account_holder" />
          </el-form-item>
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <el-form-item label="备注">
          <el-input v-model="editForm.extra.memo" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">取消</el-button>
        <el-button type="primary" @click="saveItem">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting, SwitchButton, Top, Bottom, Rank } from '@element-plus/icons-vue'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 320px)' : 'calc(100vh - 350px)')

const books = ref([])
const currentBook = ref(null)
const activeTab = ref('aux')
const auxType = ref('customer')
const auxItems = ref([])
const bookInfo = ref(null)
const selectedItems = ref([])

const showEdit = ref(false)
const editingItem = ref(null)
const editForm = ref({ code: '', name: '', extra: {} })

const auxLabel = computed(() => ({
  customer: '客户', supplier: '供应商', department: '部门',
  project: '项目', employee: '员工', warehouse: '仓库', bank_account: '银行账号'
}[auxType.value] || ''))

const importUrl = computed(() => `/api/books/${currentBook.value}/aux/${auxType.value}/import`)
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
}))

const defaultExtra = () => {
  switch (auxType.value) {
    case 'customer': case 'supplier':
      return { contact: '', phone: '', address: '', memo: '' }
    case 'department':
      return { parent: '' }
    case 'project':
      return { status: '进行中', start_date: '', end_date: '', memo: '' }
    case 'employee':
      return { department: '', phone: '', memo: '' }
    case 'warehouse':
      return { address: '', memo: '' }
    case 'bank_account':
      return { account_number: '', bank_name: '', account_holder: '', address: '', memo: '' }
    default:
      return {}
  }
}

const getExtra = (row, key) => {
  try {
    const extra = JSON.parse(row.extra || '{}')
    return extra[key] || ''
  } catch { return '' }
}

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
}

const loadAux = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/aux/${auxType.value}`)
  auxItems.value = data.data || []
}

const loadBookInfo = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}`)
  bookInfo.value = data.data
}

// Menu config
const iconMap = { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting }
const menuConfig = ref([])

const loadMenuConfig = () => {
  try {
    const saved = localStorage.getItem('zhangyi_menu_config')
    if (saved) {
      menuConfig.value = JSON.parse(saved)
    } else {
      menuConfig.value = [
        { index: '/', label: '工作台', icon: 'HomeFilled', visible: true },
        { index: '/books', label: '账套管理', icon: 'Notebook', visible: true },
        { index: '/accounts', label: '科目管理', icon: 'Memo', visible: true },
        { index: '/vouchers', label: '凭证管理', icon: 'Document', visible: true },
        { index: '/ledger', label: '账簿查询', icon: 'List', visible: true },
        { index: '/reports', label: '报表中心', icon: 'DataAnalysis', visible: true },
        { index: '/closing', label: '期末处理', icon: 'SwitchButton', visible: true },
        { index: '/settings', label: '系统设置', icon: 'Setting', visible: true },
      ]
    }
  } catch {}
}

const saveMenu = () => {
  localStorage.setItem('zhangyi_menu_config', JSON.stringify(menuConfig.value))
  // Trigger App.vue to reload
  window.dispatchEvent(new Event('menu-config-changed'))
}

const moveMenu = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= menuConfig.value.length) return
  const arr = [...menuConfig.value]
  const temp = arr[index]
  arr[index] = arr[newIndex]
  arr[newIndex] = temp
  menuConfig.value = arr
  saveMenu()
}

const resetMenu = () => {
  localStorage.removeItem('zhangyi_menu_config')
  loadMenuConfig()
  window.dispatchEvent(new Event('menu-config-changed'))
  ElMessage.success('已恢复默认菜单')
}

const onTabChange = () => {
  if (activeTab.value === 'aux') loadAux()
  else if (activeTab.value === 'book') loadBookInfo()
  else if (activeTab.value === 'menu') loadMenuConfig()
}

const onSelectionChange = (rows) => { selectedItems.value = rows }

const openAdd = () => {
  editingItem.value = null
  editForm.value = { code: '', name: '', extra: defaultExtra() }
  showEdit.value = true
}

const editAux = (row) => {
  editingItem.value = row
  let extra = {}
  try { extra = JSON.parse(row.extra || '{}') } catch {}
  editForm.value = { code: row.code, name: row.name, extra: { ...defaultExtra(), ...extra } }
  showEdit.value = true
}

const saveItem = async () => {
  try {
    const payload = {
      code: editForm.value.code,
      name: editForm.value.name,
      extra: JSON.stringify(editForm.value.extra)
    }
    if (editingItem.value) {
      await axios.put(`/api/books/${currentBook.value}/aux/${auxType.value}/${editingItem.value.id}`, payload)
    } else {
      await axios.post(`/api/books/${currentBook.value}/aux/${auxType.value}`, payload)
    }
    ElMessage.success('保存成功')
    showEdit.value = false
    editingItem.value = null
    loadAux()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const deleteAux = async (row) => {
  await ElMessageBox.confirm(`确定删除 ${row.name}？`, '确认')
  try {
    await axios.delete(`/api/books/${currentBook.value}/aux/${auxType.value}/${row.id}`)
    ElMessage.success('已删除')
    loadAux()
  } catch (e) { ElMessage.error('删除失败') }
}

const batchDelete = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedItems.value.length} 条？`, '确认')
  try {
    const ids = selectedItems.value.map(r => r.id)
    await axios.post(`/api/books/${currentBook.value}/aux/${auxType.value}/batch-delete`, { ids })
    ElMessage.success('批量删除成功')
    loadAux()
  } catch (e) { ElMessage.error('删除失败') }
}

const exportData = () => {
  const token = localStorage.getItem('token')
  window.open(`/api/books/${currentBook.value}/aux/${auxType.value}/export?token=${token}`, '_blank')
}

const onImportSuccess = (resp) => {
  ElMessage.success(resp.message || '导入成功')
  loadAux()
}

const onImportError = () => {
  ElMessage.error('导入失败')
}

watch(showEdit, (val) => {
  if (!val) editingItem.value = null
})

watch(currentBook, () => { loadAux(); loadBookInfo() })

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.toolbar { display: flex; align-items: center; flex-wrap: wrap; gap: 0; margin-bottom: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.inner-tabs :deep(.el-tabs__header) { margin-bottom: 8px; }

.menu-sort-list { display: flex; flex-direction: column; gap: 8px; }
.menu-sort-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 14px; background: #f5f7fa; border-radius: 6px;
  border: 1px solid #ebeef5;
}
.menu-sort-left { display: flex; align-items: center; gap: 10px; font-size: 14px; }
.sort-handle { cursor: grab; color: #909399; }
.menu-sort-right { display: flex; align-items: center; gap: 4px; }
</style>
