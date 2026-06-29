<template>
  <div class="settings">
    <div class="page-header">
      <h2>系统设置</h2>
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
            <!-- Voucher Template Edit Dialog -->
    <el-dialog v-model="showVtplEdit" :title="editingVtpl ? '编辑模板' : '新增模板'" :width="isMobile ? '95%' : '600px'">
      <el-form :model="vtplForm" label-width="80px">
        <el-form-item label="模板名称" required><el-input v-model="vtplForm.name" placeholder="如：收货款" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="vtplForm.category" placeholder="如：收入、费用" /></el-form-item>
        <el-form-item label="分录">
          <div v-for="(item, i) in vtplForm.items" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <el-select v-model="item.account_id" filterable placeholder="科目" style="flex: 1">
              <el-option v-for="a in accounts" :key="a.id" :label="a.code + ' ' + a.name" :value="a.id" :disabled="!a.is_leaf" />
            </el-select>
            <el-input v-model="item.memo" placeholder="摘要" style="width: 130px" />
            <el-button size="small" type="danger" link @click="vtplForm.items.splice(i, 1)" :disabled="vtplForm.items.length <= 1"><el-icon><Delete /></el-icon></el-button>
          </div>
          <el-button size="small" @click="vtplForm.items.push({ account_id: null, memo: '' })"><el-icon><Plus /></el-icon>添加</el-button>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="showVtplEdit = false">取消</el-button><el-button type="primary" @click="saveVtpl">保存</el-button></template>
    </el-dialog>

    
