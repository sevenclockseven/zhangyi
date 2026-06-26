<template>
  <div class="voucher-templates">
    <div class="page-header">
      <h2>凭证模板</h2>
      <el-button type="primary" size="small" @click="openAdd">
        <el-icon><Plus /></el-icon>新增模板
      </el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="templates" border size="small" :max-height="tableMaxHeight">
        <el-table-column prop="name" label="模板名称" min-width="150" />
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column label="分录" min-width="300">
          <template #default="{ row }">
            <span v-for="(item, i) in parseItems(row.items)" :key="i" style="margin-right: 12px; font-size: 13px; color: #606266">
              {{ item.account_code }} {{ item.account_name }}<span v-if="item.memo">（{{ item.memo }}）</span>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="editItem(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click="deleteItem(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Edit dialog -->
    <el-dialog v-model="showEdit" :title="editing ? '编辑模板' : '新增模板'" :width="isMobile ? '95%' : '600px'">
      <el-form :model="form" label-width="80px">
        <el-form-item label="模板名称" required>
          <el-input v-model="form.name" placeholder="如：收货款、付工资" />
        </el-form-item>
        <el-form-item label="分类">
          <el-input v-model="form.category" placeholder="如：收入、费用（可选）" />
        </el-form-item>
        <el-form-item label="分录">
          <div v-for="(item, i) in form.items" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <el-select v-model="item.account_id" filterable placeholder="科目" style="flex: 1">
              <el-option v-for="a in accounts" :key="a.id" :label="a.code + ' ' + a.name" :value="a.id" :disabled="!a.is_leaf" />
            </el-select>
            <el-input v-model="item.memo" placeholder="摘要" style="width: 150px" />
            <el-button size="small" type="danger" link @click="form.items.splice(i, 1)" :disabled="form.items.length <= 1">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button size="small" @click="form.items.push({ account_id: null, memo: '' })">
            <el-icon><Plus /></el-icon>添加分录
          </el-button>
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
import { ref, onMounted, watch } from 'vue'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const { isMobile } = useMobile()
const tableMaxHeight = isMobile.value ? 'calc(100vh - 200px)' : 'calc(100vh - 250px)'

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const templates = ref([])
const accounts = ref([])
const showEdit = ref(false)
const editing = ref(null)
const form = ref({ name: '', category: '', items: [{ account_id: null, memo: '' }] })

const parseItems = (itemsStr) => {
  try { return JSON.parse(itemsStr || '[]') } catch { return [] }
}

const loadData = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/voucher-templates`)
  templates.value = data.data || []
  const { data: accData } = await axios.get(`/api/books/${currentBook.value}/accounts`)
  accounts.value = accData.data || []
}

const openAdd = () => {
  editing.value = null
  form.value = { name: '', category: '', items: [{ account_id: null, memo: '' }] }
  showEdit.value = true
}

const editItem = (row) => {
  editing.value = row
  const items = parseItems(row.items)
  form.value = { name: row.name, category: row.category || '', items: items.length > 0 ? items : [{ account_id: null, memo: '' }] }
  showEdit.value = true
}

const saveItem = async () => {
  if (!form.value.name) { ElMessage.warning('请输入模板名称'); return }
  // Build items with account info
  const items = form.value.items.filter(i => i.account_id).map(i => {
    const acct = accounts.value.find(a => a.id === i.account_id)
    return { account_id: i.account_id, account_code: acct?.code || '', account_name: acct?.name || '', memo: i.memo || '' }
  })
  if (items.length === 0) { ElMessage.warning('请至少添加一条分录'); return }

  try {
    const payload = { name: form.value.name, category: form.value.category, items: JSON.stringify(items) }
    if (editing.value) {
      await axios.put(`/api/books/${currentBook.value}/voucher-templates/${editing.value.id}`, payload)
    } else {
      await axios.post(`/api/books/${currentBook.value}/voucher-templates`, payload)
    }
    ElMessage.success('保存成功')
    showEdit.value = false
    loadData()
  } catch (e) { ElMessage.error('保存失败') }
}

const deleteItem = async (row) => {
  await ElMessageBox.confirm(`确定删除模板"${row.name}"？`, '确认')
  try {
    await axios.delete(`/api/books/${currentBook.value}/voucher-templates/${row.id}`)
    ElMessage.success('已删除')
    loadData()
  } catch (e) { ElMessage.error('删除失败') }
}

watch(currentBook, (val) => { if (val) loadData() })

onMounted(() => {
  if (currentBook.value) loadData()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { color: #303133; font-size: 18px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
