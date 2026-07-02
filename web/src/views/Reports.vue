<template>
  <div class="reports">
    <div class="page-header">
      <h2>报表中心</h2>
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
      <el-tab-pane label="图表分析" name="charts" />

      <div style="margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; align-items: center">
        <template v-if="activeTab !== 'charts'">
          <el-date-picker v-model="period" type="month" value-format="YYYY-MM" placeholder="期间" @change="loadReport()" :size="isMobile ? 'small' : 'default'" />
          <el-dropdown @command="exportReport" :disabled="!reportData && !crResult">
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
        </template>
        <template v-else>
          <el-button size="small" :disabled="!chartTrend || !chartPie || !chartProfit" @click="exportChartPNG">
            <el-icon><Download /></el-icon>导出PNG
          </el-button>
          <el-button size="small" :disabled="!chartTrend || !chartPie || !chartProfit" @click="printChartReport">
            打印图表
          </el-button>
        </template>
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
                <el-table-column prop="balance" label="期末余额" width="120" align="right">
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
                <el-table-column prop="balance" label="期末余额" width="120" align="right">
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
          <template #header><strong>现金流量表</strong><span style="float: right; color: #909399; font-size: 13px">期间：{{ period }}</span></template>

          <!-- 期初/期末现金余额 -->
          <div style="margin-bottom: 12px; padding: 10px 12px; background: #f0f9eb; border-radius: 4px; display: flex; gap: 24px; flex-wrap: wrap; font-size: 13px">
            <span>期初现金余额：<strong>{{ fmt(reportData.balance?.opening_cash) }}</strong></span>
            <span>期末现金余额：<strong>{{ fmt(reportData.balance?.closing_cash) }}</strong></span>
            <span>实际现金变动：<strong :style="{ color: (reportData.balance?.actual_increase || 0) >= 0 ? '#67C23A' : '#F56C6C' }">{{ fmt(reportData.balance?.actual_increase) }}</strong></span>
          </div>

          <!-- 勾稽校验警告 -->
          <el-alert
            v-if="reportData.balance && !reportData.balance.reconciled"
            type="error"
            :closable="false"
            show-icon
            style="margin-bottom: 12px"
          >
            <template #title>
              勾稽校验不通过：现金流量表净增加额（{{ fmt(reportData.summary?.cash_increase) }}）与实际现金变动（{{ fmt(reportData.balance?.actual_increase) }}）差异 {{ fmt(Math.abs((reportData.summary?.cash_increase || 0) - (reportData.balance?.actual_increase || 0))) }} 元，请检查是否有未标记的凭证。
            </template>
          </el-alert>

          <!-- 未标记凭证警告 -->
          <el-alert
            v-if="reportData.warnings?.untagged_count > 0"
            type="warning"
            :closable="false"
            show-icon
            style="margin-bottom: 12px"
          >
            <template #title>
              <span>发现 <strong>{{ reportData.warnings.untagged_count }}</strong> 笔涉及现金科目但未标记现金流量的凭证</span>
              <el-button type="warning" link size="small" @click="showUntagged = !showUntagged" style="margin-left: 8px">
                {{ showUntagged ? '收起' : '查看详情' }}
              </el-button>
            </template>
          </el-alert>

          <!-- 未标记凭证明细 -->
          <el-table
            v-if="showUntagged && reportData.warnings?.untagged_items?.length"
            :data="reportData.warnings.untagged_items"
            border size="small" style="margin-bottom: 12px"
            :header-cell-style="{ background: '#fdf6ec', color: '#E6A23C' }"
          >
            <el-table-column prop="voucher_date" label="日期" width="110" />
            <el-table-column prop="voucher_no" label="凭证号" width="90" />
            <el-table-column prop="account_code" label="科目编码" width="90" />
            <el-table-column prop="account_name" label="科目名称" width="120" />
            <el-table-column label="借方" width="110" align="right">
              <template #default="{ row }">{{ row.debit ? fmt(row.debit) : '' }}</template>
            </el-table-column>
            <el-table-column label="贷方" width="110" align="right">
              <template #default="{ row }">{{ row.credit ? fmt(row.credit) : '' }}</template>
            </el-table-column>
            <el-table-column prop="memo" label="摘要" min-width="150" show-overflow-tooltip />
          </el-table>

          <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" row-key="item_code">
            <el-table-column prop="category" label="类别" width="100">
              <template #default="{ row }">
                <el-tag :type="{ operating: '', investing: 'success', financing: 'warning' }[row.category]" size="small">
                  {{ { operating: '经营活动', investing: '投资活动', financing: '筹资活动' }[row.category] || row.category }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="item_code" label="编码" width="90" />
            <el-table-column prop="item_name" label="项目" min-width="200" />
            <el-table-column label="金额" width="140" align="right">
              <template #default="{ row }">{{ fmt(row.amount) }}</template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 12px; padding: 12px; background: #f5f7fa; border-radius: 4px; display: flex; justify-content: space-between; flex-wrap: wrap; gap: 12px">
            <span>经营活动净额：<strong>{{ fmt(reportData.summary?.operating_total) }}</strong></span>
            <span>投资活动净额：<strong>{{ fmt(reportData.summary?.investing_total) }}</strong></span>
            <span>筹资活动净额：<strong>{{ fmt(reportData.summary?.financing_total) }}</strong></span>
            <span style="color: #409EFF; font-size: 16px">现金净增加额：<strong>{{ fmt(reportData.summary?.cash_increase) }}</strong></span>
          </div>
        </el-card>
      </div>

      <!-- 费用统计 -->
      <div v-if="activeTab === 'expense' && reportData">
        <el-card shadow="never">
          <template #header><strong>费用统计表</strong><span style="float: right; color: #909399; font-size: 13px">期间：{{ period }}</span></template>
          <el-table :data="reportData.data" border size="small" :max-height="tableMaxHeight" show-summary :summary-method="expenseSummary">
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" label="费用项目" min-width="180" />
            <el-table-column prop="amount" label="本期金额" width="140" align="right">
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
              <el-table-column label="期初借" width="100" align="right" prop="opening_debit">
                <template #default="{ row }">{{ fmt(row.opening_debit) }}</template>
              </el-table-column>
              <el-table-column label="期初贷" width="100" align="right" prop="opening_credit">
                <template #default="{ row }">{{ fmt(row.opening_credit) }}</template>
              </el-table-column>
              <el-table-column label="本期借" width="100" align="right" prop="period_debit">
                <template #default="{ row }">{{ fmt(row.period_debit) }}</template>
              </el-table-column>
              <el-table-column label="本期贷" width="100" align="right" prop="period_credit">
                <template #default="{ row }">{{ fmt(row.period_credit) }}</template>
              </el-table-column>
              <el-table-column label="期末借" width="100" align="right" prop="closing_debit">
                <template #default="{ row }">{{ fmt(row.closing_debit) }}</template>
              </el-table-column>
              <el-table-column label="期末贷" width="100" align="right" prop="closing_credit">
                <template #default="{ row }">{{ fmt(row.closing_credit) }}</template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </div>

      <!-- 科目余额表（树形） -->
      <div v-if="activeTab === 'account-balance' && reportData">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <el-button size="small" @click="expandAllBalance"><el-icon><Plus /></el-icon>全部展开</el-button>
          <el-button size="small" @click="collapseAllBalance"><el-icon><Minus /></el-icon>全部折叠</el-button>
        </div>
        <div class="table-wrapper">
          <el-table
            ref="balanceTableRef"
            :data="reportData"
            row-key="account_code"
            :tree-props="{ children: 'children' }"
            :expand-row-keys="expandRowKeys"
            :max-height="tableMaxHeight"
            border
            size="small"
            show-summary
            :summary-method="balanceSummaryMethod"
            row-class-name="balanceRowClassName"
            :stripe="false"
          >
            <el-table-column prop="account_code" label="编码" width="90" fixed />
            <el-table-column prop="account_name" label="科目" min-width="160" fixed>
              <template #default="{ row }">
                <span :style="{ fontWeight: row.level === 1 ? 'bold' : 'normal', paddingLeft: (row.level - 1) * 16 + 'px' }">{{ row.account_name }}</span>
              </template>
            </el-table-column>
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
              <el-table-column label="合计" width="110" align="right" prop="total">
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
        <el-card v-if="crResult" shadow="never" style="margin-top: 12px" class="cr-result-card">
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
          <template #header><strong>使用说明</strong></template>
          <div style="font-size: 13px; color: #606266; line-height: 1.8">
            <p><strong>1. 新建报表：</strong>点击「新建报表」，填写报表名称，添加行定义。每行需要设置：</p>
            <ul style="padding-left: 20px; margin: 4px 0">
              <li><strong>行标签</strong> — 显示在报表中的名称，如「营业收入」「管理费用」</li>
              <li><strong>取数公式</strong> — 从科目取数据的公式（见下方公式说明）</li>
              <li><strong>层级</strong> — 控制缩进，1=顶级，2=二级，以此类推</li>
              <li><strong>粗体</strong> — 勾选后该行加粗显示（适合小计/合计行）</li>
            </ul>
            <p><strong>2. 运行报表：</strong>选择期间后点击「运行」，即可查看结果。</p>
            <p><strong>3. 导出：</strong>运行后点击右上角「导出」按钮，可导出CSV/Excel或打印。</p>
            <p style="margin-top: 8px"><strong>取数公式说明：</strong></p>
            <el-table :data="formulaExamples" border size="small" style="margin: 8px 0">
              <el-table-column prop="formula" label="公式" width="200" />
              <el-table-column prop="desc" label="说明" />
              <el-table-column prop="example" label="示例" width="220" />
            </el-table>
            <p><strong>运算：</strong>支持 <code>+</code> <code>-</code> 组合，如 <code>JE('6601','借') - JE('6602','借')</code></p>
            <p><strong>科目编码：</strong>填写末级科目编码，如 6602（管理费用）。dir 填「借」或「贷」。</p>
          </div>
        </el-card>
      </div>
    </el-tabs>

    <!-- 图表分析 -->
    <div v-if="activeTab === 'charts' && currentBook" style="margin-top: 12px">
      <el-card shadow="never">
        <template #header>
          <div style="display:flex;justify-content:space-between;align-items:center">
            <strong>图表分析</strong>
            <div style="display:flex;gap:8px;align-items:center">
              <span style="font-size:13px;color:#909399">年度：</span>
              <el-date-picker v-model="chartYear" type="year" value-format="YYYY" placeholder="选择年份" size="small" @change="loadChartData" style="width:100px" />
            </div>
          </div>
        </template>
        <div class="chart-grid">
          <div>
            <div class="chart-label">收支趋势</div>
            <div ref="chartTrendRef" class="chart-box"></div>
          </div>
          <div>
            <div class="chart-label">费用构成</div>
            <div ref="chartPieRef" class="chart-box"></div>
          </div>
        </div>
        <div style="margin-top:16px">
          <div class="chart-label">利润趋势</div>
          <div ref="chartProfitRef" class="chart-box-tall"></div>
        </div>
      </el-card>
    </div>

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
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'
import { reportApi } from '../api'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, ArrowDown, Plus, Minus } from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const { isMobile } = useMobile()
const tableMaxHeight = isMobile.value ? 'calc(100vh - 320px)' : 'calc(100vh - 350px)'

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const activeTab = ref('income')
const period = ref(new Date().toISOString().slice(0, 7))
const reportData = ref(null)
const showUntagged = ref(false)
const balanceTableRef = ref(null)