</el-tabs>

        <div class="toolbar">
          <el-button type="primary" size="small" @click="openAdd">
            <el-icon><Plus /></el-icon>新增{{ auxLabel }}
          </el-button>
          <el-button size="small" @click="exportData">
            <el-icon><Download /></el-icon>导出
          </el-button>
          <el-upload
            :action="importUrl"
            :headers="uploadHeaders"
            :show-file-list="false"
            :on-success="onImportSuccess"
            :on-error="onImportError"
            accept=".csv"
            style="display: inline-block; margin-left: 8px"
          >
            <el-button size="small"><el-icon><Upload /></el-icon>导入CSV</el-button>
          </el-upload>
          <el-button size="small" @click="downloadTemplate" style="margin-left: 8px">
            <el-icon><Download /></el-icon>下载模板
          </el-button>
          <el-button size="small" type="danger" :disabled="selectedItems.length === 0" @click="batchDelete" style="margin-left: 8px">
            删除选中({{ selectedItems.length }})
          </el-button>
        </div>

        <div class="table-wrapper">
          <el-table :data="auxItems" border size="small" @selection-change="onSelectionChange" :max-height="tableMaxHeight">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="code" label="编码" width="100" />
            <el-table-column prop="name" :label="auxType === 'employee' ? '姓名' : '名称'" min-width="120" />
            <!-- Dynamic columns based on type -->
            <template v-if="auxType === 'customer' || auxType === 'supplier'">
              <el-table-column label="联系人" width="100">
                <template #default="{ row }">{{ getExtra(row, 'contact') }}</template>
              </el-table-column>
              <el-table-column label="电话" width="120">
                <template #default="{ row }">{{ getExtra(row, 'phone') }}</template>
              </el-table-column>
              <el-table-column label="地址" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'department'">
              <el-table-column label="上级部门" width="120">
                <template #default="{ row }">{{ getExtra(row, 'parent') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'project'">
              <el-table-column label="状态" width="80">
                <template #default="{ row }">{{ getExtra(row, 'status') }}</template>
              </el-table-column>
              <el-table-column label="开始日期" width="100">
                <template #default="{ row }">{{ getExtra(row, 'start_date') }}</template>
              </el-table-column>
              <el-table-column label="结束日期" width="100">
                <template #default="{ row }">{{ getExtra(row, 'end_date') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'employee'">
              <el-table-column label="部门" width="100">
                <template #default="{ row }">{{ getExtra(row, 'department') }}</template>
              </el-table-column>
              <el-table-column label="电话" width="120">
                <template #default="{ row }">{{ getExtra(row, 'phone') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'warehouse'">
              <el-table-column label="地址" min-width="180">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <template v-if="auxType === 'bank_account'">
              <el-table-column label="银行账号" width="150">
                <template #default="{ row }">{{ getExtra(row, 'account_number') }}</template>
              </el-table-column>
              <el-table-column label="开户行" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'bank_name') }}</template>
              </el-table-column>
              <el-table-column label="户名" width="120">
                <template #default="{ row }">{{ getExtra(row, 'account_holder') }}</template>
              </el-table-column>
              <el-table-column label="地址" min-width="150">
                <template #default="{ row }">{{ getExtra(row, 'address') }}</template>
              </el-table-column>
            </template>
            <el-table-column label="备注" min-width="100">
              <template #default="{ row }">{{ getExtra(row, 'memo') }}</template>
            </el-table-column>
            <el-table-column prop="is_active" label="状态" width="70" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '启用' : '停' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="editAux(row)">编辑</el-button>
                <el-button size="small" type="danger" link @click="deleteAux(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="科目模板" name="templates">
        <el-card shadow="never" v-if="v2Manifest" style="margin-bottom: 12px">
          <template #header><strong>v2 行业模板体系</strong></template>
          <el-descriptions :column="isMobile ? 1 : 3" border size="small">
            <el-descriptions-item label="版本">{{ v2Manifest.version }}</el-descriptions-item>
            <el-descriptions-item label="会计准则">{{ Object.values(v2Manifest.standards || {}).map(s => s.name).join('、') }}</el-descriptions-item>
            <el-descriptions-item label="模板总数">{{ v2Manifest.templates?.length || 0 }} 个</el-descriptions-item>
          </el-descriptions>
          <el-table :data="v2Manifest.templates || []" border size="small" style="margin-top: 12px" max-height="400">
            <el-table-column prop="id" label="模板ID" min-width="220" />
            <el-table-column label="准则" width="120">
              <template #default="{ row }">{{ v2Manifest.standards?.[row.standard]?.name || row.standard }}</template>
            </el-table-column>
            <el-table-column label="行业" width="100">
              <template #default="{ row }">{{ v2Manifest.industries?.[row.industry]?.name || row.industry }}</template>
            </el-table-column>
            <el-table-column label="纳税人" width="100">
              <template #default="{ row }">{{ v2Manifest.taxpayer_types?.[row.taxpayer]?.name || row.taxpayer }}</template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>旧版模板（兼容）</span>
              <el-button size="small" type="primary" @click="syncAllTemplates" :loading="syncing">
                <el-icon><Refresh /></el-icon>同步模板到账套
              </el-button>
            </div>
          </template>
          <el-table :data="templateVersions" border size="small">
            <el-table-column prop="id" label="模板ID" width="150" />
            <el-table-column prop="name" label="模板名称" min-width="150" />
            <el-table-column prop="version" label="版本" width="120" />
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="凭证模板" name="vtpl">
        <div v-if="currentBook">
          <div style="margin-bottom: 12px">
            <el-button type="primary" size="small" @click="openVtplAdd">
              <el-icon><Plus /></el-icon>新增模板
            </el-button>
          </div>
          <el-table :data="vtplList" border size="small">
            <el-table-column prop="name" label="模板名称" min-width="150" />
            <el-table-column prop="category" label="分类" width="120" />
            <el-table-column label="分录" min-width="250">
              <template #default="{ row }">
                <span v-for="(item, i) in parseVtplItems(row.items)" :key="i" style="margin-right: 10px; font-size: 13px; color: #606266">
                  {{ item.account_code }} {{ item.account_name }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="editVtpl(row)">编辑</el-button>
                <el-button size="small" type="danger" link @click="deleteVtpl(row)">删除</el-button>
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

      <el-tab-pane label="菜单排序" name="menu">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>菜单排序（拖拽调整顺序，可控制显示/隐藏）</span>
              <el-button size="small" @click="resetMenu">恢复默认</el-button>
            </div>
          </template>
          <div class="menu-sort-list">
            <div v-for="(item, index) in menuConfig" :key="item.index" class="menu-sort-item">
              <div class="menu-sort-left">
                <el-icon class="sort-handle"><Rank /></el-icon>
                <el-icon><component :is="iconMap[item.icon] || HomeFilled" /></el-icon>
                <span>{{ item.label }}</span>
              </div>
              <div class="menu-sort-right">
                <el-button size="small" :disabled="index === 0" @click="moveMenu(index, -1)">
                  <el-icon><Top /></el-icon>
                </el-button>
                <el-button size="small" :disabled="index === menuConfig.length - 1" @click="moveMenu(index, 1)">
                  <el-icon><Bottom /></el-icon>
                </el-button>
                <el-switch v-model="item.visible" @change="saveMenu" style="margin-left: 12px" />
              </div>
            </div>
          </div>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="用户管理" name="users">
        <div style="margin-bottom: 12px">
          <el-button type="primary" size="small" @click="openUserAdd"><el-icon><Plus /></el-icon>新增用户</el-button>
        </div>
        <div class="table-wrapper">
          <el-table :data="allUsers" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="username" label="用户名" width="150" />
            <el-table-column prop="real_name" label="姓名" width="120" />
            <el-table-column prop="role" label="角色" width="100">
              <template #default="{ row }">
                <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">{{ row.role === 'admin' ? '管理员' : '普通用户' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180" />
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="editUser(row)">编辑</el-button>
                <el-button size="small" type="warning" link @click="resetUserPwd(row)">重置密码</el-button>
                <el-button size="small" type="danger" link @click="deleteUser(row)" :disabled="row.role === 'admin'">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="账套权限" name="permissions">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center">
          <span>选择账套：</span>
          <el-select v-model="permBookId" placeholder="请选择账套" @change="loadBookUsers" style="width: 250px">
            <el-option v-for="b in books" :key="b.id" :label="b.name" :value="b.id" />
          </el-select>
          <el-button type="primary" size="small" @click="openPermAdd" :disabled="!permBookId"><el-icon><Plus /></el-icon>添加用户</el-button>
        </div>
        <div class="table-wrapper">
          <el-table :data="bookUsers" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="username" label="用户名" width="150" />
            <el-table-column prop="real_name" label="姓名" width="120" />
            <el-table-column prop="role" label="权限" width="120">
              <template #default="{ row }">
                <el-select v-model="row.role" size="small" @change="updatePerm(row)" style="width: 100px">
                  <el-option label="完全控制" value="full" />
                  <el-option label="只读" value="readonly" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button size="small" type="danger" link @click="removePerm(row)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="操作日志" name="logs">
        <div style="margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap">
          <el-input v-model="logFilters.operator" placeholder="操作人" style="width: 120px" size="small" clearable />
          <el-input v-model="logFilters.module" placeholder="模块" style="width: 120px" size="small" clearable />
          <el-input v-model="logFilters.action" placeholder="操作" style="width: 120px" size="small" clearable />
          <el-button size="small" @click="loadLogs"><el-icon><Refresh /></el-icon>查询</el-button>
        </div>
        <div class="table-wrapper">
          <el-table :data="logs" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="created_at" label="时间" width="180" />
            <el-table-column prop="operator" label="操作人" width="120" />
            <el-table-column prop="module" label="模块" width="100" />
            <el-table-column prop="action" label="操作" width="100" />
            <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
            <el-table-column prop="ip" label="IP" width="130" />
          </el-table>
        </div>
        <div style="margin-top: 12px; display: flex; justify-content: flex-end">
          <el-pagination
            v-model:current-page="logPage"
            :page-size="50"
            :total="logTotal"
            layout="total, prev, pager, next"
            @current-change="loadLogs"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="备份恢复" name="backup">
        <div style="margin-bottom: 12px">
          <el-button type="primary" size="small" @click="createBackup" :disabled="backupLoading">
            <el-icon><Download /></el-icon>立即备份
          </el-button>
        </div>
        <div class="table-wrapper">
          <el-table :data="backups" border size="small" :max-height="tableMaxHeight">
            <el-table-column prop="name" label="备份文件" min-width="250" />
            <el-table-column label="大小" width="100">
              <template #default="{ row }">{{ (row.size / 1024).toFixed(1) }} KB</template>
            </el-table-column>
            <el-table-column label="时间" width="200">
              <template #default="{ row }">{{ new Date(row.time).toLocaleString('zh-CN') }}</template>
            </el-table-column>
            <el-table-column label="操作" width="250">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="downloadBackup(row.name)">
                  <el-icon><Download /></el-icon>下载
                </el-button>
                <el-button size="small" type="warning" link @click="restoreBackup(row.name)">
                  <el-icon><Upload /></el-icon>恢复
                </el-button>
                <el-button size="small" type="danger" link @click="deleteBackupFile(row.name)">
                  <el-icon><Delete /></el-icon>删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

        <!-- Voucher Template Edit Dialog -->
    <el-dialog v-model="showVtplEdit" :title="editingVtpl ? '编辑模板' : '新增模板'" :width="isMobile ? '95%' : '600px'">
      <el-form :model="vtplForm" label-width="80px">
        <el-form-item label="模板名称" required><el-input v-model="vtplForm.name" placeholder="如：收货款" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="vtplForm.category" placeholder="如：收入、费用" /></el-form-item>
        <el-form-item label="分录">
          <div v-for="(item, i) in vtplForm.items" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <el-select v-model="item.account_id" filterable placeholder="科目" style="flex: 1">
              <el-option v-for="a in accounts" :key="a.id" :label="a.code + ' ' + a.name" :value="a.id" :disabled="!a.is_leaf" />
            </el-select>
            <el-input v-model="item.memo" placeholder="摘要" style="width: 130px" />
            <el-button size="small" type="danger" link @click="vtplForm.items.splice(i, 1)" :disabled="vtplForm.items.length <= 1"><el-icon><Delete /></el-icon></el-button>
          </div>
          <el-button size="small" @click="vtplForm.items.push({ account_id: null, memo: '' })"><el-icon><Plus /></el-icon>添加</el-button>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="showVtplEdit = false">取消</el-button><el-button type="primary" @click="saveVtpl">保存</el-button></template>
    </el-dialog>

    
</el-tabs>

    <!-- Add/Edit dialog -->
    <el-dialog v-model="showEdit" :title="editingItem ? '编辑' : '新增' + auxLabel" :width="isMobile ? '95%' : '550px'">
      <el-form :model="editForm" label-width="80px" size="small">
        <el-form-item label="编码" required>
          <el-input v-model="editForm.code" placeholder="唯一编码" />
        </el-form-item>
        <el-form-item :label="auxType === 'employee' ? '姓名' : '名称'" required>
          <el-input v-model="editForm.name" :placeholder="auxType === 'employee' ? '姓名' : '名称'" />
        </el-form-item>

        <!-- Customer/Supplier extra fields -->
        <template v-if="auxType === 'customer' || auxType === 'supplier'">
          <el-form-item label="联系人">
            <el-input v-model="editForm.extra.contact" />
          </el-form-item>
          <el-form-item label="电话">
            <el-input v-model="editForm.extra.phone" />
          </el-form-item>
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <!-- Department -->
        <template v-if="auxType === 'department'">
          <el-form-item label="上级部门">
            <el-input v-model="editForm.extra.parent" />
          </el-form-item>
        </template>

        <!-- Project -->
        <template v-if="auxType === 'project'">
          <el-form-item label="状态">
            <el-select v-model="editForm.extra.status" style="width: 100%">
              <el-option label="进行中" value="进行中" />
              <el-option label="已完成" value="已完成" />
              <el-option label="已暂停" value="已暂停" />
            </el-select>
          </el-form-item>
          <el-form-item label="开始日期">
            <el-date-picker v-model="editForm.extra.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束日期">
            <el-date-picker v-model="editForm.extra.end_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          </el-form-item>
        </template>

        <!-- Employee -->
        <template v-if="auxType === 'employee'">
          <el-form-item label="部门">
            <el-input v-model="editForm.extra.department" />
          </el-form-item>
          <el-form-item label="电话">
            <el-input v-model="editForm.extra.phone" />
          </el-form-item>
        </template>

        <!-- Warehouse -->
        <template v-if="auxType === 'warehouse'">
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <!-- Bank Account -->
        <template v-if="auxType === 'bank_account'">
          <el-form-item label="银行账号">
            <el-input v-model="editForm.extra.account_number" />
          </el-form-item>
          <el-form-item label="开户行">
            <el-input v-model="editForm.extra.bank_name" />
          </el-form-item>
          <el-form-item label="户名">
            <el-input v-model="editForm.extra.account_holder" />
          </el-form-item>
          <el-form-item label="地址">
            <el-input v-model="editForm.extra.address" />
          </el-form-item>
        </template>

        <el-form-item label="备注">
          <el-input v-model="editForm.extra.memo" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">取消</el-button>
        <el-button type="primary" @click="saveItem">保存</el-button>
      </template>
    </el-dialog>

    <!-- User Edit Dialog -->
    <el-dialog v-model="showUserEdit" :title="editingUser ? '编辑用户' : '新增用户'" :width="isMobile ? '95%' : '450px'">
      <el-form :model="userForm" label-width="80px">
        <el-form-item label="用户名" required><el-input v-model="userForm.username" :disabled="!!editingUser" /></el-form-item>
        <el-form-item label="密码" :required="!editingUser">
          <el-input v-model="userForm.password" type="password" show-password :placeholder="editingUser ? '不修改请留空' : '至少6位'" />
        </el-form-item>
        <el-form-item label="姓名"><el-input v-model="userForm.real_name" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userForm.role">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showUserEdit = false">取消</el-button>
        <el-button type="primary" @click="saveUser">保存</el-button>
      </template>
    </el-dialog>

    <!-- Permission Add Dialog -->
    <el-dialog v-model="showPermAdd" title="添加用户权限" :width="isMobile ? '95%' : '450px'">
      <el-form :model="permForm" label-width="80px">
        <el-form-item label="用户" required>
          <el-select v-model="permForm.user_id" filterable placeholder="选择用户">
            <el-option v-for="u in allUsers" :key="u.id" :label="u.username + (u.real_name ? ' (' + u.real_name + ')' : '')" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="权限">
          <el-select v-model="permForm.role">
            <el-option label="完全控制" value="full" />
            <el-option label="只读" value="readonly" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPermAdd = false">取消</el-button>
        <el-button type="primary" @click="savePerm">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { auxApi, bookApi, templateApi, voucherTemplateApi, systemApi, bookUserApi, userApi } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting, SwitchButton, Coin, Top, Bottom, Rank, Delete, Plus, Download, Upload, Refresh } from '@element-plus/icons-vue'
