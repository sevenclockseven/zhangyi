<template>
  <div class="books">
    <div class="page-header">
      <h2>账套管理</h2>
      <el-button type="primary" @click="showCreate = true">
        <el-icon><Plus /></el-icon>新建账套
      </el-button>
    </div>

    <el-table :data="books" stripe style="width: 100%">
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
          <el-button size="small" type="primary" link>科目</el-button>
          <el-button size="small" type="primary" link>凭证</el-button>
          <el-button size="small" type="danger" link>删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create dialog -->
    <el-dialog v-model="showCreate" title="新建账套" width="600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="客户名称" required>
          <el-input v-model="form.name" placeholder="请输入客户名称" />
        </el-form-item>
        <el-form-item label="行业类型" required>
          <el-select v-model="form.industry" multiple placeholder="选择行业">
            <el-option label="制造业" value="manufacturing" />
            <el-option label="零售业" value="retail" />
            <el-option label="服务业" value="service" />
            <el-option label="建筑业" value="construction" />
            <el-option label="房地产业" value="realestate" />
          </el-select>
        </el-form-item>
        <el-form-item label="纳税人类型">
          <el-radio-group v-model="form.taxpayer_type">
            <el-radio value="small_scale">小规模纳税人</el-radio>
            <el-radio value="general">一般纳税人</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="启用期间" required>
          <el-date-picker v-model="form.start_date" type="month" value-format="YYYY-MM" placeholder="选择启用月份" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="form.contact" />
        </el-form-item>
        <el-form-item label="联系电话">
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
import { ElMessage } from 'element-plus'

const books = ref([])
const showCreate = ref(false)
const form = ref({
  name: '',
  industry: [],
  taxpayer_type: 'small_scale',
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
  try {
    await axios.post('/api/books', {
      ...form.value,
      code: 'BK' + String(Date.now()).slice(-6),
      industry: form.value.industry.join(',')
    })
    ElMessage.success('创建成功')
    showCreate.value = false
    loadBooks()
  } catch (e) {
    ElMessage.error('创建失败: ' + (e.response?.data?.error || e.message))
  }
}

onMounted(loadBooks)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  color: #303133;
}
</style>
