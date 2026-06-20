import { ref } from 'vue'

const openCount = ref(0)

export function useModalState() {
  function openModal() {
    openCount.value++
    document.body.classList.add('modal-open')
  }

  function closeModal() {
    if (openCount.value <= 0) return
    openCount.value--
    if (openCount.value === 0) {
      document.body.classList.remove('modal-open')
    }
  }

  return { openCount, openModal, closeModal }
}