import { useBookStore } from '../stores/book'
import { useMobile } from '../composables/useMobile'

const { isMobile } = useMobile()
const tableMaxHeight = computed(() => isMobile.value ? 'calc(100vh - 320px)' : 'calc(100vh - 350px)')

const { currentBookId: currentBook, books, setCurrentBook } = useBookStore()
const activeTab = ref('aux')
const auxType = ref('customer')
const auxItems = ref([])
const bookInfo = ref(null)
const selectedItems = ref([])

const showEdit = ref(false)
const editingItem = ref(null)
const editForm = ref({ code: '', name: '', extra: {} })

const auxLabel = computed(() => ({
  customer: '客户', supplier: '供应商', department: '部门',
  project: '项目', employee: '员工', warehouse: '仓库', bank_account: '银行账号'
}[auxType.value] || ''))

const importUrl = computed(() => `/api/books/${currentBook.value}/aux/${auxType.value}/import`)
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
}))

const defaultExtra = () => {
  switch (auxType.value) {
    case 'customer': case 'supplier':
      return { contact: '', phone: '', address: '', memo: '' }
    case 'department':
      return { parent: '' }
    case 'project':
      return { status: '进行中', start_date: '', end_date: '', memo: '' }
    case 'employee':
      return { department: '', phone: '', memo: '' }
    case 'warehouse':
      return { address: '', memo: '' }
    case 'bank_account':
      return { account_number: '', bank_name: '', account_holder: '', address: '', memo: '' }
    default:
      return {}
  }
}

