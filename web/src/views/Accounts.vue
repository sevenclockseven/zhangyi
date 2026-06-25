<template>
  <div class="accounts">
    <div class="page-header">
      <h2>科目管理</h2>
      <div>
        <el-select v-model="currentBook" placeholder="选择账套" style="width: 200px; margin-right: 12px" @change="loadAccounts">
          <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
        </el-select>
        <el-button @click="syncTemplate" :disabled="!currentBook" type="success" plain>
          <el-icon><Refresh /></el-icon>同步模板
        </el-button>
        <el-button type="primary" @click="showAdd = true" :disabled="!currentBook">
          <el-icon><Plus /></el-icon>新增科目
        </el-button>
      </div>
    </div>

    <el-row :gutter="16" v-if="currentBook">
      <!-- Account tree -->
      <el-col :span="10">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>科目树</span>
              <el-input v-model="searchText" placeholder="搜索科目" clearable style="width: 160px" size="small" />
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
                <el-tag v-if="!data.is_active" type="info" size="small" style="margin-left: 8px">停用</el-tag>
                <el-tag v-if="data.is_system" type="warning" size="small" style="margin-left: 4px">系统</el-tag>
              </span>
            </template>
          </el-tree>
        </el-card>
      </el-col>

      <!-- Account detail -->
      <el-col :span="14">
        <el-card shadow="never" v-if="selectedAccount">
          <template #header>科目详情</template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="编码">{{ selectedAccount.code }}</el-descriptions-item>
            <el-descriptions-item label="名称">{{ selectedAccount.name }}</el-descriptions-item>
            <el-descriptions-item label="方向">{{ selectedAccount.direction }}</el-descriptions-item>
            <el-descriptions-item label="层级">{{ selectedAccount.level }}级</el-descriptions-item>
            <el-descriptions-item label="末级科目">
              <el-tag :type="selectedAccount.is_leaf ? 'success' : 'info'" size="small">
                {{ selectedAccount.is_leaf ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="系统科目">
              <el-tag :type="selectedAccount.is_system ? 'warning' : 'info'" size="small">
                {{ selectedAccount.is_system ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="辅助核算" :span="2">
              {{ selectedAccount.aux_types || '无' }}
            </el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 12px">
            <el-button size="small" @click="editAccount(selectedAccount)">编辑</el-button>
            <el-button size="small" :type="selectedAccount.is_active ? 'warning' : 'success'"
              @click="toggleActive(selectedAccount)">
              {{ selectedAccount.is_active ? '停用' : '启用' }}
            </el-button>
            <el-button size="small" type="danger" @click="deleteAccount(selectedAccount)" :disabled="selectedAccount.is_system">
              删除
            </el-button>
          </div>
        </el-card>
        <el-card shadow="never" v-else>
          <el-empty description="点击左侧科目查看详情" :image-size="80" />
        </el-card>
      </el-col>
    </el-row>

    <!-- Add/Edit dialog -->
    <el-dialog v-model="showAdd" :title="editingAccount ? '编辑科目' : '新增科目'" width="500px">
      <el-form :model="form" label-width="80px">
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
            <el-checkbox value="bank_account">银行账号</el-checkbox>
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
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const books = ref([])
const currentBook = ref(null)
const accounts = ref([])
const accountTree = ref([])
const selectedAccount = ref(null)
const searchText = ref('')
const treeRef = ref(null)

const showAdd = ref(false)
const editingAccount = ref(null)
const form = ref({ code: '', name: '', direction: '借', aux_types: [] })

watch(searchText, (val) => { treeRef.value?.filter(val) })

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0 && !currentBook.value) currentBook.value = books.value[0].id
}

const loadAccounts = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/accounts`)
  accounts.value = data.data || []
  accountTree.value = buildTree(accounts.value)
}

const buildTree = (list) => {
  const map = {}
  const roots = []
  list.forEach(a => { map[a.code] = { ...a, children: [] } })
  list.forEach(a => {
    if (a.parent_code && map[a.parent_code]) {
      map[a.parent_code].children.push(map[a.code])
    } else if (!a.parent_code) {
      roots.push(map[a.code])
    }
  })
  // Remove empty children arrays
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
  form.value = {
    code: acct.code,
    name: acct.name,
    direction: acct.direction,
    aux_types: acct.aux_types ? acct.aux_types.split(',') : []
  }
  showAdd.value = true
}

const saveAccount = async () => {
  try {
    const payload = { ...form.value, aux_types: form.value.aux_types.join(',') }
    if (editingAccount.value) {
      await axios.put(`/api/books/${currentBook.value}/accounts/${editingAccount.value.id}`, payload)
      ElMessage.success('修改成功')
    } else {
      await axios.post(`/api/books/${currentBook.value}/accounts`, payload)
      ElMessage.success('新增成功')
    }
    showAdd.value = false
    editingAccount.value = null
    loadAccounts()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const toggleActive = async (acct) => {
  try {
    await axios.put(`/api/books/${currentBook.value}/accounts/${acct.id}`, { is_active: !acct.is_active })
    ElMessage.success(acct.is_active ? '已停用' : '已启用')
    loadAccounts()
  } catch (e) { ElMessage.error('操作失败') }
}

const deleteAccount = async (acct) => {
  await ElMessageBox.confirm(`确定删除科目 ${acct.code} ${acct.name}？`, '确认')
  try {
    await axios.delete(`/api/books/${currentBook.value}/accounts/${acct.id}`)
    ElMessage.success('删除成功')
    selectedAccount.value = null
    loadAccounts()
  } catch (e) { ElMessage.error(e.response?.data?.error || '删除失败') }
}

const syncTemplate = async () => {
  try {
    await axios.post(`/api/books/${currentBook.value}/sync-template`)
    ElMessage.success('模板同步成功')
    loadAccounts()
  } catch (e) { ElMessage.error('同步失败') }
}

watch(showAdd, (val) => {
  if (!val) editingAccount.value = null
  else if (!editingAccount.value) form.value = { code: '', name: '', direction: '借', aux_types: [] }
})

watch(currentBook, () => {
  loadAccounts()
  selectedAccount.value = null
})

onMounted(loadBooks)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { color: #303133; }
.tree-node { display: flex; align-items: center; font-size: 13px; }
</style>
