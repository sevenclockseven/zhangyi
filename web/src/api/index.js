import axios from 'axios'

const http = axios

// Track active requests for cancellation on book switch
let activeRequestControllers = new Map()
let requestCounter = 0

export function cancelBookRequests() {
  activeRequestControllers.forEach((controller) => {
    controller.abort()
  })
  activeRequestControllers.clear()
}

// Add request interceptor to track cancellable requests
http.interceptors.request.use((config) => {
  // Skip auth/health requests from tracking
  if (config.url?.includes('/auth/login') || config.url?.includes('/health')) {
    return config
  }
  const requestId = ++requestCounter
  const controller = new AbortController()
  config.signal = controller.signal
  activeRequestControllers.set(requestId, controller)
  config._requestId = requestId
  return config
})

http.interceptors.response.use(
  (response) => {
    const requestId = response.config?._requestId
    if (requestId) {
      activeRequestControllers.delete(requestId)
    }
    return response
  },
  (error) => {
    if (error.config?._requestId) {
      activeRequestControllers.delete(error.config._requestId)
    }
    // Ignore aborted requests (from book switch)
    if (error.name === 'CanceledError' || error.code === 'ERR_CANCELED') {
      return Promise.reject({ __cancelled: true })
    }
    return Promise.reject(error)
  }
)

export const bookApi = {
  list: () => http.get('/api/books'),
  get: (id) => http.get(`/api/books/${id}`),
  create: (data) => http.post('/api/books', data),
  update: (id, data) => http.put(`/api/books/${id}`, data),
  delete: (id) => http.delete(`/api/books/${id}`),
  syncTemplate: (id, data) => http.post(`/api/books/${id}/sync-template`, data),
  syncAllTemplates: (id, data) => http.post(`/api/books/${id}/sync-all-templates`, data),
  trialBalance: (id) => http.get(`/api/books/${id}/trial-balance`),
}

export const accountApi = {
  list: (bookId) => http.get(`/api/books/${bookId}/accounts`),
  tree: (bookId) => http.get(`/api/books/${bookId}/accounts/tree`),
  create: (bookId, data) => http.post(`/api/books/${bookId}/accounts`, data),
  update: (bookId, acid, data) => http.put(`/api/books/${bookId}/accounts/${acid}`, data),
  delete: (bookId, acid) => http.delete(`/api/books/${bookId}/accounts/${acid}`),
}

