import { ref } from 'vue'
import { extractErrorMessage } from '../utils/error'

export function useNotice() {
  const notice = ref({
    show: false,
    color: 'info',
    text: '',
    timeout: 3600,
    key: 0,
  })

  function showNotice(color, text, timeout = 3600) {
    if (!text) {
      return
    }

    const duration = Number(timeout)
    notice.value = {
      show: true,
      color,
      text,
      timeout: duration > 0 ? duration : -1,
      key: notice.value.key + 1,
    }
  }

  function closeNotice() {
    notice.value.show = false
  }

  function showError(error, fallback = '操作失败') {
    const message = extractErrorMessage(error)
    showNotice('error', message || fallback, 5200)
  }

  return {
    notice,
    showNotice,
    closeNotice,
    showError,
  }
}