const getExtra = (row, key) => {
  try {
    const extra = JSON.parse(row.extra || '{}')
    return extra[key] || ''
  } catch { return '' }
}

const loadAux = async () => {
  if (!currentBook.value) return
  const { data } = await auxApi.list(currentBook.value, auxType.value)
  auxItems.value = data.data || []
}

const loadBookInfo = async () => {
  if (!currentBook.value) return
  const { data } = await bookApi.get(currentBook.value)
  bookInfo.value = data.data
}

// Menu config
const iconMap = { HomeFilled, Notebook, Memo, Document, List, DataAnalysis, Setting }
const menuConfig = ref([])

const defaultMenu = [
  { index: '/', label: '工作台', icon: 'HomeFilled', visible: true },
  { index: '/books', label: '账套管理', icon: 'Notebook', visible: true },
  { index: '/accounts', label: '科目管理', icon: 'Memo', visible: true },
  { index: '/vouchers', label: '凭证管理', icon: 'Document', visible: true },
  { index: '/ledger', label: '账簿查询', icon: 'List', visible: true },
  { index: '/reports', label: '报表中心', icon: 'DataAnalysis', visible: true },
  { index: '/opening-balance', label: '期初余额', icon: 'Coin', visible: true },
          { index: '/closing', label: '期末处理', icon: 'SwitchButton', visible: true },
  { index: '/settings', label: '系统设置', icon: 'Setting', visible: true },
]

