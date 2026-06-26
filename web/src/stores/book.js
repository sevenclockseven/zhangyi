import { ref, watch } from 'vue'

const STORAGE_KEY = 'zhangyi_current_book'

// Shared book selection state across all views
const currentBookId = ref(localStorage.getItem(STORAGE_KEY) ? Number(localStorage.getItem(STORAGE_KEY)) : null)
const books = ref([])
const booksLoaded = ref(false)

watch(currentBookId, (val) => {
  if (val) {
    localStorage.setItem(STORAGE_KEY, String(val))
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
})

export function useBookStore() {
  return {
    currentBookId,
    books,
    booksLoaded,
    setCurrentBook(id) {
      currentBookId.value = id
    },
    clearCurrentBook() {
      currentBookId.value = null
    },
    setBooks(list) {
      books.value = list
      booksLoaded.value = true
      // Auto-select first book if none selected
      if (!currentBookId.value && list.length > 0) {
        currentBookId.value = list[0].id
      }
      // Clear if selected book no longer exists
      if (currentBookId.value && !list.find(b => b.id === currentBookId.value)) {
        currentBookId.value = list.length > 0 ? list[0].id : null
      }
    }
  }
}