const expandRowKeys = ref([])

// 图表分析
const chartYear = ref(new Date().getFullYear().toString())
const chartTrendRef = ref(null)
const chartPieRef = ref(null)
const chartProfitRef = ref(null)
let chartTrend = null
let chartPie = null
let chartProfit = null

const loadChartData = async () => {
  if (!currentBook.value) return
  try {
    const { data } = await reportApi.monthlyTrend(currentBook.value, chartYear.value)
    await nextTick()
    // Re-init charts if DOM was destroyed and re-created by v-if
    if (chartTrendRef.value && (!chartTrend || chartTrend.isDisposed())) {
      chartTrend = echarts.init(chartTrendRef.value)
    }
    if (chartPieRef.value && (!chartPie || chartPie.isDisposed())) {
      chartPie = echarts.init(chartPieRef.value)
    }
    if (chartProfitRef.value && (!chartProfit || chartProfit.isDisposed())) {
      chartProfit = echarts.init(chartProfitRef.value)
    }
    renderChartTrend(data)
    renderChartPie(data)
    renderChartProfit(data)
  } catch (e) { console.error(e) }
}

const renderChartTrend = (d) => {
  if (!chartTrendRef.value || !d.months) return
  if (!chartTrend || chartTrend.isDisposed()) chartTrend = echarts.init(chartTrendRef.value)
  const shortMonths = d.months.map(m => m.split('-')[1] + '月')
  chartTrend.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['收入', '费用'], right: 10, top: 0 },
    grid: { left: 55, right: 15, top: 35, bottom: 25 },
    xAxis: { type: 'category', data: shortMonths },
    yAxis: { type: 'value', axisLabel: { formatter: v => v >= 10000 ? (v/10000).toFixed(0) + '万' : v } },
    series: [
      { name: '收入', type: 'bar', data: d.revenue, itemStyle: { color: '#67C23A' } },
      { name: '费用', type: 'bar', data: d.expense, itemStyle: { color: '#E6A23C' } }
    ]
  })
}