const loadMenuConfig = () => {
  try {
    const saved = localStorage.getItem('zhangyi_menu_config')
    if (saved) {
      const parsed = JSON.parse(saved)
      // Merge: use saved order, append new default items not in saved config
      const savedIndexes = new Set(parsed.map(p => p.index))
      const extras = defaultMenu.filter(d => !savedIndexes.has(d.index))
      menuConfig.value = [...parsed, ...extras].map(item => {
        const def = defaultMenu.find(d => d.index === item.index)
        return def ? { ...def, ...item } : item
      })
    } else {
      menuConfig.value = [...defaultMenu]
    }
  } catch {}
}

const saveMenu = () => {
  localStorage.setItem('zhangyi_menu_config', JSON.stringify(menuConfig.value))
  // Trigger App.vue to reload
  window.dispatchEvent(new Event('menu-config-changed'))
}

const moveMenu = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= menuConfig.value.length) return
  const arr = [...menuConfig.value]
  const temp = arr[index]
  arr[index] = arr[newIndex]
  arr[newIndex] = temp
  menuConfig.value = arr
  saveMenu()
}

const resetMenu = () => {
  localStorage.removeItem('zhangyi_menu_config')
  loadMenuConfig()
  window.dispatchEvent(new Event('menu-config-changed'))
  ElMessage.success('已恢复默认菜单')
}

