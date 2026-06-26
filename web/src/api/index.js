import axios from 'axios'

const http = axios

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

export const healthApi = {
  check: () => http.get('/api/health'),
}
