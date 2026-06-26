<template>
  <div class="custom-reports">
    <div class="page-header">
      <h2>自定义报表</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openAdd">
          <el-icon><Plus /></el-icon>新建报表
        </el-button>
      </div>
    </div>

    <div v-if="currentBook">
      <!-- Template list -->
      <el-card shadow="never" style="margin-bottom: 16px">
        <template #header><strong>报表模板</strong></template>
        <div v-if="templates.length === 0" style="color: #909399; padding: 20px; text-align: center">
          暂无自定义报表模板，点击"新建报表"创建
        </div>
        <div v-else class="template-grid">
          <el-card v-for="tpl in templates" :key="tpl.id" shadow="hover" class="template-card">
            <div class="template-header">
              <span class="template-name">{{ tpl.name }}</span>
              <el-button size="small" type="danger" link @click="deleteTemplate(tpl)">删除</el-button>
            </div>
            <div class="template-meta">类型：{{ tpl.type || '自定义' }}</div>
            <div class="template-actions">
              <el-button size="small" type="primary" @click="runReport(tpl)">运行</el-button>
              <el-date-picker v-model="runPeriod" type="month" value-format="YYYY-MM" placeholder="期间" size="small" style="width: 130px; margin-left: 8px" />
            </div>
          </el-card>
        </div>
      </el-card>

      <!-- Report result -->
      <el-card v-if="reportResult" shadow="never">
        <template #header>
          <strong>{{ reportResult.name }}</strong>
          <span style="float: right; color: #909399; font-size: 13px">期间：{{ reportResult.period }}</span>
        </template>
        <el-table :data="reportResult.data" border size="small" :max-height="tableMaxHeight">
          <el-table-column prop="label" label="项目" min-width="200">
            <template #default="{ row }">
              <span :style="{ fontWeight: row.bold ? 'bold' : 'normal', paddingLeft: (row.level - 1) * 20 + 'px' }">{{ row.label }}</span>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="150" align="right">
            <template #default="{ row }">{{ fmt(row.amount) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- Formula help -->
      <el-card shadow="never" style="margin-top: 16px">
        <template #header><strong>取数公式说明</strong></template>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="JE('code', 'dir')">本期发生额，如 JE('6602', '借')</el-descriptions-item>
          <el-descriptions-item label="QM('code', 'dir')">期末余额，如 QM('1002', '借')</el-descriptions-item>
          <el-descriptions-item label="QC('code', 'dir')">期初余额，如 QC('1001', '借')</el-descriptions-item>
          <el-descriptions-item label="JL('code', 'dir')">本年累计发生额，如 JL('5001', '贷')</el-descriptions-item>
          <el-descriptions-item label="加减运算">支持 + - 运算，如 JE('6601','借') - JE('6602','借')</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </div>

    <!-- Create dialog -->
    <el-dialog v-model="showAdd" title="新建自定义报表" :width="isMobile ? '95%' : '650px'">
      <el-form :model="form" label-width="80px">
        <el-form-item label="报表名称" required>
          <el-input v-model="form.name" placeholder="如：费用汇总表" />
        </el-form-item>
        <el-form-item label="行定义">
          <div v-for="(row, i) in form.rows" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <el-input v-model="row.label" placeholder="行标签" style="flex: 1" />
            <el-input v-model="row.formula" placeholder="取数公式" style="flex: 1" />
            <el-input-number v-model="row.level" :min="1" :max="4" size="small" style="width: 70px" controls-position="right" />
            <el-checkbox v-model="row.bold">粗体</el-checkbox>
            <el-button size="small" type="danger" link @click="form.rows.splice(i, 1)"><el-icon><Delete /></el-icon></el-button>
          </div>
          <el-button size="small" @click="form.rows.push({ label: '', formula: '', level: 1, bold: false })">
            <el-icon><Plus /></el-icon>添加行
          </el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdd = false">取消</el-button>
        <el-button type="primary" @click="saveTemplate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'
import { reportApi } from '../api'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const { isMobile } = useMobile()
const tableMaxHeight = isMobile.value ? 'calc(100vh - 400px)' : 'calc(100vh - 450px)'

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const templates = ref([])
const reportResult = ref(null)
const runPeriod = ref(new Date().toISOString().slice(0, 7))

const showAdd = ref(false)
const form = ref({ name: '', rows: [{ label: '', formula: '', level: 1, bold: false }] })

const loadTemplates = async () => {
  if (!currentBook.value) return
  const { data } = await reportApi.templates.list(currentBook.value)
  templates.value = data.data || []
}

const openAdd = () => {
  form.value = { name: '', rows: [{ label: '', formula: '', level: 1, bold: false }] }
  showAdd.value = true
}

const saveTemplate = async () => {
  if (!form.value.name) { ElMessage.warning('请输入报表名称'); return }
  try {
    await reportApi.templates.create(currentBook.value, {
      name: form.value.name,
      type: 'custom',
      config: JSON.stringify({ rows: form.value.rows })
    })
    ElMessage.success('保存成功')
    showAdd.value = false
    loadTemplates()
  } catch (e) { ElMessage.error('保存失败') }
}

const deleteTemplate = async (tpl) => {
  await ElMessageBox.confirm(`确定删除"${tpl.name}"？`, '确认')
  try {
    await reportApi.templates.delete(currentBook.value, tpl.id)
    ElMessage.success('已删除')
    loadTemplates()
    reportResult.value = null
  } catch (e) { ElMessage.error('删除失败') }
}

const runReport = async (tpl) => {
  if (!runPeriod.value) { ElMessage.warning('请选择期间'); return }
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/reports/custom/${tpl.id}?period=${runPeriod.value}`)
    reportResult.value = data
  } catch (e) { ElMessage.error('生成报表失败') }
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

watch(currentBook, (val) => { if (val) loadTemplates() })

onMounted(() => {
  if (currentBook.value) loadTemplates()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.template-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.template-card :deep(.el-card__body) { padding: 14px; }
.template-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.template-name { font-weight: 600; font-size: 15px; }
.template-meta { font-size: 13px; color: #909399; margin-bottom: 10px; }
.template-actions { display: flex; align-items: center; }
</style>
