<template>
  <div class="dashboard">
    <h2>工作台</h2>

    <!-- 第一行：原有统计 -->
    <div class="stats-grid">
      <el-card shadow="hover">
        <template #header>账套总数</template>
        <div class="stat-value">{{ stats.totalBooks }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>本月凭证（全部账套）</template>
        <div class="stat-value">{{ stats.monthVouchers }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待审核（全部账套）</template>
        <div class="stat-value warning">{{ stats.pendingReview }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>待记账（全部账套）</template>
        <div class="stat-value info">{{ stats.pendingPost }}</div>
      </el-card>
    </div>

    <!-- 第二行：当前账套财务概览 -->
    <div class="stats-grid" v-if="currentBook">
      <el-card shadow="hover">
        <template #header>本月收入</template>
        <div class="stat-value green">{{ formatWan(trend.revenue[currentMonthIdx]) }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>本月费用</template>
        <div class="stat-value orange">{{ formatWan(trend.expense[currentMonthIdx]) }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>本月利润</template>
        <div class="stat-value blue">{{ formatWan(trend.profit[currentMonthIdx]) }}</div>
      </el-card>
      <el-card shadow="hover">
        <template #header>累计利润（本年）</template>
        <div class="stat-value blue">{{ formatWan(yearProfit) }}</div>
      </el-card>
    </div>

    <!-- 第三行：收支趋势 + 快捷操作/系统信息 -->
    <div class="bottom-grid" v-if="currentBook">
      <el-card>
        <template #header>
          <div style="display:flex;justify-content:space-between;align-items:center">
            <span>收支趋势（近12个月）</span>
            <el-tag size="small">当前账套</el-tag>
          </div>
        </template>
        <div ref="trendChartRef" class="chart-box"></div>
      </el-card>
      <div class="side-col">
        <el-card>
          <template #header>快捷操作</template>
          <el-space wrap>
            <el-button type="primary" @click="$router.push('/books')">
              <el-icon><Plus /></el-icon>新建账套
            </el-button>
            <el-button @click="$router.push('/vouchers')">
              <el-icon><Document /></el-icon>录入凭证
            </el-button>
            <el-button @click="$router.push('/opening-balance')">
              <el-icon><Coin /></el-icon>期初余额
            </el-button>
            <el-button @click="$router.push('/reports')">
              <el-icon><DataAnalysis /></el-icon>查看报表
            </el-button>
            <el-button @click="$router.push('/closing')">
              <el-icon><SwitchButton /></el-icon>期末处理
            </el-button>
          </el-space>
        </el-card>
        <el-card>
          <template #header>系统信息</template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="版本">{{ systemInfo.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="数据库">SQLite</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag type="success" size="small">{{ systemInfo.status || '-' }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </div>
    </div>

    <!-- 第四行：费用饼图 + 利润趋势 -->
    <div class="bottom-grid" v-if="currentBook && trend.expense_breakdown && trend.expense_breakdown.length > 0">
      <el-card>
        <template #header>费用构成（本年）</template>
        <div ref="pieChartRef" class="chart-box"></div>
      </el-card>
      <el-card>
        <template #header>利润趋势（近12个月）</template>
        <div ref="profitChartRef" class="chart-box"></div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { bookApi, voucherApi, healthApi, reportApi } from '../api'
import { Coin, DataAnalysis, Document, SwitchButton } from '@element-plus/icons-vue'
import { useBookStore } from '../stores/book'
import * as echarts from 'echarts'

const { currentBookId: currentBook } = useBookStore()

const stats = ref({ totalBooks: 0, monthVouchers: 0, pendingReview: 0, pendingPost: 0 })
const systemInfo = ref({ version: '', status: '' })
const trend = ref({ months: [], revenue: [], expense: [], profit: [], expense_breakdown: [] })

const trendChartRef = ref(null)
const pieChartRef = ref(null)
const profitChartRef = ref(null)
let trendChart = null
let pieChart = null
let profitChart = null

const currentMonthIdx = computed(() => {
  return new Date().getMonth()
})

const yearProfit = computed(() => {
  return (trend.value.profit || []).reduce((sum, v) => sum + v, 0)
})

const formatWan = (v) => {
  if (v == null || isNaN(v)) return '-'
  if (Math.abs(v) >= 10000) return (v / 10000).toFixed(1) + '万'
  return v.toLocaleString('zh-CN', { maximumFractionDigits: 0 })
}

const loadStats = async () => {
  try {
    const { data } = await bookApi.list()
    const books = data.data || []
    stats.value.totalBooks = books.length

    const now = new Date()
    const monthPrefix = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
    let allVouchers = []
    for (const book of books) {
      const { data: vData } = await voucherApi.list(book.id)
      allVouchers = allVouchers.concat(vData.data || [])
    }
    stats.value.monthVouchers = allVouchers.filter(v => v.date && v.date.startsWith(monthPrefix)).length
    stats.value.pendingReview = allVouchers.filter(v => v.status === 'draft').length
    stats.value.pendingPost = allVouchers.filter(v => v.status === 'reviewed').length

    const { data: health } = await healthApi.check()
    systemInfo.value = health
  } catch (e) {
    console.error('Failed to load stats:', e)
  }
}

const loadTrend = async () => {
  if (!currentBook.value) return
  try {
    const year = new Date().getFullYear().toString()
    const { data } = await reportApi.monthlyTrend(currentBook.value, year)
    trend.value = data
    await nextTick()
    renderCharts()
  } catch (e) {
    console.error('Failed to load trend:', e)
  }
}

const renderCharts = () => {
  const d = trend.value
  if (!d.months || d.months.length === 0) return
  const shortMonths = d.months.map(m => m.split('-')[1] + '月')

  // 收支趋势
  if (trendChartRef.value) {
    if (!trendChart) trendChart = echarts.init(trendChartRef.value)
    trendChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['收入', '费用', '利润'], right: 10, top: 0 },
      grid: { left: 55, right: 15, top: 35, bottom: 25 },
      xAxis: { type: 'category', data: shortMonths, axisLabel: { fontSize: 11 } },
      yAxis: { type: 'value', axisLabel: { fontSize: 11, formatter: v => v >= 10000 ? (v/10000).toFixed(0) + '万' : v } },
      series: [
        { name: '收入', type: 'line', data: d.revenue, smooth: true, itemStyle: { color: '#67C23A' }, areaStyle: { color: 'rgba(103,194,58,0.08)' }, symbol: 'circle', symbolSize: 4 },
        { name: '费用', type: 'line', data: d.expense, smooth: true, itemStyle: { color: '#E6A23C' }, areaStyle: { color: 'rgba(230,162,60,0.08)' }, symbol: 'circle', symbolSize: 4 },
        { name: '利润', type: 'bar', data: d.profit, itemStyle: { color: 'rgba(64,158,255,0.5)' }, barWidth: 16 }
      ]
    })
  }

  // 费用饼图
  if (pieChartRef.value && d.expense_breakdown && d.expense_breakdown.length > 0) {
    if (!pieChart) pieChart = echarts.init(pieChartRef.value)
    const colors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399', '#b37feb', '#36cfc9']
    pieChart.setOption({
      tooltip: { trigger: 'item', formatter: '{b}: {c}万 ({d}%)' },
      legend: { orient: 'vertical', right: 10, top: 'center', textStyle: { fontSize: 12 } },
      series: [{
        type: 'pie', radius: ['35%', '65%'], center: ['38%', '50%'],
        label: { show: false },
        emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
        data: d.expense_breakdown.map((item, i) => ({
          value: item.value >= 10000 ? +(item.value / 10000).toFixed(2) : +item.value.toFixed(2),
          name: item.name,
          itemStyle: { color: colors[i % colors.length] }
        }))
      }]
    })
  }

  // 利润趋势
  if (profitChartRef.value) {
    if (!profitChart) profitChart = echarts.init(profitChartRef.value)
    profitChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 55, right: 15, top: 20, bottom: 25 },
      xAxis: { type: 'category', data: shortMonths, axisLabel: { fontSize: 11 } },
      yAxis: { type: 'value', axisLabel: { fontSize: 11, formatter: v => v >= 10000 ? (v/10000).toFixed(0) + '万' : v } },
      series: [{
        type: 'bar', data: d.profit, barWidth: 20,
        itemStyle: { color: p => p.value >= 0 ? '#409EFF' : '#F56C6C' },
        markLine: { data: [{ type: 'average', name: '平均', lineStyle: { type: 'dashed', color: '#909399' } }], label: { fontSize: 11 } }
      }]
    })
  }
}

const handleResize = () => {
  trendChart?.resize()
  pieChart?.resize()
  profitChart?.resize()
}

onMounted(() => {
  loadStats()
  loadTrend()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  pieChart?.dispose()
  profitChart?.dispose()
})

watch(currentBook, () => {
  trend.value = { months: [], revenue: [], expense: [], profit: [], expense_breakdown: [] }
  trendChart?.dispose(); trendChart = null
  pieChart?.dispose(); pieChart = null
  profitChart?.dispose(); profitChart = null
  loadTrend()
})
</script>

<style scoped>
.dashboard h2 { margin-bottom: 20px; color: #303133; font-size: 18px; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.bottom-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 16px;
  margin-top: 16px;
}

.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-value {
  font-size: 30px;
  font-weight: bold;
  color: #409EFF;
  text-align: center;
  padding: 8px 0;
}

.stat-value.warning { color: #E6A23C; }
.stat-value.info { color: #909399; }
.stat-value.green { color: #67C23A; }
.stat-value.orange { color: #E6A23C; }

.chart-box { height: 280px; }

@media (max-width: 767px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .bottom-grid { grid-template-columns: 1fr; gap: 12px; }
  .stat-value { font-size: 24px; }
  .chart-box { height: 240px; }
}
</style>