// Template versions
const templateVersions = ref([])
const v2Manifest = ref(null)
const syncing = ref(false)

const loadTemplateVersions = async () => {
  try {
    const { data } = await templateApi.versions()
    templateVersions.value = data.data || []
  } catch (e) { console.error(e) }
  try {
    const { data } = await templateApi.manifest()
    v2Manifest.value = data
  } catch (e) { /* v2 not available */ }
}

const syncAllTemplates = async () => {
  if (!currentBook.value) return
  syncing.value = true
  try {
    const { data } = await bookApi.syncAllTemplates(currentBook.value)
    ElMessage.success(data.message || '同步成功')
  } catch (e) { ElMessage.error(e.response?.data?.error || '同步失败') }
  finally { syncing.value = false }
}

// Voucher Templates
const vtplList = ref([])
const showVtplEdit = ref(false)
const editingVtpl = ref(null)
const vtplForm = ref({ name: '', category: '', items: [{ account_id: null, memo: '' }] })

// Phase 3: User Management
const allUsers = ref([])
const showUserEdit = ref(false)
const editingUser = ref(null)
const userForm = ref({ username: '', password: '', real_name: '', role: 'user' })

// Phase 3: Book Permissions
const permBookId = ref(null)
const bookUsers = ref([])
const showPermAdd = ref(false)
const permForm = ref({ user_id: null, role: 'full' })

// Phase 3: Operation Logs
const logs = ref([])
const logTotal = ref(0)
const logPage = ref(1)
const logFilters = ref({ operator: '', module: '', action: '' })

// Phase 3: Backup & Restore
const backups = ref([])
const backupLoading = ref(false)

const loadVtplList = async () => {
  if (!currentBook.value) return
  const { data } = await voucherTemplateApi.list(currentBook.value)
  vtplList.value = data.data || []
}
const parseVtplItems = (s) => { try { return JSON.parse(s || '[]') } catch { return [] } }
const openVtplAdd = () => {
  editingVtpl.value = null
  vtplForm.value = { name: '', category: '', items: [{ account_id: null, memo: '' }] }
  showVtplEdit.value = true
}
const editVtpl = (row) => {
  editingVtpl.value = row
  const items = parseVtplItems(row.items)
  vtplForm.value = { name: row.name, category: row.category || '', items: items.length > 0 ? items : [{ account_id: null, memo: '' }] }
  showVtplEdit.value = true
}
const saveVtpl = async () => {
  if (!vtplForm.value.name) { ElMessage.warning('请输入模板名称'); return }
  const items = vtplForm.value.items.filter(i => i.account_id).map(i => {
    const acct = auxItems.value.find(a => a.id === i.account_id) || {}
    return { account_id: i.account_id, account_code: acct.code || i.account_code || '', account_name: acct.name || i.account_name || '', memo: i.memo || '' }
  })
  if (items.length === 0) { ElMessage.warning('请至少添加一条分录'); return }
  try {
    const payload = { name: vtplForm.value.name, category: vtplForm.value.category, items: JSON.stringify(items) }
    if (editingVtpl.value) {
      await voucherTemplateApi.update(currentBook.value, editingVtpl.value.id, payload)
    } else {
      await voucherTemplateApi.create(currentBook.value, payload)
    }
    ElMessage.success('保存成功')
    showVtplEdit.value = false
    loadVtplList()
  } catch (e) { ElMessage.error('保存失败') }
}
const deleteVtpl = async (row) => {
  await ElMessageBox.confirm(`确定删除"${row.name}"？`, '确认')
  await voucherTemplateApi.delete(currentBook.value, row.id)
  ElMessage.success('已删除')
  loadVtplList()
}

