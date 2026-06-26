<template>
  <div class="accounts">
    <div class="page-header">
      <h2>科目管理</h2>
      <div class="header-actions">
        <el-button @click="syncTemplate" :disabled="!currentBook" type="success" plain size="small">
          <el-icon><Refresh /></el-icon>同步模板
        </el-button>
        <el-button type="primary" @click="showAdd = true" :disabled="!currentBook" size="small">
          <el-icon><Plus /></el-icon>新增
        </el-button>
      </div>
    </div>

    <div :class="isMobile ? 'accounts-layout-mobile' : 'accounts-layout'" v-if="currentBook">
      <!-- Account tree -->
      <el-card shadow="never" :class="isMobile && selectedAccount ? 'hidden-mobile' : ''">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center; gap: 8px">
            <span>科目树</span>
            <el-input v-model="searchText" placeholder="搜索" clearable style="width: 140px" size="small" />
          </div>
        </template>
        <el-tree
          :data="accountTree"
          :props="{ label: 'name', children: 'children' }"
          node-key="id"
          default-expand-all
          highlight-current
          :filter-node-method="filterNode"
          ref="treeRef"
          @node-click="selectAccount"
        >
          <template #default="{ data }">
            <span class="tree-node">
              <span>{{ data.code }} {{ data.name }}</span>
              <el-tag v-if="!data.is_active" type="info" size="small" style="margin-left: 4px">停</el-tag>
            </span>
          </template>
        </el-tree>
      </el-card>

      <!-- Account detail -->
      <el-card shadow="never" v-if="selectedAccount" :class="isMobile ? 'detail-mobile' : ''">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center">
            <span>科目详情</span>
            <el-button v-if="isMobile" size="small" link @click="selectedAccount = null">返回列表</el-button>
          </div>
        </template>
        <el-descriptions :column="isMobile ? 1 : 2" border size="small">
          <el-descriptions-item label="编码">{{ selectedAccount.code }}</el-descriptions-item>
          <el-descriptions-item label="名称">{{ selectedAccount.name }}</el-descriptions-item>
          <el-descriptions-item label="方向">{{ selectedAccount.direction }}</el-descriptions-item>
          <el-descriptions-item label="层级">{{ selectedAccount.level }}级</el-descriptions-item>
          <el-descriptions-item label="末级">
            <el-tag :type="selectedAccount.is_leaf ? 'success' : 'info'" size="small">{{ selectedAccount.is_leaf ? '是' : '否' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="系统科目">
            <el-tag :type="selectedAccount.is_system ? 'warning' : 'info'" size="small">{{ selectedAccount.is_system ? '是' : '否' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="辅助核算" :span="isMobile ? 1 : 2">{{ selectedAccount.aux_types || '无' }}</el-descriptions-item>
        </el-descriptions>
        <div style="margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap">
          <el-button size="small" @click="editAccount(selectedAccount)">编辑</el-button>
          <el-button size="small" :type="selectedAccount.is_active ? 'warning' : 'success'" @click="toggleActive(selectedAccount)">
            {{ selectedAccount.is_active ? '停用' : '启用' }}
          </el-button>
          <el-button size="small" type="danger" @click="deleteAccount(selectedAccount)" :disabled="selectedAccount.is_system">删除</el-button>
        </div>
      </el-card>
      <el-card shadow="never" v-else-if="!isMobile">
        <el-empty description="点击左侧科目查看详情" :image-size="60" />
      </el-card>
    </div>

    <!-- Add/Edit dialog -->
    <el-dialog v-model="showAdd" :title="editingAccount ? '编辑科目' : '新增科目'" :width="isMobile ? '95%' : '500px'">
      <el-form :model="form" :label-width="isMobile ? '70px' : '80px'">
        <el-form-item label="编码" required>
          <el-input v-model="form.code" placeholder="如: 1002.01" :disabled="!!editingAccount" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="科目名称" />
        </el-form-item>
        <el-form-item label="方向">
          <el-radio-group v-model="form.direction">
            <el-radio value="借">借</el-radio>
            <el-radio value="贷">贷</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="辅助核算">
          <el-checkbox-group v-model="form.aux_types">
            <el-checkbox value="customer">客户</el-checkbox>
            <el-checkbox value="supplier">供应商</el-checkbox>
            <el-checkbox value="department">部门</el-checkbox>
            <el-checkbox value="project">项目</el-checkbox>
            <el-checkbox value="employee">员工</el-checkbox>
            <el-checkbox value="warehouse">仓库</el-checkbox>
            <el-checkbox value="bank_account">银行</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdd = false">取消</el-button>
        <el-button type="primary" @click="saveAccount">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useBookStore } from '../stores/book'
import { useRoute } from 'vue-router'
import { useMobile } from '../composables/useMobile'
import { accountApi, bookApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { isMobile } = useMobile()
const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const route = useRoute()
const accounts = ref([])
const accountTree = ref([])
const selectedAccount = ref(null)
const searchText = ref('')
const treeRef = ref(null)

const showAdd = ref(false)
const editingAccount = ref(null)
const form = ref({ code: '', name: '', direction: '借', aux_types: [] })

watch(searchText, (val) => { treeRef.value?.filter(val) })

const loadAccounts = async () => {
  if (!currentBook.value) return
  const { data } = await accountApi.list(currentBook.value)
  accounts.value = data.data || []
  accountTree.value = buildTree(accounts.value)
}

const buildTree = (list) => {
  const map = {}
  const roots = []
  list.forEach(a => { map[a.code] = { ...a, children: [] } })
  list.forEach(a => {
    if (a.parent_code && map[a.parent_code]) map[a.parent_code].children.push(map[a.code])
    else if (!a.parent_code) roots.push(map[a.code])
  })
  const clean = (nodes) => nodes.forEach(n => {
    if (n.children.length === 0) delete n.children
    else clean(n.children)
  })
  clean(roots)
  return roots
}

const filterNode = (value, data) => {
  if (!value) return true
  return data.code.includes(value) || data.name.includes(value)
}

const selectAccount = (data) => { selectedAccount.value = data }

const editAccount = (acct) => {
  editingAccount.value = acct
  form.value = { code: acct.code, name: acct.name, direction: acct.direction, aux_types: acct.aux_types ? acct.aux_types.split(',') : [] }
  showAdd.value = true
}

const saveAccount = async () => {
  try {
    const payload = { ...form.value, aux_types: form.value.aux_types.join(',') }
    if (editingAccount.value) {
      await accountApi.update(currentBook.value, editingAccount.value.id, payload)
    } else {
      await accountApi.create(currentBook.value, payload)
    }
    ElMessage.success('保存成功')
    showAdd.value = false
    editingAccount.value = null
    loadAccounts()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const toggleActive = async (acct) => {
  try {
    await accountApi.update(currentBook.value, acct.id, { is_active: !acct.is_active })
    ElMessage.success(acct.is_active ? '已停用' : '已启用')
    loadAccounts()
  } catch (e) { ElMessage.error('操作失败') }
}

const deleteAccount = async (acct) => {
  await ElMessageBox.confirm(`确定删除 ${acct.code} ${acct.name}？`, '确认')
  try {
    await accountApi.delete(currentBook.value, acct.id)
    ElMessage.success('已删除')
    selectedAccount.value = null
    loadAccounts()
  } catch (e) { ElMessage.error(e.response?.data?.error || '删除失败') }
}

const syncTemplate = async () => {
  try {
    await bookApi.syncTemplate(currentBook.value)
    ElMessage.success('同步成功')
    loadAccounts()
  } catch (e) { ElMessage.error('同步失败') }
}

watch(showAdd, (val) => {
  if (!val) editingAccount.value = null
  else if (!editingAccount.value) form.value = { code: '', name: '', direction: '借', aux_types: [] }
})

watch(currentBook, (val) => { if (val) { loadAccounts(); selectedAccount.value = null } })

onMounted(() => {
  if (currentBook.value) loadAccounts()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }

.accounts-layout { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.accounts-layout-mobile { display: flex; flex-direction: column; gap: 12px; }

.hidden-mobile { display: none; }
.detail-mobile { position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 100; overflow-y: auto; background: #f5f7fa; }

.tree-node { display: flex; align-items: center; font-size: 13px; }

@media (max-width: 767px) {
  .header-actions { width: 100%; }
  .header-actions .el-select { flex: 1; }
}
</style>
