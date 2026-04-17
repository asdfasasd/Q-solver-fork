import { ref, reactive } from 'vue'
import { Quit } from '../../wailsjs/runtime/runtime'

export function useUI() {
  const toasts = ref([])
  const activeButtons = reactive({
    toggle: false,
    solve: false,
    clickthrough: false
  })
  const isClickThrough = ref(false)
  const mainVisible = ref(true)
  const isStealthMode = ref(true)
  const hasStarted = ref(false)

  function showToast(text, type = 'error', duration = 2000) {
    const id = Date.now() + Math.random()
    const toast = reactive({ id, text, type, show: false })
    toasts.value.push(toast)
    
    setTimeout(() => { toast.show = true }, 100)
    setTimeout(() => {
      toast.show = false
      setTimeout(() => {
        const idx = toasts.value.findIndex(t => t.id === id)
        if (idx !== -1) toasts.value.splice(idx, 1)
      }, 400)
    }, duration)
  }

  function flash(which) {
    activeButtons[which] = true
    setTimeout(() => { activeButtons[which] = false }, 200)
  }

  function quit() {
    Quit()
  }

  return {
    toasts,
    activeButtons,
    isClickThrough,
    mainVisible,
    isStealthMode,
    hasStarted,
    showToast,
    flash,
    quit
  }
}