const onTabChange = () => {
  if (activeTab.value === 'aux') loadAux()
  else if (activeTab.value === 'book') loadBookInfo()
  else if (activeTab.value === 'menu') loadMenuConfig()
  else if (activeTab.value === 'templates') loadTemplateVersions()
  else if (activeTab.value === 'vtpl') loadVtplList()
  }

const onSelectionChange = (rows) => { selectedItems.value = rows }

const openAdd = () => {
  editingItem.value = null
  editForm.value = { code: '', name: '', extra: defaultExtra() }
  showEdit.value = true
}

const editAux = (row) => {
  editingItem.value = row
  let extra = {}
  try { extra = JSON.parse(row.extra || '{}') } catch {}
  editForm.value = { code: row.code, name: row.name, extra: { ...defaultExtra(), ...extra } }
  showEdit.value = true
}

const saveItem = async () => {
  try {
    const payload = {
      code: editForm.value.code,
      name: editForm.value.name,
      extra: JSON.stringify(editForm.value.extra)
    }
    if (editingItem.value) {
      await auxApi.update(currentBook.value, auxType.value, editingItem.value.id, payload)
    } else {
      await auxApi.create(currentBook.value, auxType.value, payload)
    }
    ElMessage.success('保存成功')
    showEdit.value = false
    editingItem.value = null
    loadAux()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}

const deleteAux = async (row) => {
  await ElMessageBox.confirm(`确定删除 ${row.name}？`, '确认')
  try {
    await auxApi.delete(currentBook.value, auxType.value, row.id)
    ElMessage.success('已删除')
    loadAux()
  } catch (e) { ElMessage.error('删除失败') }
}

const batchDelete = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedItems.value.length} 条？`, '确认')
  try {
    const ids = selectedItems.value.map(r => r.id)
    await auxApi.batchDelete(currentBook.value, auxType.value, ids)
    ElMessage.success('批量删除成功')
    loadAux()
  } catch (e) { ElMessage.error('删除失败') }
}

const exportData = async () => {
  try {
    const url = auxApi.exportUrl(currentBook.value, auxType.value)
    const token = localStorage.getItem('token')
    window.open(`${url}?token=${token}`, '_blank')
  } catch (e) { ElMessage.error('导出失败') }
}

const downloadTemplate = () => {
  const templates = {
    customer: '编码,名称,联系人,电话,地址,备注\nK001,示例客户,张三,13800000000,,测试数据',
    supplier: '编码,名称,联系人,电话,地址,备注\nG001,示例供应商,李四,13900000000,,测试数据',
    department: '编码,名称,备注\nBM01,总经理室,',
    project: '编码,名称,状态,开始日期,结束日期,备注\nXM01,示例项目,进行中,2026-01-01,,测试项目',
    employee: '编码,名称,部门,电话,备注\nYG01,张三,BM01,13800000000,',
    warehouse: '编码,名称,地址,备注\nCK01,主仓库,,',
    bank_account: '编码,名称,账号,开户行,户主,地址,备注\nYH01,工行基本户,1234567890,工商银行,,,'
  }
  const csv = '\uFEFF' + (templates[auxType.value] || '编码,名称,备注\n001,示例,')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${auxType.value}_template.csv`
  a.click()
  URL.revokeObjectURL(url)
}

const onImportSuccess = (resp) => {
  ElMessage.success(resp.message || '导入成功')
  loadAux()
}

const onImportError = () => {
  ElMessage.error('导入失败')
}

// ===== User Management =====
const loadUsers = async () => {
  try {
    const { data } = await userApi.list()
    allUsers.value = data.data || []
  } catch (e) { console.error(e) }
}
const openUserAdd = () => {
  editingUser.value = null
  userForm.value = { username: '', password: '', real_name: '', role: 'user' }
  showUserEdit.value = true
}
const editUser = (row) => {
  editingUser.value = row
  userForm.value = { username: row.username, password: '', real_name: row.real_name, role: row.role }
  showUserEdit.value = true
}
const saveUser = async () => {
  try {
    if (editingUser.value) {
      const payload = { real_name: userForm.value.real_name, role: userForm.value.role }
      if (userForm.value.password) payload.password = userForm.value.password
      await userApi.update(editingUser.value.id, payload)
    } else {
      await userApi.create(userForm.value)
    }
    ElMessage.success('保存成功')
    showUserEdit.value = false
    loadUsers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '保存失败') }
}
const deleteUser = async (row) => {
  await ElMessageBox.confirm(`确定删除用户 "${row.username}"？`, '确认')
  await userApi.delete(row.id)
  ElMessage.success('已删除')
  loadUsers()
}
const resetUserPwd = async (row) => {
  const { value: pwd } = await ElMessageBox.prompt('请输入新密码', `重置 ${row.username} 的密码`, {
    inputPattern: /.{6,}/, inputErrorMessage: '密码至少6位'
  })
  await userApi.resetPassword(row.id, { password: pwd })
  ElMessage.success('密码已重置')
}