const renderChartPie = (d) => {
  if (!chartPieRef.value || !d.expense_breakdown || d.expense_breakdown.length === 0) return
  if (!chartPie || chartPie.isDisposed()) chartPie = echarts.init(chartPieRef.value)
  const colors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399', '#b37feb', '#36cfc9']
  chartPie.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c}万 ({d}%)' },
    legend: { orient: 'vertical', right: 10, top: 'center' },
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

const renderChartProfit = (d) => {
  if (!chartProfitRef.value || !d.months) return
  if (!chartProfit || chartProfit.isDisposed()) chartProfit = echarts.init(chartProfitRef.value)
  const shortMonths = d.months.map(m => m.split('-')[1] + '月')
  chartProfit.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 55, right: 15, top: 20, bottom: 25 },
    xAxis: { type: 'category', data: shortMonths },
    yAxis: { type: 'value', axisLabel: { formatter: v => v >= 10000 ? (v/10000).toFixed(0) + '万' : v } },
    series: [{
      type: 'bar', data: d.profit, barWidth: 20,
      itemStyle: { color: p => p.value >= 0 ? '#409EFF' : '#F56C6C' },
      markLine: { data: [{ type: 'average', name: '平均', lineStyle: { type: 'dashed', color: '#909399' } }], label: { fontSize: 11 } }
    }]
  })
}

