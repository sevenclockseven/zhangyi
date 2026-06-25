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

      <div style="margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap">
        <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadReport" :size="isMobile ? 'small' : 'default'" />
        <el-button size="small" @click="exportReport" :disabled="!reportData">
          <el-icon><Download /></el-icon>导出
        </el-button>
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
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

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

const exportReport = () => {
  const token = localStorage.getItem('token')
  const typeMap = { 'income': 'income', 'balance-sheet': 'balance-sheet', 'account-balance': 'account-balance' }
  const type = typeMap[activeTab.value]
  if (!type) { ElMessage.warning('该报表暂不支持导出'); return }
  window.open(`/api/books/${currentBook.value}/reports/export?type=${type}&period=${period.value}&token=***}`, '_blank')
}

const fmt = (v) => {
  if (!v && v !== 0) return ''
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

onMounted(() => {
  loadBooks()
  window.addEventListener('resize', () => { isMobile.value = window.innerWidth < 768 })
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.report-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.report-stack { display: flex; flex-direction: column; gap: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
</style>