export const voucherApi = {
  list: (bookId, params) => http.get(`/api/books/${bookId}/vouchers`, { params }),
  get: (bookId, id) => http.get(`/api/books/${bookId}/vouchers/${id}`),
  create: (bookId, data) => http.post(`/api/books/${bookId}/vouchers`, data),
  update: (bookId, id, data) => http.put(`/api/books/${bookId}/vouchers/${id}`, data),
  delete: (bookId, id) => http.delete(`/api/books/${bookId}/vouchers/${id}`),
  review: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/review`),
  unreview: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/unreview`),
  post: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/post`),
  unpost: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/unpost`),
  void: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/void`),
  restore: (bookId, id) => http.post(`/api/books/${bookId}/vouchers/${id}/restore`),
  batchReview: (bookId, ids) => http.post(`/api/books/${bookId}/vouchers/batch-review`, { ids }),
  batchPost: (bookId, ids) => http.post(`/api/books/${bookId}/vouchers/batch-post`, { ids }),
  exportUrl: (bookId) => `/api/books/${bookId}/vouchers/export`,
}

export const voucherTemplateApi = {
  list: (bookId) => http.get(`/api/books/${bookId}/voucher-templates`),
  create: (bookId, data) => http.post(`/api/books/${bookId}/voucher-templates`, data),
  update: (bookId, tid, data) => http.put(`/api/books/${bookId}/voucher-templates/${tid}`, data),
  delete: (bookId, tid) => http.delete(`/api/books/${bookId}/voucher-templates/${tid}`),
}

export const reportApi = {
  balanceSheet: (bookId) => http.get(`/api/books/${bookId}/reports/balance-sheet`),
  incomeStatement: (bookId) => http.get(`/api/books/${bookId}/reports/income-statement`),
  cashFlow: (bookId) => http.get(`/api/books/${bookId}/reports/cash-flow`),
  accountBalance: (bookId) => http.get(`/api/books/${bookId}/reports/account-balance`),
  incomeStatementV2: (bookId, period) => http.get(`/api/books/${bookId}/reports/income-statement-v2`, { params: { period } }),
  expense: (bookId) => http.get(`/api/books/${bookId}/reports/expense`),
  generalLedger: (bookId) => http.get(`/api/books/${bookId}/reports/general-ledger`),
  arAp: (bookId) => http.get(`/api/books/${bookId}/reports/ar-ap`),
  custom: (bookId, rid) => http.get(`/api/books/${bookId}/reports/custom/${rid}`),
  exportUrl: (bookId) => `/api/books/${bookId}/reports/export`,
  templates: {
    list: (bookId) => http.get(`/api/books/${bookId}/reports/templates`),
    create: (bookId, data) => http.post(`/api/books/${bookId}/reports/templates`, data),
    update: (bookId, tid, data) => http.put(`/api/books/${bookId}/reports/templates/${tid}`, data),
    delete: (bookId, tid) => http.delete(`/api/books/${bookId}/reports/templates/${tid}`),
  },
}

export const auxApi = {
  list: (bookId, type) => http.get(`/api/books/${bookId}/aux/${type}`),
  create: (bookId, type, data) => http.post(`/api/books/${bookId}/aux/${type}`, data),
  update: (bookId, type, aid, data) => http.put(`/api/books/${bookId}/aux/${type}/${aid}`, data),
  delete: (bookId, type, aid) => http.delete(`/api/books/${bookId}/aux/${type}/${aid}`),
  exportUrl: (bookId, type) => `/api/books/${bookId}/aux/${type}/export`,
  import: (bookId, type, data) => http.post(`/api/books/${bookId}/aux/${type}/import`, data),
  batchDelete: (bookId, type, ids) => http.post(`/api/books/${bookId}/aux/${type}/batch-delete`, { ids }),
}

export const openingApi = {
  get: (bookId) => http.get(`/api/books/${bookId}/opening-balances`),
  save: (bookId, data) => http.post(`/api/books/${bookId}/opening-balances`, data),
  exportUrl: (bookId) => `/api/books/${bookId}/opening-balances/export`,
  import: (bookId, data) => http.post(`/api/books/${bookId}/opening-balances/import`, data),
}

export const closingApi = {
  status: (bookId) => http.get(`/api/books/${bookId}/closing/status`),
  autoTransfer: (bookId) => http.post(`/api/books/${bookId}/closing/auto-transfer`),
  close: (bookId) => http.post(`/api/books/${bookId}/closing/close`),
  unclose: (bookId) => http.post(`/api/books/${bookId}/closing/unclose`),
}

export const ledgerApi = {
  journal: (bookId, params) => http.get(`/api/books/${bookId}/ledger/journal`, { params }),
  multiColumn: (bookId, params) => http.get(`/api/books/${bookId}/ledger/multi-column`, { params }),
}

export const authApi = {
  login: (data) => http.post('/api/auth/login', data),
  register: (data) => http.post('/api/auth/register', data),
  me: () => http.get('/api/auth/me'),
  changePassword: (data) => http.put('/api/auth/password', data),
}

export const templateApi = {
  versions: () => http.get('/api/templates/versions'),
  manifest: () => http.get('/api/templates/manifest'),
}


export const systemApi = {
  backups: {
    list: () => http.get('/api/system/backups'),
    create: () => http.post('/api/system/backups'),
    download: (name) => `/api/system/backups/${name}`,
    delete: (name) => http.delete(`/api/system/backups/${name}`),
    restore: (name) => http.post(`/api/system/backups/${name}/restore`),
  },
  logs: {
    list: (params) => http.get('/api/system/logs', { params }),
  },
}

export const bookUserApi = {
  list: (bookId) => http.get(`/api/books/${bookId}/users`),
  create: (bookId, data) => http.post(`/api/books/${bookId}/users`, data),
  update: (bookId, buid, data) => http.put(`/api/books/${bookId}/users/${buid}`, data),
  delete: (bookId, buid) => http.delete(`/api/books/${bookId}/users/${buid}`),
}

export const userApi = {
  list: () => http.get('/api/users'),
  create: (data) => http.post('/api/users', data),
  update: (uid, data) => http.put(`/api/users/${uid}`, data),
  delete: (uid) => http.delete(`/api/users/${uid}`),
  resetPassword: (uid, data) => http.put(`/api/users/${uid}/reset-password`, data),
}

export const assetApi = {
  // 分类
  listCategories: (bookId) => http.get(`/api/books/${bookId}/assets/categories`),
  createCategory: (bookId, data) => http.post(`/api/books/${bookId}/assets/categories`, data),
  updateCategory: (bookId, aid, data) => http.put(`/api/books/${bookId}/assets/categories/${aid}`, data),
  deleteCategory: (bookId, aid) => http.delete(`/api/books/${bookId}/assets/categories/${aid}`),
  // 卡片
  listCards: (bookId, params) => http.get(`/api/books/${bookId}/assets`, { params }),
  getCard: (bookId, cardId) => http.get(`/api/books/${bookId}/assets/${cardId}`),
  createCard: (bookId, data) => http.post(`/api/books/${bookId}/assets`, data),
  updateCard: (bookId, cardId, data) => http.put(`/api/books/${bookId}/assets/${cardId}`, data),
  deleteCard: (bookId, cardId) => http.delete(`/api/books/${bookId}/assets/${cardId}`),
  // 折旧
  calcDepreciation: (bookId, period) => http.get(`/api/books/${bookId}/assets/depreciation/calc`, { params: { period } }),
  runDepreciation: (bookId, period) => http.post(`/api/books/${bookId}/assets/depreciation/run`, period ? { period } : {}),
  // 台账
  summary: (bookId) => http.get(`/api/books/${bookId}/assets/summary`),
  changeStatus: (bookId, cardId, data) => http.put(`/api/books/${bookId}/assets/${cardId}/status`, data),
  getTransactions: (bookId, cardId) => http.get(`/api/books/${bookId}/assets/transactions/${cardId}`),
  getAllTransactions: (bookId) => http.get(`/api/books/${bookId}/assets/transactions`),
  importAssets: (bookId, data) => http.post(`/api/books/${bookId}/assets/import`, data),
  exportAssets: (bookId) => http.get(`/api/books/${bookId}/assets/export`),
}

export const healthApi = {
  check: () => http.get('/api/health'),
}
