import { ref, watch } from 'vue'
import {
  CheckQRCodeLogin,
  LoginWithCookie,
  Logout,
  StartQRCodeLogin,
} from '../../bindings/sevenplayer/app'

export function useLoginSession({
  loggedIn,
  user,
  showNotice,
  showError,
  onLoggedIn,
  onLoggedOut,
}) {
  const loginLoading = ref(false)
  const cookieSubmitting = ref(false)
  const loginDialog = ref(false)
  const loginTab = ref('qr')
  const sessionId = ref('')
  const qrImage = ref('')
  const loginStatus = ref('等待生成二维码')
  const cookieInput = ref('')
  const pollingTimer = ref(null)

  watch(loginDialog, (opened) => {
    if (!opened) {
      clearPolling()
      resetLoginSession()
      return
    }

    if (loginTab.value === 'qr' && !sessionId.value && !qrImage.value && !loginLoading.value) {
      startLogin().catch((error) => showError(error, '二维码生成失败'))
    }
  })

  watch(loginTab, (tab) => {
    if (!loginDialog.value) {
      return
    }

    if (tab === 'qr') {
      if (!sessionId.value && !qrImage.value && !loginLoading.value) {
        startLogin().catch((error) => showError(error, '二维码生成失败'))
      }
      return
    }

    clearPolling()
    resetLoginSession()
  })

  function openLoginDialog(tab = 'qr') {
    loginTab.value = tab
    loginDialog.value = true
  }

  function closeLoginDialog() {
    loginDialog.value = false
  }

  async function startLogin() {
    loginLoading.value = true
    clearPolling()
    resetLoginSession()

    try {
      const session = await StartQRCodeLogin()
      sessionId.value = session.sessionId
      qrImage.value = session.qrCodeDataUrl
      loginStatus.value = '推荐优先使用 Cookie 登录，使用 APP 扫码会挤掉其他浏览器已登录的会话状态。'

      pollingTimer.value = window.setInterval(() => {
        pollLogin().catch((error) => showError(error, '登录检查失败'))
      }, 2000)
    } catch (error) {
      showError(error, '二维码生成失败')
    } finally {
      loginLoading.value = false
    }
  }

  async function pollLogin() {
    if (!sessionId.value) {
      return
    }

    const result = await CheckQRCodeLogin(sessionId.value)
    loginStatus.value = result?.message || '等待扫码'

    if (!result || result.state === 'waiting' || result.state === 'scanned') {
      return
    }

    if (result.state === 'authenticated' && result.loggedIn) {
      clearPolling()
      resetLoginSession()
      loginDialog.value = false
      loggedIn.value = true
      user.value = result.user ?? null
      showNotice('success', '登录成功，已恢复 115 会话。')
      await onLoggedIn?.()
      return
    }

    if (result.state === 'expired' || result.state === 'canceled') {
      clearPolling()
      loginStatus.value = result.message || '二维码已失效，请重新生成。'
      showNotice('warning', loginStatus.value)
    }
  }

  async function submitCookieLogin() {
    cookieSubmitting.value = true

    try {
      const result = await LoginWithCookie(cookieInput.value)
      loggedIn.value = Boolean(result?.loggedIn)
      user.value = result?.user ?? null
      loginDialog.value = false
      cookieInput.value = ''
      showNotice('success', result?.message || 'Cookie 登录成功。')
      await onLoggedIn?.()
    } catch (error) {
      showError(error, 'Cookie 登录失败')
    } finally {
      cookieSubmitting.value = false
    }
  }

  async function pasteCookieFromClipboard() {
    try {
      cookieInput.value = await navigator.clipboard.readText()
    } catch (error) {
      showError(error, '读取剪贴板失败')
    }
  }

  async function handleLogout() {
    try {
      await Logout()
      clearPolling()
      loginDialog.value = false
      loggedIn.value = false
      user.value = null
      await onLoggedOut?.()
      showNotice('success', '已退出登录，本地会话已清除。')
    } catch (error) {
      showError(error, '退出登录失败')
      throw error
    }
  }

  function clearPolling() {
    if (pollingTimer.value) {
      window.clearInterval(pollingTimer.value)
      pollingTimer.value = null
    }
  }

  function resetLoginSession() {
    sessionId.value = ''
    qrImage.value = ''
    loginStatus.value = '等待生成二维码'
  }

  return {
    loginLoading,
    cookieSubmitting,
    loginDialog,
    loginTab,
    qrImage,
    loginStatus,
    cookieInput,
    openLoginDialog,
    closeLoginDialog,
    startLogin,
    submitCookieLogin,
    pasteCookieFromClipboard,
    handleLogout,
    clearPolling,
  }
}