const mergeChartImages = () => {
  const dpr = 2
  const gap = 20
  const titleHeight = 50
  const charts = [chartTrend, chartPie, chartProfit].filter(c => c)
  if (!charts.length) return null

  let maxWidth = 0
  const images = charts.map(c => {
    const img = new Image()
    img.src = c.getDataURL({ pixelRatio: dpr, backgroundColor: '#fff' })
    if (c === chartTrend) maxWidth = img.width || 600
    return img
  })

  // Wait for all images to load
  return new Promise((resolve) => {
    let loaded = 0
    const check = () => {
      loaded++
      if (loaded < images.length) return
      const totalHeight = images.reduce((s, img) => s + img.height, 0) + gap * (images.length - 1) + titleHeight
      const w = Math.max(...images.map(img => img.width), 600)

      const canvas = document.createElement('canvas')
      canvas.width = w
      canvas.height = totalHeight
      const ctx = canvas.getContext('2d')

      // Title
      ctx.fillStyle = '#303133'
      ctx.font = `bold ${16 * dpr}px Microsoft YaHei, sans-serif`
      ctx.textAlign = 'center'
      ctx.fillText(`图表分析 - ${chartYear.value}年`, w / 2, 30 * dpr)

      let y = titleHeight
      images.forEach(img => {
        ctx.drawImage(img, 0, y)
        y += img.height + gap
      })

      resolve(canvas.toDataURL('image/png'))
    }
    images.forEach(img => {
      if (img.complete) { check() } else { img.onload = check; img.onerror = check }
    })
  })
}

const exportChartPNG = async () => {
  try {
    const dataURL = await mergeChartImages()
    if (!dataURL) { ElMessage.warning('无可导出的图表'); return }
    const a = document.createElement('a')
    a.href = dataURL
    a.download = `图表分析_${chartYear.value}.png`
    a.click()
    ElMessage.success('导出成功')
  } catch (e) { console.error(e); ElMessage.error('导出失败') }
}

