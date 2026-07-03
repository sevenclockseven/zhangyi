import { ref, computed, watch } from 'vue'
import { cancelBookRequests } from '../api/index.js'

const STORAGE_KEY = 'zhangyi_current_book'

// Shared book selection state across all views
const currentBookId = ref(localStorage.getItem(STORAGE_KEY) ? Number(localStorage.getItem(STORAGE_KEY)) : null)
const books = ref([])
const booksLoaded = ref(false)

// Book permissions: { bookId: 'admin'|'writable'|'readonly' }
const bookPermissions = ref({})

// Current user global role
const userRole = ref('')

// Load from localStorage on init
try {
  const saved = localStorage.getItem('book_permissions')
  if (saved) bookPermissions.value = JSON.parse(saved)
  const user = localStorage.getItem('user')
  if (user) userRole.value = JSON.parse(user).role || ''
} catch {}

watch(currentBookId, (val) => {
  // Cancel in-flight requests when switching books to prevent race conditions
  cancelBookRequests()
  if (val) {
    localStorage.setItem(STORAGE_KEY, String(val))
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
})

export function useBookStore() {
  // Current book's role for this user
  const currentBookRole = computed(() => {
    if (!currentBookId.value) return ''
    return bookPermissions.value[currentBookId.value] || ''
  })

  // Is this user admin globally?
  const isAdmin = computed(() => userRole.value === 'admin')

  // Can the user write (create/edit/delete/review/post) in current book?
  const canWrite = computed(() => {
    if (isAdmin.value) return true
    const role = currentBookRole.value
    return role === 'admin' || role === 'writable' || role === 'full'
  })

  // Can the user manage system settings?
  const canManageSystem = computed(() => isAdmin.value)

  // Set permissions (called after login)
  function setPermissions(perms) {
    bookPermissions.value = perms || {}
    localStorage.setItem('book_permissions', JSON.stringify(bookPermissions.value))
  }

  // Set user role (called after login)
  function setUserRole(role) {
    userRole.value = role || ''
  }

  // Clear all on logout
  function clearAuth() {
    bookPermissions.value = {}
    userRole.value = ''
    localStorage.removeItem('book_permissions')
  }

  return {
    currentBookId,
    books,
    booksLoaded,
    currentBookRole,
    isAdmin,
    canWrite,
    canManageSystem,
    setPermissions,
    setUserRole,
    clearAuth,
    setCurrentBook(id) {
      currentBookId.value = id
    },
    clearCurrentBook() {
      currentBookId.value = null
    },
    setBooks(list) {
      // 非管理员只显示有权限的账套
      let filtered = list
      if (!isAdmin.value && Object.keys(bookPermissions.value).length > 0) {
        filtered = list.filter(b => bookPermissions.value[b.id])
      }
      books.value = filtered
      booksLoaded.value = true
      // Auto-select first book if none selected
      if (!currentBookId.value && filtered.length > 0) {
        currentBookId.value = filtered[0].id
      }
      // Clear if selected book no longer exists
      if (currentBookId.value && !filtered.find(b => b.id === currentBookId.value)) {
        currentBookId.value = filtered.length > 0 ? filtered[0].id : null
      }
    }
  }
}
