<template>
  <div class="inventory">
    <div class="page-header">
      <h2>进销存管理</h2>
    </div>

    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <!-- 商品管理 -->
      <el-tab-pane label="商品管理" name="goods">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-input v-model="goodsFilter.keyword" placeholder="搜索编码/名称" style="width: 200px" clearable @keyup.enter="loadGoods" />
          <el-button type="primary" @click="loadGoods">查询</el-button>
          <el-button type="success" @click="openGoodsDialog()"><el-icon><Plus /></el-icon>新增商品</el-button>
        </div>
        <el-table :data="goodsList" border size="small" style="width: 100%">
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="category" label="分类" width="100" />
          <el-table-column prop="unit" label="单位" width="60" />
          <el-table-column prop="ref_price" label="参考价" width="90" />
          <el-table-column prop="min_stock" label="最低库存" width="90" />
          <el-table-column prop="is_active" label="状态" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '启用' : '停' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="openGoodsDialog(row)">编辑</el-button>
              <el-button size="small" type="danger" link @click="deleteGoods(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 采购管理 -->
      <el-tab-pane label="采购管理" name="purchases">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-input v-model="purchaseFilter.keyword" placeholder="单号/供应商" style="width: 200px" clearable @keyup.enter="loadPurchases" />
          <el-select v-model="purchaseFilter.status" placeholder="状态" clearable style="width: 120px" @change="loadPurchases">
            <el-option label="草稿" value="draft" />
            <el-option label="已过账" value="posted" />
            <el-option label="已作废" value="voided" />
          </el-select>
          <el-button type="primary" @click="loadPurchases">查询</el-button>
          <el-button type="success" @click="openPurchaseDialog()"><el-icon><Plus /></el-icon>新增采购单</el-button>
        </div>
        <el-table :data="purchaseList" border size="small" style="width: 100%">
          <el-table-column prop="order_no" label="单号" width="160" />
          <el-table-column prop="date" label="日期" width="110" />
          <el-table-column prop="total_amount" label="金额" width="100" />
          <el-table-column prop="payment_term" label="付款方式" width="90">
            <template #default="{ row }">{{ row.payment_term === 'cash' ? '现结' : '赊账' }}</template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'posted' ? 'success' : row.status === 'voided' ? 'info' : 'warning'" size="small">
                {{ row.status === 'posted' ? '已过账' : row.status === 'voided' ? '已作废' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="memo" label="备注" min-width="120" show-overflow-tooltip />
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'draft'" size="small" type="success" link @click="postPurchase(row)">过账</el-button>
              <el-button v-if="row.status === 'draft'" size="small" type="info" link @click="voidPurchase(row)">作废</el-button>
              <el-button size="small" type="primary" link @click="viewPurchase(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 销售管理 -->
      <el-tab-pane label="销售管理" name="sales">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-input v-model="salesFilter.keyword" placeholder="单号/客户" style="width: 200px" clearable @keyup.enter="loadSalesList" />
          <el-select v-model="salesFilter.status" placeholder="状态" clearable style="width: 120px" @change="loadSalesList">
            <el-option label="草稿" value="draft" />
            <el-option label="已过账" value="posted" />
            <el-option label="已作废" value="voided" />
          </el-select>
          <el-button type="primary" @click="loadSalesList">查询</el-button>
          <el-button type="success" @click="openSalesDialog()"><el-icon><Plus /></el-icon>新增销售单</el-button>
        </div>
        <el-table :data="salesList" border size="small" style="width: 100%">
          <el-table-column prop="order_no" label="单号" width="160" />
          <el-table-column prop="date" label="日期" width="110" />
          <el-table-column prop="total_amount" label="金额" width="100" />
          <el-table-column prop="cost_amount" label="成本" width="100" />
          <el-table-column prop="payment_term" label="付款方式" width="90">
            <template #default="{ row }">{{ row.payment_term === 'cash' ? '现结' : '赊账' }}</template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'posted' ? 'success' : row.status === 'voided' ? 'info' : 'warning'" size="small">
                {{ row.status === 'posted' ? '已过账' : row.status === 'voided' ? '已作废' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="memo" label="备注" min-width="120" show-overflow-tooltip />
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'draft'" size="small" type="success" link @click="postSales(row)">过账</el-button>
              <el-button v-if="row.status === 'draft'" size="small" type="info" link @click="voidSales(row)">作废</el-button>
              <el-button size="small" type="primary" link @click="viewSales(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 收付款 -->
      <el-tab-pane label="收付款" name="payments">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-select v-model="paymentFilter.type" placeholder="类型" clearable style="width: 120px" @change="loadPayments">
            <el-option label="收款" value="receipt" />
            <el-option label="付款" value="payment" />
          </el-select>
          <el-select v-model="paymentFilter.status" placeholder="状态" clearable style="width: 120px" @change="loadPayments">
            <el-option label="草稿" value="draft" />
            <el-option label="已过账" value="posted" />
            <el-option label="已作废" value="voided" />
          </el-select>
          <el-button type="primary" @click="loadPayments">查询</el-button>
          <el-button type="success" @click="openPaymentDialog()"><el-icon><Plus /></el-icon>新增收付款</el-button>
        </div>
        <el-table :data="paymentList" border size="small" style="width: 100%">
          <el-table-column prop="record_no" label="单号" width="160" />
          <el-table-column prop="type" label="类型" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.type === 'receipt' ? 'success' : 'warning'" size="small">{{ row.type === 'receipt' ? '收款' : '付款' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="date" label="日期" width="110" />
          <el-table-column prop="amount" label="金额" width="100" />
          <el-table-column prop="method" label="方式" width="80">
            <template #default="{ row }">{{ row.method === 'cash' ? '现金' : '银行' }}</template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'posted' ? 'success' : row.status === 'voided' ? 'info' : 'warning'" size="small">
                {{ row.status === 'posted' ? '已过账' : row.status === 'voided' ? '已作废' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'draft'" size="small" type="success" link @click="postPayment(row)">过账</el-button>
              <el-button v-if="row.status === 'draft'" size="small" type="info" link @click="voidPayment(row)">作废</el-button>
              <el-button size="small" type="primary" link @click="viewPayment(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 库存报表 -->
      <el-tab-pane label="库存报表" name="stock">
        <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap">
          <el-input v-model="stockFilter.keyword" placeholder="商品编码/名称" style="width: 200px" clearable @keyup.enter="loadStockSummary" />
          <el-button type="primary" @click="loadStockSummary">查询</el-button>
        </div>
        <el-table :data="stockSummary" border size="small" style="width: 100%">
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="unit" label="单位" width="60" />
          <el-table-column prop="quantity" label="库存数量" width="100" />
          <el-table-column prop="unit_cost" label="单位成本" width="100" />
          <el-table-column prop="total_cost" label="库存金额" width="110" />
          <el-table-column prop="warehouse_name" label="仓库" width="100" />
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 商品编辑对话框 -->
    <el-dialog v-model="showGoodsDialog" :title="editingGoods ? '编辑商品' : '新增商品'" :width="isMobile ? '95%' : '550px'">
      <el-form :model="goodsForm" label-width="100px">
        <el-form-item label="编码" required><el-input v-model="goodsForm.code" placeholder="如 G001" /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="goodsForm.name" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="goodsForm.category" /></el-form-item>
        <el-form-item label="单位" required><el-input v-model="goodsForm.unit" placeholder="如：个、箱、kg" /></el-form-item>
        <el-form-item label="条码"><el-input v-model="goodsForm.barcode" /></el-form-item>
        <el-form-item label="参考价"><el-input-number v-model="goodsForm.ref_price" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="最低库存"><el-input-number v-model="goodsForm.min_stock" :min="0" :precision="2" /></el-form-item>
        <el-form-item label="计价方式">
          <el-select v-model="goodsForm.cost_method" style="width: 100%">
            <el-option label="移动加权平均" value="weighted_avg" />
            <el-option label="先进先出" value="fifo" />
            <el-option label="个别计价" value="specific" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGoodsDialog = false">取消</el-button>
        <el-button type="primary" @click="saveGoods">保存</el-button>
      </template>
    </el-dialog>

    <!-- 采购单对话框 -->
    <el-dialog v-model="showPurchaseDialog" :title="editingPurchase ? '编辑采购单' : '新增采购单'" :width="isMobile ? '95%' : '700px'" top="5vh">
      <el-form :model="purchaseForm" label-width="80px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="日期" required><el-date-picker v-model="purchaseForm.date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="供应商" required>
            <el-select v-model="purchaseForm.supplier_id" filterable placeholder="选择供应商" style="width:100%">
              <el-option v-for="s in suppliers" :key="s.id" :label="s.name" :value="s.id" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="仓库" required>
            <el-select v-model="purchaseForm.warehouse_id" filterable placeholder="选择仓库" style="width:100%">
              <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
            </el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="付款方式">
            <el-select v-model="purchaseForm.payment_term" style="width:100%">
              <el-option label="现结" value="cash" />
              <el-option label="赊账" value="credit" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="purchaseForm.memo" /></el-form-item>
        <el-divider>明细</el-divider>
        <div v-for="(item, i) in purchaseForm.items" :key="i" style="display:flex;gap:8px;margin-bottom:8px;align-items:center">
          <el-select v-model="item.goods_id" filterable placeholder="商品" style="flex:1">
            <el-option v-for="g in goodsList" :key="g.id" :label="g.code + ' ' + g.name" :value="g.id" />
          </el-select>
          <el-input-number v-model="item.quantity" :min="0" :precision="2" style="width:100px" placeholder="数量" />
          <el-input-number v-model="item.unit_price" :min="0" :precision="2" style="width:100px" placeholder="单价" />
          <span style="width:80px;text-align:right">{{ (item.quantity * item.unit_price).toFixed(2) }}</span>
          <el-button size="small" type="danger" link @click="purchaseForm.items.splice(i,1)" :disabled="purchaseForm.items.length <= 1"><el-icon><Delete /></el-icon></el-button>
        </div>
        <el-button size="small" @click="purchaseForm.items.push({ goods_id: null, quantity: 1, unit_price: 0 })"><el-icon><Plus /></el-icon>添加明细</el-button>
        <div style="text-align:right;margin-top:8px;font-weight:bold">合计: {{ purchaseFormTotal }}</div>
      </el-form>
      <template #footer>
        <el-button @click="showPurchaseDialog = false">取消</el-button>
        <el-button type="primary" @click="savePurchase">保存</el-button>
      </template>
    </el-dialog>

    <!-- 销售单对话框 -->
    <el-dialog v-model="showSalesDialog" :title="editingSales ? '编辑销售单' : '新增销售单'" :width="isMobile ? '95%' : '700px'" top="5vh">
      <el-form :model="salesForm" label-width="80px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="日期" required><el-date-picker v-model="salesForm.date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="客户" required>
            <el-select v-model="salesForm.customer_id" filterable placeholder="选择客户" style="width:100%">
              <el-option v-for="c in customers" :key="c.id" :label="c.name" :value="c.id" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="仓库" required>
            <el-select v-model="salesForm.warehouse_id" filterable placeholder="选择仓库" style="width:100%">
              <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
            </el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="付款方式">
            <el-select v-model="salesForm.payment_term" style="width:100%">
              <el-option label="现结" value="cash" />
              <el-option label="赊账" value="credit" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="salesForm.memo" /></el-form-item>
        <el-divider>明细</el-divider>
        <div v-for="(item, i) in salesForm.items" :key="i" style="display:flex;gap:8px;margin-bottom:8px;align-items:center">
          <el-select v-model="item.goods_id" filterable placeholder="商品" style="flex:1">
            <el-option v-for="g in goodsList" :key="g.id" :label="g.code + ' ' + g.name" :value="g.id" />
          </el-select>
          <el-input-number v-model="item.quantity" :min="0" :precision="2" style="width:100px" placeholder="数量" />
          <el-input-number v-model="item.unit_price" :min="0" :precision="2" style="width:100px" placeholder="单价" />
          <span style="width:80px;text-align:right">{{ (item.quantity * item.unit_price).toFixed(2) }}</span>
          <el-button size="small" type="danger" link @click="salesForm.items.splice(i,1)" :disabled="salesForm.items.length <= 1"><el-icon><Delete /></el-icon></el-button>
        </div>
        <el-button size="small" @click="salesForm.items.push({ goods_id: null, quantity: 1, unit_price: 0 })"><el-icon><Plus /></el-icon>添加明细</el-button>
        <div style="text-align:right;margin-top:8px;font-weight:bold">合计: {{ salesFormTotal }}</div>
      </el-form>
      <template #footer>
        <el-button @click="showSalesDialog = false">取消</el-button>
        <el-button type="primary" @click="saveSales">保存</el-button>
      </template>
    </el-dialog>

    <!-- 收付款对话框 -->
    <el-dialog v-model="showPaymentDialog" :title="editingPayment ? '编辑收付款' : '新增收付款'" :width="isMobile ? '95%' : '550px'">
      <el-form :model="paymentForm" label-width="80px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="日期" required><el-date-picker v-model="paymentForm.date" type="date" value-format="YYYY-MM-DD" style="width:100%" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="类型" required>
            <el-select v-model="paymentForm.type" style="width:100%">
              <el-option label="收款" value="receipt" />
              <el-option label="付款" value="payment" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="往来单位" required>
            <el-select v-model="paymentForm.counterparty_id" filterable placeholder="选择" style="width:100%">
              <el-option v-for="c in counterpartyList" :key="c.id" :label="c.name" :value="c.id" />
            </el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="金额" required><el-input-number v-model="paymentForm.amount" :min="0" :precision="2" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="方式">
            <el-select v-model="paymentForm.method" style="width:100%">
              <el-option label="银行" value="bank" />
              <el-option label="现金" value="cash" />
            </el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="往来类型">
            <el-select v-model="paymentForm.counterparty_type" style="width:100%">
              <el-option label="客户" value="customer" />
              <el-option label="供应商" value="supplier" />
            </el-select>
          </el-form-item></el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="paymentForm.memo" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPaymentDialog = false">取消</el-button>
        <el-button type="primary" @click="savePayment">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { inventoryApi, auxApi } from '../api'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'

const { currentBookId: currentBook } = useBookStore()
const { isMobile } = useMobile()
const activeTab = ref('goods')

// ========== Goods ==========
const goodsList = ref([])
const goodsFilter = ref({ keyword: '' })
const showGoodsDialog = ref(false)
const editingGoods = ref(null)
const goodsForm = ref({ code: '', name: '', category: '', unit: '', barcode: '', ref_price: 0, min_stock: 0, cost_method: 'weighted_avg' })

const loadGoods = async () => {
  if (!currentBook.value) return
  const { data } = await inventoryApi.listGoods(currentBook.value, goodsFilter.value)
  goodsList.value = data.data || []
}
const openGoodsDialog = (row) => {
  editingGoods.value = row || null
  if (row) {
    goodsForm.value = { ...row }
  } else {
    goodsForm.value = { code: '', name: '', category: '', unit: '', barcode: '', ref_price: 0, min_stock: 0, cost_method: 'weighted_avg' }
  }
  showGoodsDialog.value = true
}
const saveGoods = async () => {
  if (!goodsForm.value.code || !goodsForm.value.name || !goodsForm.value.unit) {
    ElMessage.warning('请填写必填项'); return
  }
  try {
    if (editingGoods.value) {
      await inventoryApi.updateGoods(currentBook.value, editingGoods.value.id, goodsForm.value)
    } else {
      await inventoryApi.createGoods(currentBook.value, goodsForm.value)
    }
    ElMessage.success('保存成功')
    showGoodsDialog.value = false
    loadGoods()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}
const deleteGoods = async (row) => {
  await ElMessageBox.confirm(`确定删除"${row.name}"？`, '确认')
  await inventoryApi.deleteGoods(currentBook.value, row.id)
  ElMessage.success('已删除')
  loadGoods()
}

// ========== Purchases ==========
const purchaseList = ref([])
const purchaseFilter = ref({ keyword: '', status: '' })
const showPurchaseDialog = ref(false)
const editingPurchase = ref(null)
const purchaseForm = ref({ date: '', supplier_id: null, warehouse_id: null, payment_term: 'credit', memo: '', items: [{ goods_id: null, quantity: 1, unit_price: 0 }] })

const purchaseFormTotal = computed(() => purchaseForm.value.items.reduce((s, i) => s + (i.quantity * i.unit_price), 0).toFixed(2))

const loadPurchases = async () => {
  if (!currentBook.value) return
  const { data } = await inventoryApi.listPurchases(currentBook.value, purchaseFilter.value)
  purchaseList.value = data.data || []
}
const openPurchaseDialog = () => {
  editingPurchase.value = null
  purchaseForm.value = { date: new Date().toISOString().slice(0, 10), supplier_id: null, warehouse_id: null, payment_term: 'credit', memo: '', items: [{ goods_id: null, quantity: 1, unit_price: 0 }] }
  showPurchaseDialog.value = true
}
const savePurchase = async () => {
  if (!purchaseForm.value.date || !purchaseForm.value.supplier_id || !purchaseForm.value.warehouse_id) {
    ElMessage.warning('请填写必填项'); return
  }
  const validItems = purchaseForm.value.items.filter(i => i.goods_id && i.quantity > 0)
  if (validItems.length === 0) { ElMessage.warning('请至少添加一条明细'); return }
  try {
    await inventoryApi.createPurchase(currentBook.value, { ...purchaseForm.value, items: validItems })
    ElMessage.success('保存成功')
    showPurchaseDialog.value = false
    loadPurchases()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}
const postPurchase = async (row) => {
  await ElMessageBox.confirm('确定过账？将生成会计凭证。', '确认')
  await inventoryApi.postPurchase(currentBook.value, row.id)
  ElMessage.success('已过账')
  loadPurchases()
}
const voidPurchase = async (row) => {
  await ElMessageBox.confirm('确定作废此单据？', '确认')
  await inventoryApi.voidPurchase(currentBook.value, row.id)
  ElMessage.success('已作废')
  loadPurchases()
}
const viewPurchase = (row) => {
  ElMessage.info(`查看采购单 ${row.order_no}`)
}

// ========== Sales ==========
const salesList = ref([])
const salesFilter = ref({ keyword: '', status: '' })
const showSalesDialog = ref(false)
const editingSales = ref(null)
const salesForm = ref({ date: '', customer_id: null, warehouse_id: null, payment_term: 'credit', memo: '', items: [{ goods_id: null, quantity: 1, unit_price: 0 }] })

const salesFormTotal = computed(() => salesForm.value.items.reduce((s, i) => s + (i.quantity * i.unit_price), 0).toFixed(2))

const loadSalesList = async () => {
  if (!currentBook.value) return
  const { data } = await inventoryApi.listSales(currentBook.value, salesFilter.value)
  salesList.value = data.data || []
}
const openSalesDialog = () => {
  editingSales.value = null
  salesForm.value = { date: new Date().toISOString().slice(0, 10), customer_id: null, warehouse_id: null, payment_term: 'credit', memo: '', items: [{ goods_id: null, quantity: 1, unit_price: 0 }] }
  showSalesDialog.value = true
}
const saveSales = async () => {
  if (!salesForm.value.date || !salesForm.value.customer_id || !salesForm.value.warehouse_id) {
    ElMessage.warning('请填写必填项'); return
  }
  const validItems = salesForm.value.items.filter(i => i.goods_id && i.quantity > 0)
  if (validItems.length === 0) { ElMessage.warning('请至少添加一条明细'); return }
  try {
    await inventoryApi.createSales(currentBook.value, { ...salesForm.value, items: validItems })
    ElMessage.success('保存成功')
    showSalesDialog.value = false
    loadSalesList()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}
const postSales = async (row) => {
  await ElMessageBox.confirm('确定过账？将生成两张会计凭证（收入+成本）。', '确认')
  await inventoryApi.postSales(currentBook.value, row.id)
  ElMessage.success('已过账')
  loadSalesList()
}
const voidSales = async (row) => {
  await ElMessageBox.confirm('确定作废此单据？', '确认')
  await inventoryApi.voidSales(currentBook.value, row.id)
  ElMessage.success('已作废')
  loadSalesList()
}
const viewSales = (row) => {
  ElMessage.info(`查看销售单 ${row.order_no}`)
}

// ========== Payments ==========
const paymentList = ref([])
const paymentFilter = ref({ type: '', status: '' })
const showPaymentDialog = ref(false)
const editingPayment = ref(null)
const paymentForm = ref({ date: '', type: 'receipt', counterparty_type: 'customer', counterparty_id: null, amount: 0, method: 'bank', memo: '' })

const loadPayments = async () => {
  if (!currentBook.value) return
  const { data } = await inventoryApi.listPayments(currentBook.value, paymentFilter.value)
  paymentList.value = data.data || []
}
const openPaymentDialog = () => {
  editingPayment.value = null
  paymentForm.value = { date: new Date().toISOString().slice(0, 10), type: 'receipt', counterparty_type: 'customer', counterparty_id: null, amount: 0, method: 'bank', memo: '' }
  showPaymentDialog.value = true
}
const savePayment = async () => {
  if (!paymentForm.value.date || !paymentForm.value.counterparty_id || !paymentForm.value.amount) {
    ElMessage.warning('请填写必填项'); return
  }
  try {
    await inventoryApi.createPayment(currentBook.value, paymentForm.value)
    ElMessage.success('保存成功')
    showPaymentDialog.value = false
    loadPayments()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}
const postPayment = async (row) => {
  await ElMessageBox.confirm('确定过账？将生成会计凭证。', '确认')
  await inventoryApi.postPayment(currentBook.value, row.id)
  ElMessage.success('已过账')
  loadPayments()
}
const voidPayment = async (row) => {
  await ElMessageBox.confirm('确定作废此单据？', '确认')
  await inventoryApi.voidPayment(currentBook.value, row.id)
  ElMessage.success('已作废')
  loadPayments()
}
const viewPayment = (row) => {
  ElMessage.info(`查看收付款 ${row.record_no}`)
}

// ========== Stock ==========
const stockSummary = ref([])
const stockFilter = ref({ keyword: '' })

const loadStockSummary = async () => {
  if (!currentBook.value) return
  const { data } = await inventoryApi.stockSummary(currentBook.value, stockFilter.value)
  stockSummary.value = data.data || []
}

// ========== Counterparty (for payment dialog) ==========
const suppliers = ref([])
const customers = ref([])
const warehouses = ref([])
const counterpartyList = computed(() => paymentForm.value.counterparty_type === 'customer' ? customers.value : suppliers.value)

const loadAux = async () => {
  if (!currentBook.value) return
  try {
    const { data: supData } = await auxApi.list(currentBook.value, 'supplier')
    suppliers.value = (supData.data || []).filter(a => a.is_active !== false).map(a => ({ id: a.id, name: a.name }))
  } catch {}
  try {
    const { data: custData } = await auxApi.list(currentBook.value, 'customer')
    customers.value = (custData.data || []).filter(a => a.is_active !== false).map(a => ({ id: a.id, name: a.name }))
  } catch {}
  try {
    const { data: whData } = await auxApi.list(currentBook.value, 'warehouse')
    warehouses.value = (whData.data || []).filter(a => a.is_active !== false).map(a => ({ id: a.id, name: a.name }))
  } catch {}
}

const onTabChange = (tab) => {
  if (tab === 'goods') loadGoods()
  else if (tab === 'purchases') loadPurchases()
  else if (tab === 'sales') loadSalesList()
  else if (tab === 'payments') loadPayments()
  else if (tab === 'stock') loadStockSummary()
}

onMounted(() => {
  if (currentBook.value) {
    loadAux()
    loadGoods()
  }
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
.page-header h2 { color: #303133; font-size: 18px; }
</style>