const printChartReport = async () => {
  try {
    const dataURL = await mergeChartImages()
    if (!dataURL) { ElMessage.warning('无可打印的图表'); return }
    const printWin = window.open('', '_blank')
    printWin.document.write(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>图表分析 ${chartYear.value}</title>
      <style>*{margin:0;padding:0;box-sizing:border-box}body{padding:15mm;text-align:center}img{max-width:100%;height:auto}@media print{body{padding:10mm}@page{margin:10mm}}</style></head><body>
      <img src="${dataURL}"></body></html>`)
    printWin.document.close()
    printWin.focus()
    setTimeout(() => { printWin.print(); printWin.close() }, 300)
  } catch (e) { console.error(e); ElMessage.error('打印失败') }
}

const expandAllBalance = () => {
  // Use nextTick to ensure table is rendered with new data
  nextTick(() => {
    const allKeys = []
    const walk = (nodes) => {
      for (const n of nodes) {
        if (n.children && n.children.length) {
          allKeys.push(n.account_code)
          walk(n.children)
        }
      }
    }
    walk(reportData.value || [])
    expandRowKeys.value = allKeys
  })
}

const collapseAllBalance = () => {
  nextTick(() => {
    expandRowKeys.value = []
  })
}

const balanceRowClassName = ({ row }) => {
  if (row.level === 1) {
    const code = row.account_code || ''
    if (code.startsWith('1')) return 'row-asset'
    if (code.startsWith('2')) return 'row-liability'
    if (code.startsWith('3')) return 'row-equity'
    if (code.startsWith('4')) return 'row-cost'
    if (code.startsWith('5')) return 'row-expense'
  }
  return ''
}

const balanceSummaryMethod = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i === 1 || i === 2) { sums[i] = ''; return }
    const key = ['opening_debit', 'opening_credit', 'period_debit', 'period_credit', 'closing_debit', 'closing_credit'][i - 3]
    if (key) sums[i] = fmt(data.reduce((s, r) => s + (r[key] || 0), 0))
    else sums[i] = ''
  })
  return sums
}

const loadReport = async () => {
  if (activeTab.value === 'custom') { loadCrList(); return }
  if (activeTab.value === 'charts') { loadChartData(); return }
  if (!currentBook.value || !period.value) return
  reportData.value = null
  showUntagged.value = false
  try {
    const base = `/api/books/${currentBook.value}/reports`
    if (activeTab.value === 'income') {
      const { data } = await reportApi.incomeStatementV2(currentBook.value, period.value)
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
      // Auto-expand all after load
      expandAllBalance()
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

const expenseSummary = ({ columns, data }) => {
  const sums = []
  columns.forEach((col, i) => {
    if (i === 0) { sums[i] = '合计'; return }
    if (i === 1) { sums[i] = ''; return }
    if (i === 2) {
      const total = data.reduce((s, r) => s + (r.amount || 0), 0)
      sums[i] = fmt(total)
    }
  })
  return sums
}

const exportReport = async (format) => {
  if (activeTab.value === 'custom' && !crResult.value) { ElMessage.warning('请先运行自定义报表，再导出'); return }
  if (activeTab.value !== 'custom' && !reportData.value) { ElMessage.warning('请先加载报表数据'); return }

  if (format === 'csv') {
    exportCSV()
  } else if (format === 'print') {
    printReport()
  } else if (format === 'excel') {
    exportExcel()
  }
}

const getReportTables = () => {
  if (activeTab.value === 'custom' && crResult.value) {
    // Custom report: only get the result card table
    const resultCard = document.querySelector('.cr-result-card')
    return resultCard ? resultCard.querySelectorAll('.el-table__body-wrapper table') : []
  }
  return document.querySelectorAll('.el-tabs__content .el-table__body-wrapper table')
}

const exportCSV = () => {
  const tables = getReportTables()
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
  let printHTML = ''
  if (activeTab.value === 'custom' && crResult.value) {
    // Custom report: only print title + result table
    const resultCard = document.querySelector('.cr-result-card')
    if (!resultCard) { ElMessage.warning('无数据可打印'); return }
    const clone = resultCard.cloneNode(true)
    clone.querySelectorAll('button').forEach(el => el.remove())
    printHTML = `<h2>${crResult.value.name || '自定义报表'}</h2><p style="text-align:center;color:#909399;margin-bottom:12px">期间：${crResult.value.period || ''}</p>` + clone.innerHTML
  } else {
    // Standard report: get visible report section
    const activePane = document.querySelector('.el-tabs__content')
    if (!activePane) { ElMessage.warning('无数据可打印'); return }
    const clone = activePane.cloneNode(true)
    clone.querySelectorAll('button, .el-dropdown, .el-date-picker, .el-date-editor, .el-input, .el-select, .no-print').forEach(el => el.remove())
    clone.querySelectorAll('div').forEach(div => {
      if (div.children.length === 0 && div.textContent.trim() === '') div.remove()
    })
    printHTML = clone.innerHTML
  }
  const printWin = window.open('', '_blank')
  printWin.document.write(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>${activeTab.value === 'custom' ? crResult.value?.name || '自定义报表' : '报表打印'}</title>
    <style>
      * { margin: 0; padding: 0; box-sizing: border-box; }
      body { font-family: 'Microsoft YaHei', 'SimSun', sans-serif; padding: 15mm; color: #333; }
      h2 { font-size: 18px; margin-bottom: 12px; text-align: center; }
      table { border-collapse: collapse; width: 100%; margin-bottom: 12px; font-size: 12px; }
      th, td { border: 1px solid #999; padding: 4px 8px; text-align: left; }
      th { background: #f0f0f0; font-weight: bold; }
      td { background: #fff; }
      .el-card { border: none !important; box-shadow: none !important; margin-bottom: 12px; }
      .el-card__header { padding: 6px 0 !important; border-bottom: 1px solid #ddd; font-size: 14px; }
      .el-card__body { padding: 8px 0 !important; }
      .el-descriptions { display: none; }
      @media print {
        body { padding: 10mm; }
        @page { margin: 10mm; }
      }
    </style></head><body>${printHTML}</body></html>`)
  printWin.document.close()
  printWin.focus()
  setTimeout(() => { printWin.print(); printWin.close(); }, 300)
}

const exportExcel = () => {
  const tables = getReportTables()
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

watch(currentBook, (val) => { if (val) { loadReport(); if (activeTab.value === 'charts') loadChartData() } })
onMounted(() => {
  if (currentBook.value) loadReport()
  window.addEventListener('resize', handleChartResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', handleChartResize)
  chartTrend?.dispose(); chartPie?.dispose(); chartProfit?.dispose()
})
const handleChartResize = () => {
  chartTrend?.resize(); chartPie?.resize(); chartProfit?.resize()
}
// Custom Reports
const crList = ref([])
const crRunPeriod = ref(new Date().toISOString().slice(0, 7))
const crResult = ref(null)
const showCrEdit = ref(false)
const crForm = ref({ name: '', rows: [{ label: '', formula: '', level: 1, bold: false }] })
const formulaExamples = [
  { formula: "JE('code','dir')", desc: '本期发生额（借方或贷方发生额）', example: "JE('6602','借') → 管理费用本期借方发生" },
  { formula: "QM('code','dir')", desc: '期末余额', example: "QM('1002','借') → 银行存款期末余额" },
  { formula: "QC('code','dir')", desc: '期初余额', example: "QC('1001','借') → 库存现金期初余额" },
  { formula: "JL('code','dir')", desc: '本年累计发生额', example: "JL('5001','贷') → 主营业务本年累计" }
]

const loadCrList = async () => {
  if (!currentBook.value) return
  const { data } = await reportApi.templates.list(currentBook.value)
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
      await reportApi.templates.update(currentBook.value, crForm.value.id, payload)
    } else {
      await reportApi.templates.create(currentBook.value, payload)
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
  await reportApi.templates.delete(currentBook.value, tpl.id)
  ElMessage.success('已删除')
  loadCrList()
  crResult.value = null
}

</script>

<style>
/* 一级科目颜色标识（全局兜底，加 .el-table 提升权重覆盖 E+ 全量 CSS） */
.el-table tr.row-asset td,
.el-table tr.row-asset .el-table__cell { background-color: #d9ecff !important; }
.el-table tr.row-liability td,
.el-table tr.row-liability .el-table__cell { background-color: #fce4d6 !important; }
.el-table tr.row-equity td,
.el-table tr.row-equity .el-table__cell { background-color: #d9f7be !important; }
.el-table tr.row-cost td,
.el-table tr.row-cost .el-table__cell { background-color: #efdbff !important; }
.el-table tr.row-expense td,
.el-table tr.row-expense .el-table__cell { background-color: #ffd6d6 !important; }
</style>
<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.report-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.report-stack { display: flex; flex-direction: column; gap: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }

.chart-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.chart-label { font-size: 13px; color: #606266; font-weight: 500; margin-bottom: 8px; }
.chart-box { height: 280px; }
.chart-box-tall { height: 280px; }
@media (max-width: 768px) { .chart-grid { grid-template-columns: 1fr; } }

/* 一级科目颜色标识 */
:deep(.row-asset) td,
:deep(.row-asset .el-table__cell) { background-color: #d9ecff !important; }

:deep(.row-liability) td,
:deep(.row-liability .el-table__cell) { background-color: #fce4d6 !important; }

:deep(.row-equity) td,
:deep(.row-equity .el-table__cell) { background-color: #d9f7be !important; }

:deep(.row-cost) td,
:deep(.row-cost .el-table__cell) { background-color: #efdbff !important; }

:deep(.row-expense) td,
:deep(.row-expense .el-table__cell) { background-color: #ffd6d6 !important; }
</style>
