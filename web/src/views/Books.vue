<template>
  <div class="books">
    <div class="page-header">
      <h2>账套管理</h2>
      <el-button type="primary" @click="showCreate = true">
        <el-icon><Plus /></el-icon>新建账套
      </el-button>
    </div>

    <!-- Mobile card list -->
    <div class="book-cards" v-if="isMobile">
      <el-card v-for="book in books" :key="book.id" class="book-card" shadow="hover">
        <div class="book-card-header">
          <span class="book-name">{{ book.name }}</span>
          <el-tag :type="book.status === 'active' ? 'success' : 'info'" size="small">
            {{ book.status === 'active' ? '启用' : '停用' }}
          </el-tag>
        </div>
        <div class="book-card-info">
          <div>编码：{{ book.code }}</div>
          <div>行业：{{ book.industry }}</div>
          <div>启用：{{ book.start_date }}</div>
        </div>
        <div class="book-card-actions">
          <el-button size="small" type="primary" link @click="$router.push('/accounts?book=' + book.id)">科目</el-button>
          <el-button size="small" type="primary" link @click="$router.push('/vouchers?book=' + book.id)">凭证</el-button>
          <el-button size="small" type="danger" link @click="deleteBook(book)">删除</el-button>
        </div>
      </el-card>
      <el-empty v-if="books.length === 0" description="暂无账套" />
    </div>

    <!-- Desktop table -->
    <el-table v-else :data="books" stripe style="width: 100%">
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="客户名称" />
      <el-table-column prop="industry" label="行业" width="150" />
      <el-table-column prop="taxpayer_type" label="纳税人类型" width="120">
        <template #default="{ row }">
          {{ row.taxpayer_type === 'general' ? '一般纳税人' : '小规模' }}
        </template>
      </el-table-column>
      <el-table-column prop="start_date" label="启用期间" width="100" />
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="$router.push('/accounts?book=' + row.id)">科目</el-button>
          <el-button size="small" type="primary" link @click="$router.push('/vouchers?book=' + row.id)">凭证</el-button>
          <el-button size="small" type="danger" link @click="deleteBook(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create dialog -->
    <el-dialog v-model="showCreate" title="新建账套" :width="isMobile ? '95%' : '600px'">
      <el-form :model="form" :label-width="isMobile ? '80px' : '100px'">
        <el-form-item label="客户名称" required>
          <el-input v-model="form.name" placeholder="请输入客户名称" />
        </el-form-item>
        <el-form-item label="行业类型" required>
          <el-select v-model="form.industry" multiple placeholder="选择行业" style="width: 100%">
            <el-option label="制造业" value="manufacturing" />
            <el-option label="零售业" value="retail" />
            <el-option label="服务业" value="service" />
            <el-option label="建筑业" value="construction" />
            <el-option label="房地产业" value="real_estate" />
            <el-option label="运输业" value="transport" />
            <el-option label="农业" value="agriculture" />
          </el-select>
        </el-form-item>
        <el-form-item label="纳税人类型">
          <el-radio-group v-model="form.taxpayer_type">
            <el-radio value="small">小规模</el-radio>
            <el-radio value="general">一般纳税人</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="会计准则">
          <el-radio-group v-model="form.accounting_standard">
            <el-radio value="small_business">小企业会计准则</el-radio>
            <el-radio value="enterprise">企业会计准则</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="启用期间" required>
          <el-date-picker v-model="form.start_date" type="month" value-format="YYYY-MM" placeholder="选择月份" style="width: 100%" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="form.contact" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="form.phone" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="createBook">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const books = ref([])
const showCreate = ref(false)
const form = ref({
  name: '',
  industry: [],
  taxpayer_type: 'small',
  accounting_standard: 'small_business',
  start_date: '',
  contact: '',
  phone: ''
})

const loadBooks = async () => {
  try {
    const { data } = await axios.get('/api/books')
    books.value = data.data || []
  } catch (e) {
    console.error('Failed to load books:', e)
  }
}

const createBook = async () => {
  if (!form.value.name || !form.value.industry.length || !form.value.start_date) {
    ElMessage.warning('请填写必填项')
    return
  }
  try {
    await axios.post('/api/books', form.value)
    ElMessage.success('创建成功')
    showCreate.value = false
    form.value = { name: '', industry: [], taxpayer_type: 'small', accounting_standard: 'small_business', start_date: '', contact: '', phone: '' }
    loadBooks()
  } catch (e) {
    ElMessage.error('创建失败: ' + (e.response?.data?.error || e.message))
  }
}

const deleteBook = async (book) => {
  try {
    await ElMessageBox.confirm(`确定删除账套"${book.name}"？`, '确认', { type: 'warning' })
    await axios.delete(`/api/books/${book.id}`)
    ElMessage.success('已删除')
    loadBooks()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
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
  font-size: 18px;
}

.book-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.book-card :deep(.el-card__body) {
  padding: 14px;
}

.book-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.book-name {
  font-weight: 600;
  font-size: 15px;
  color: #303133;
}

.book-card-info {
  font-size: 13px;
  color: #909399;
  line-height: 1.8;
  margin-bottom: 8px;
}

.book-card-actions {
  display: flex;
  gap: 8px;
  border-top: 1px solid #ebeef5;
  padding-top: 8px;
}
</style>
