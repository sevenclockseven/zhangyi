<template>
  <div class="reports">
    <div class="page-header">
      <h2>报表中心</h2>
      <el-select v-model="currentBook" placeholder="选择账套" :style="{ width: isMobile ? '100%' : '200px' }" @change="loadReport">
        <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
    </div>

    <el-tabs v-model="activeTab" v-if="currentBook" @tab-change="loadReport">
      <el-tab-pane label="利润表" name="income" />
      <el-tab-pane label="资产负债表" name="balance-sheet" />
      <el-tab-pane label="现金流量表" name="cash-flow" />
      <el-tab-pane label="费用统计" name="expense" />
      <el-tab-pane label="总账报表" name="general-ledger" />
      <el-tab-pane label="科目余额" name="account-balance" />
      <el-tab-pane label="应收统计" name="ar" />
      <el-tab-pane label="应付统计" name="ap" />
      <el-tab-pane label="自定义报表" name="custom" />

      <div style="margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap">
        <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadReport" :size="isMobile ? 'small' : 'default'" />
        <el-dropdown @command="exportReport" :disabled="!reportData">
          <el-button size="small" :disabled="!reportData">
            <el-icon><Download /></el-icon>导出 <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="csv">导出CSV</el-dropdown-item>
              <el-dropdown-item command="excel">导出Excel</el-dropdown-item>
              <el-dropdown-item command="print">打印</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <!-- 利润表 (新格式) -->
      <div v-if="activeTab === 'income' && reportData">
        <el-card shadow="never">
          <template #header><strong>利润表</strong><span style="float: right; color: #909399; font-size: 13px">期间：{{ period }}</span></template>
          <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" show-summary :summary-method="incomeSummary">
            <el-table-column prop="name" label="项目" min-width="200">
              <template #default="{ row }">
                <span :style="{ fontWeight: row.bold ? 'bold' : 'normal', paddingLeft: (row.level - 1) * 20 + 'px' }">{{ row.name }}</span>
              </template>
            </el-table-column>
            <el-table-column label="本期金额" width="150" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>

      <!-- 资产负债表 -->
      <div v-if="activeTab === 'balance-sheet' && reportData">
        <div :class="isMobile ? 'report-stack' : 'report-row'">
          <el-card shadow="never">
            <template #header><strong>资产</strong></template>
            <div class="table-wrapper">
              <el-table :data="reportData.assets" border size="small" show-summary :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="期末余额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.balance) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
          <el-card shadow="never">
            <template #header><strong>负债及权益</strong></template>
            <div class="table-wrapper">
              <el-table :data="[...(reportData.liabilities || []), ...(reportData.equity || [])]" border size="small" show-summary :max-height="tableMaxHeight">
                <el-table-column prop="code" label="编码" width="70" />
                <el-table-column prop="name" label="项目" min-width="120" />
                <el-table-column label="期末余额" width="120" align="right">
                  <template #default="{ row }">{{ fmt(row.balance) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
        </div>
      </div>

      <!-- 现金流量表 -->
      <div v-if="activeTab === 'cash-flow' && reportData">
        <el-card shadow="never">
          <template #header><strong>现金流量表</strong></template>
          <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="category" label="类别" width="100">
              <template #default="{ row }">
                {{ { operating: '经营活动', investing: '投资活动', financing: '筹资活动' }[row.category] || row.category }}
              </template>
            </el-table-column>
            <el-table-column prop="item_name" label="项目" min-width="200" />
            <el-table-column label="金额" width="140" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 12px; padding: 12px; background: #f5f7fa; border-radius: 4px; font-weight: bold">
            现金净增加额：{{ fmt(reportData.summary?.cash_increase) }}
          </div>
        </el-card>
      </div>

      <!-- 费用统计 -->
      <div v-if="activeTab === 'expense' && reportData">
        <el-card shadow="never">
          <template #header><strong>费用统计表</strong><span style="float: right; color: #909399; font-size: 13px">期间：{{ period }}</span></template>
          <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" show-summary>
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="费用项目" min-width="180" />
            <el-table-column label="本期金额" width="140" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card shadow="never" v-if="reportData.sub_items && reportData.sub_items.length > 0" style="margin-top: 12px">
          <template #header><strong>管理费用明细</strong></template>
          <el-table :data="reportData.sub_items" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="明细项目" min-width="180" />
            <el-table-column label="本期金额" width="140" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>

      <!-- 总账报表 -->
      <div v-if="activeTab === 'general-ledger' && reportData">
        <el-card shadow="never">
          <template #header><strong>总账报表</strong><span style="float: right; color: #909399; font-size: 13px">期间：{{ period }}</span></template>
          <div class="table-wrapper">
            <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" show-summary>
              <el-table-column prop="code" label="科目编码" width="100" fixed />
              <el-table-column prop="name" label="科目名称" min-width="130" fixed />
              <el-table-column prop="direction" label="向" width="50" align="center" />
              <el-table-column label="期初借" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.opening_debit) }}</template>
              </el-table-column>
              <el-table-column label="期初贷" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.opening_credit) }}</template>
              </el-table-column>
              <el-table-column label="本期借" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
              </el-table-column>
              <el-table-column label="本期贷" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
              </el-table-column>
              <el-table-column label="期末借" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.closing_debit) }}</template>
              </el-table-column>
              <el-table-column label="期末贷" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.closing_credit) }}</template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </div>

      <!-- 科目余额表 -->
      <div v-if="activeTab === 'account-balance' && reportData">
        <div class="table-wrapper">
          <el-table :data="reportData" border size="small" show-summary :max-height="tableMaxHeight">
            <el-table-column prop="account_code" label="编码" width="90" fixed />
            <el-table-column prop="account_name" label="科目" min-width="120" fixed />
            <el-table-column prop="direction" label="向" width="50" align="center" />
            <el-table-column label="期初" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.opening_debit || row.opening_credit) }}</template>
            </el-table-column>
            <el-table-column label="本期借" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
            </el-table-column>
            <el-table-column label="本期贷" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
            </el-table-column>
            <el-table-column label="期末" width="100" align="right">
              <template #default="{ row }">{{ fmt(row.closing_debit || row.closing_credit) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 应收/应付统计 -->
      <div v-if="(activeTab === 'ar' || activeTab === 'ap') && reportData">
        <el-card shadow="never">
          <template #header>
            <strong>{{ activeTab === 'ar' ? '应收账款统计及帐龄分析' : '应付账款统计及帐龄分析' }}</strong>
          </template>
          <div class="table-wrapper">
            <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" show-summary>
              <el-table-column prop="code" label="编码" width="80" fixed />
              <el-table-column prop="name" :label="activeTab === 'ar' ? '客户' : '供应商'" min-width="120" fixed />
              <el-table-column label="合计" width="110" align="right">
                <template #default="{ row }">{{ fmt(row.total) }}</template>
              </el-table-column>
              <el-table-column label="未到期" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.current) }}</template>
              </el-table-column>
              <el-table-column label="1个月内" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.month_1) }}</template>
              </el-table-column>
              <el-table-column label="1-3月" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.month_3) }}</template>
              </el-table-column>
              <el-table-column label="3-6月" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.month_6) }}</template>
              </el-table-column>
              <el-table-column label="6-12月" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.month_12) }}</template>
              </el-table-column>
              <el-table-column label="1年以上" width="100" align="right">
                <template #default="{ row }">{{ fmt(row.over_1_year) }}</template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </div>
      <!-- 自定义报表 -->
      <div v-if="activeTab === 'custom'">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-button type="primary" size="small" @click="openCrAdd">
            <el-icon><Plus /></el-icon>新建报表
          </el-button>
          <el-date-picker v-model="crRunPeriod" type="month" value-format="YYYY-MM" placeholder="期间" size="small" style="width: 140px" />
        </div>
        <el-table :data="crList" border size="small">
          <el-table-column prop="name" label="报表名称" min-width="180" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column label="操作" width="180">
            <template #default="{ row }">
              <el-button size="small" type="success" link @click="runCr(row)">运行</el-button>
              <el-button size="small" type="primary" link @click="editCr(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="deleteCr(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-card v-if="crResult" shadow="never" style="margin-top: 12px">
          <template #header><strong>{{ crResult.name }}</strong><span style="float: right; color: #909399; font-size: 13px">{{ crResult.period }}</span></template>
          <el-table :data="crResult.data" border size="small">
            <el-table-column prop="label" label="项目" min-width="200">
              <template #default="{ row }">
                <span :style="{ fontWeight: row.bold ? 'bold' : 'normal', paddingLeft: (row.level - 1) * 20 + 'px' }">{{ row.label }}</span>
              </template>
            </el-table-column>
            <el-table-column label="金额" width="140" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card shadow="never" style="margin-top: 12px">
          <template #header><strong>取数公式</strong></template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="JE('code','dir')">本期发生额</el-descriptions-item>
            <el-descriptions-item label="QM('code','dir')">期末余额</el-descriptions-item>
            <el-descriptions-item label="QC('code','dir')">期初余额</el-descriptions-item>
            <el-descriptions-item label="JL('code','dir')">本年累计</el-descriptions-item>
            <el-descriptions-item label="运算">支持 + - ，如 JE('6601','借') - JE('6602','借')</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </div>
    </el-tabs>

    <!-- Custom Report Edit Dialog -->
    <el-dialog v-model="showCrEdit" :title="crForm.id ? '编辑自定义报表' : '新建自定义报表'" :width="isMobile ? '95%' : '650px'">
      <el-form :model="crForm" label-width="80px">
        <el-form-item label="报表名称" required><el-input v-model="crForm.name" placeholder="如：费用汇总表" /></el-form-item>
        <el-form-item label="行定义">
          <div v-for="(row, i) in crForm.rows" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <el-input v-model="row.label" placeholder="行标签" style="flex: 1" />
            <el-input v-model="row.formula" placeholder="如 JE('6602','借')" style="flex: 1" />
            <el-input-number v-model="row.level" :min="1" :max="4" size="small" style="width: 70px" />
            <el-checkbox v-model="row.bold">粗</el-checkbox>
            <el-button size="small" type="danger" link @click="crForm.rows.splice(i, 1)"><el-icon><Delete /></el-icon></el-button>
          </div>
          <el-button size="small" @click="crForm.rows.push({ label: '', formula: '', level: 1, bold: false })"><el-icon><Plus /></el-icon>添加行</el-button>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="showCrEdit = false">取消</el-button><el-button type="primary" @click="saveCr">保存</el-button></template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const isMobile = ref(window.innerWidth < 768)
const tableMaxHeight = isMobile.value ? 'calc(100vh - 320px)' : 'calc(100vh - 350px)'

const books = ref([])
const currentBook = ref(null)
const activeTab = ref('income')
const period = ref(new Date().toISOString().slice(0, 7))
const reportData = ref(null)

const loadBooks = async () => {
  const { data } = await axios.get('/api/books')
  books.value = data.data || []
  if (books.value.length > 0) currentBook.value = books.value[0].id
}

const loadReport = async () => {
  if (activeTab.value === 'custom') { loadCrList(); return }
  if (!currentBook.value || !period.value) return
  reportData.value = null
  try {
    const base = `/api/books/${currentBook.value}/reports`
    if (activeTab.value === 'income') {
      const { data } = await axios.get(`${base}/income-statement-v2?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'balance-sheet') {
      const { data } = await axios.get(`${base}/balance-sheet?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'cash-flow') {
      const { data } = await axios.get(`${base}/cash-flow?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'expense') {
      const { data } = await axios.get(`${base}/expense?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'general-ledger') {
      const { data } = await axios.get(`${base}/general-ledger?period=${period.value}`)
      reportData.value = data
    } else if (activeTab.value === 'account-balance') {
      const { data } = await axios.get(`${base}/account-balance?period=${period.value}`)
      reportData.value = data.data || []
    } else if (activeTab.value === 'ar') {
      const { data } = await axios.get(`${base}/ar-ap?type=ar`)
      reportData.value = data
    } else if (activeTab.value === 'ap') {
      const { data } = await axios.get(`${base}/ar-ap?type=ap`)
      reportData.value = data
    }
  } catch (e) { console.error(e) }
}

const incomeSummary = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '净利润'; return }
    if (i === 1) {
      const netRow = data.find(r => r.name === '四、净利润')
      sums[i] = fmt(netRow ? netRow.amount : 0)
    }
  })
  return sums
}

