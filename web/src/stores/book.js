import { ref, watch } from 'vue'

const STORAGE_KEY = 'zhangyi_current_book'

// Shared book selection state across all views
const currentBookId = ref(localStorage.getItem(STORAGE_KEY) ? Number(localStorage.getItem(STORAGE_KEY)) : null)

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
    setCurrentBook(id) {
      currentBookId.value = id
    },
    clearCurrentBook() {
      currentBookId.value = null
    }
  }
}
