import { reactive, readonly } from 'vue'

interface Toast {
  message: string
  type: 'success' | 'error'
  visible: boolean
}

const toast = reactive<Toast>({
  message: '',
  type: 'success',
  visible: false
})

let timer: NodeJS.Timeout | null = null

export const useToast = () => {
  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    if (timer) clearTimeout(timer)
    
    toast.message = message
    toast.type = type
    toast.visible = true
    
    timer = setTimeout(() => {
      toast.visible = false
    }, 5000)
  }

  const showSuccess = (message: string) => showToast(message, 'success')
  const showError = (message: string) => showToast(message, 'error')
  const clearToast = () => {
    toast.visible = false
    if (timer) clearTimeout(timer)
  }

  return {
    toast: readonly(toast),
    showSuccess,
    showError,
    clearToast
  }
}