const exportReport = async (format) => {
  if (activeTab.value === 'custom') { ElMessage.warning('自定义报表请使用运行结果导出'); return }
  if (!reportData.value) { ElMessage.warning('请先加载报表数据'); return }

  if (format === 'csv') {
    exportCSV()
  } else if (format === 'print') {
    printReport()
  } else if (format === 'excel') {
    exportExcel()
  }
}

const exportCSV = () => {
  const tables = document.querySelectorAll('.el-table__body-wrapper table')
  if (!tables.length) { ElMessage.warning('无数据可导出'); return }
  let csv = '\uFEFF' // BOM for Excel
  tables.forEach((table, ti) => {
    if (ti > 0) csv += '\n'
    // Headers
    const headers = []
    table.querySelectorAll('thead th').forEach(th => {
      headers.push('"' + th.innerText.replace(/"/g, '""') + '"')
    })
    csv += headers.join(',') + '\n'
    // Rows
    table.querySelectorAll('tbody tr').forEach(tr => {
      const row = []
      tr.querySelectorAll('td').forEach(td => {
        row.push('"' + td.innerText.replace(/"/g, '""') + '"')
      })
      csv += row.join(',') + '\n'
    })
  })
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${activeTab.value}_${period.value}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

const printReport = () => {
  const el = document.querySelector('.reports .el-tabs__content')
  if (!el) { ElMessage.warning('无数据可打印'); return }
  const printWin = window.open('', '_blank')
  printWin.document.write(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>报表打印</title>
    <style>
      body { font-family: 'Microsoft YaHei', sans-serif; padding: 20px; }
      table { border-collapse: collapse; width: 100%; margin-bottom: 16px; }
      th, td { border: 1px solid #ccc; padding: 6px 10px; text-align: left; font-size: 13px; }
      th { background: #f5f7fa; font-weight: bold; }
      .el-card { border: none !important; box-shadow: none !important; }
      .el-card__header { padding: 8px 0 !important; border-bottom: 1px solid #eee; }
      h3 { margin: 0 0 8px; }
      .no-print { display: none !important; }
      @media print { body { padding: 0; } }
    </style></head><body>${el.innerHTML}</body></html>`)
  printWin.document.close()
  printWin.focus()
  setTimeout(() => { printWin.print(); printWin.close(); }, 300)
}

const exportExcel = () => {
  const tables = document.querySelectorAll('.el-table__body-wrapper table')
  if (!tables.length) { ElMessage.warning('无数据可导出'); return }
  let tableHTML = ''
  tables.forEach(t => { tableHTML += t.outerHTML + '<br>' })
  const html = `<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
    <head><meta charset="utf-8">
    <!--[if gte mso 9]><xml><x:ExcelWorkbook><x:ExcelWorksheets><x:ExcelWorksheet>
    <x:Name>报表</x:Name><x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions>
    </x:ExcelWorksheet></x:ExcelWorksheets></x:ExcelWorkbook></xml><![endif]-->
    <style>table{border-collapse:collapse} th,td{border:1px solid #000;padding:4px 8px;font-size:12px} th{background:#f0f0f0;font-weight:bold}</style>
    </head><body>${tableHTML}</body></html>`
  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${activeTab.value}_${period.value}.xls`
  a.click()
  URL.revokeObjectURL(url)
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
// Custom Reports
const crList = ref([])
const crRunPeriod = ref(new Date().toISOString().slice(0, 7))
const crResult = ref(null)
const showCrEdit = ref(false)
const crForm = ref({ name: '', rows: [{ label: '', formula: '', level: 1, bold: false }] })

const loadCrList = async () => {
  if (!currentBook.value) return
  const { data } = await axios.get(`/api/books/${currentBook.value}/reports/templates`)
  crList.value = data.data || []
}
const openCrAdd = () => {
  crForm.value = { id: null, name: '', rows: [{ label: '', formula: '', level: 1, bold: false }] }
  showCrEdit.value = true
}
const editCr = (tpl) => {
  let rows = [{ label: '', formula: '', level: 1, bold: false }]
  try {
    const config = JSON.parse(tpl.config)
    if (config.rows && config.rows.length > 0) rows = config.rows
  } catch (e) {}
  crForm.value = { id: tpl.id, name: tpl.name, rows }
  showCrEdit.value = true
}
const saveCr = async () => {
  if (!crForm.value.name) { ElMessage.warning('请输入报表名称'); return }
  try {
    const payload = {
      name: crForm.value.name, type: 'custom', config: JSON.stringify({ rows: crForm.value.rows })
    }
    if (crForm.value.id) {
      await axios.put(`/api/books/${currentBook.value}/reports/templates/${crForm.value.id}`, payload)
    } else {
      await axios.post(`/api/books/${currentBook.value}/reports/templates`, payload)
    }
    ElMessage.success('保存成功')
    showCrEdit.value = false
    loadCrList()
  } catch (e) { ElMessage.error('保存失败') }
}
const runCr = async (tpl) => {
  if (!crRunPeriod.value) { ElMessage.warning('请选择期间'); return }
  try {
    const { data } = await axios.get(`/api/books/${currentBook.value}/reports/custom/${tpl.id}?period=${crRunPeriod.value}`)
    crResult.value = data
  } catch (e) { ElMessage.error('生成失败') }
}
const deleteCr = async (tpl) => {
  await ElMessageBox.confirm(`确定删除"${tpl.name}"？`, '确认')
  await axios.delete(`/api/books/${currentBook.value}/reports/templates/${tpl.id}`)
  ElMessage.success('已删除')
  loadCrList()
  crResult.value = null
}

</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.report-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.report-stack { display: flex; flex-direction: column; gap: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
