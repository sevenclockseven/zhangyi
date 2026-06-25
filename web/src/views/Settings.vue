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

        <div style="margin-bottom: 12px">
          <el-button type="primary" size="small" @click="showAuxEdit = true">
            <el-icon><Plus /></el-icon>新增{{ auxLabel }}
          </el-button>
        </div>

        <div class="table-wrapper">
          <el-table :data="auxItems" border size="small">
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="is_active" label="状态" width="70" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '启用' : '停' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
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
    </el-tabs>

    <!-- Aux edit dialog -->
    <el-dialog v-model="showAuxEdit" :title="editingAux ? '编辑' : '新增' + auxLabel" :width="isMobile ? '90%' : '400px'">
      <el-form :model="auxForm" label-width="60px">
        <el-form-item label="编码" required>
          <el-input v-model="auxForm.code" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="auxForm.name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAuxEdit = false">取消</el-button>
        <el-button type="primary" @click="saveAux">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const books = ref([])
const currentBook = ref(null)
const activeTab = ref('aux')
const auxType = ref('customer')
const auxItems = ref([])
const bookInfo = ref(null)

const showAuxEdit = ref(false)
const editingAux = ref(null)
const auxForm = ref({ code: '', name: '' })

const auxLabel = computed(() => ({
  customer: '客户', supplier: '供应商', department: '部门',
  project: '项目', employee: '员工', warehouse: '仓库', bank_account: '银行账号'
}[auxType.value] || ''))

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

const onTabChange = () => {
  if (activeTab.value === 'aux') loadAux()
  else if (activeTab.value === 'book') loadBookInfo()
}

const editAux = (row) => {
  editingAux.value = row
  auxForm.value = { code: row.code, name: row.name }
  showAuxEdit.value = true
}

const saveAux = async () => {
  try {
    if (editingAux.value) {
      await axios.put(`/api/books/${currentBook.value}/aux/${auxType.value}/${editingAux.value.id}`, auxForm.value)
    } else {
      await axios.post(`/api/books/${currentBook.value}/aux/${auxType.value}`, auxForm.value)
    }
    ElMessage.success('保存成功')
    showAuxEdit.value = false
    editingAux.value = null
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

watch(showAuxEdit, (val) => {
  if (!val) editingAux.value = null
  else if (!editingAux.value) auxForm.value = { code: '', name: '' }
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
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.inner-tabs :deep(.el-tabs__header) { margin-bottom: 8px; }
</style>