// ===== Book Permissions =====
const loadBookUsers = async () => {
  if (!permBookId.value) return
  try {
    const { data } = await bookUserApi.list(permBookId.value)
    bookUsers.value = data.data || []
  } catch (e) { console.error(e) }
}
const openPermAdd = () => {
  permForm.value = { user_id: null, role: 'full' }
  showPermAdd.value = true
}
const savePerm = async () => {
  try {
    await bookUserApi.create(permBookId.value, permForm.value)
    ElMessage.success('添加成功')
    showPermAdd.value = false
    loadBookUsers()
  } catch (e) { ElMessage.error(e.response?.data?.error || '添加失败') }
}
const updatePerm = async (row) => {
  try {
    await bookUserApi.update(permBookId.value, row.id, { role: row.role })
    ElMessage.success('更新成功')
  } catch (e) { ElMessage.error('更新失败') }
}
const removePerm = async (row) => {
  await ElMessageBox.confirm(`确定移除用户 "${row.username}" 的权限？`, '确认')
  await bookUserApi.delete(permBookId.value, row.id)
  ElMessage.success('已移除')
  loadBookUsers()
}

// ===== Operation Logs =====
const loadLogs = async () => {
  try {
    const params = { page: logPage.value, page_size: 50, ...logFilters.value }
    const { data } = await systemApi.logs.list(params)
    logs.value = data.data || []
    logTotal.value = data.total || 0
  } catch (e) { console.error(e) }
}

// ===== Backup & Restore =====
const loadBackups = async () => {
  try {
    const { data } = await systemApi.backups.list()
    backups.value = data.data || []
  } catch (e) { console.error(e) }
}
const createBackup = async () => {
  backupLoading.value = true
  try {
    const { data } = await systemApi.backups.create()
    ElMessage.success(data.message || '备份成功')
    loadBackups()
  } catch (e) { ElMessage.error('备份失败') }
  backupLoading.value = false
}
const downloadBackup = (name) => {
  const token = localStorage.getItem('token')
  window.open(`${systemApi.backups.download(name)}?token=${token}`, '_blank')
}
const restoreBackup = async (name) => {
  await ElMessageBox.confirm('恢复将覆盖当前数据，确定继续？', '警告', { type: 'warning' })
  try {
    const { data } = await systemApi.backups.restore(name)
    ElMessage.success(data.message || '恢复成功，请重启服务')
  } catch (e) { ElMessage.error('恢复失败') }
}
const deleteBackupFile = async (name) => {
  await ElMessageBox.confirm(`确定删除备份 "${name}"？`, '确认')
  await systemApi.backups.delete(name)
  ElMessage.success('已删除')
  loadBackups()
}

watch(showEdit, (val) => {
  if (!val) editingItem.value = null
})

watch(currentBook, (val) => { if (val) { loadAux(); loadBookInfo(); loadVtplList() } })

onMounted(() => {
  if (currentBook.value) { loadAux(); loadBookInfo(); loadVtplList() }
  loadUsers()
  loadBackups()
  loadLogs()
})
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; flex-wrap: wrap; gap: 8px; }
.page-header h2 { color: #303133; font-size: 18px; }
.toolbar { display: flex; align-items: center; flex-wrap: wrap; gap: 0; margin-bottom: 12px; }
.table-wrapper { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.inner-tabs :deep(.el-tabs__header) { margin-bottom: 8px; }

.menu-sort-list { display: flex; flex-direction: column; gap: 8px; }
.menu-sort-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 14px; background: #f5f7fa; border-radius: 6px;
  border: 1px solid #ebeef5;
}
.menu-sort-left { display: flex; align-items: center; gap: 10px; font-size: 14px; }
.sort-handle { cursor: grab; color: #909399; }
.menu-sort-right { display: flex; align-items: center; gap: 4px; }
</style>
