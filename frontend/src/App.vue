<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  AddOfflineTasks,
  Bootstrap,
  CheckQRCodeLogin,
  ClearPlaybackProgress,
  ClearSubtitlePath,
  DeletePlayerPath,
  DeleteOfflineTasks,
  ListDirectory,
  ListOfflineTasks,
  LoginWithCookie,
  Logout,
  PlayFile,
  PreviewDirectory,
  SearchFiles,
  SaveFileListDensity,
  SavePlayerDisabled,
  SavePlayerPath,
  SavePreferredPlayer,
  SaveSmallFileFilterMB,
  SaveShowTitleBadgesEnabled,
  SelectPlayerPath,
  SelectSubtitlePath,
  SelectTorrentFileAsMagnet,
  StartQRCodeLogin,
} from '../bindings/panplayer/app'
import { Clipboard } from '@wailsio/runtime'

const navigationItems = [
  { value: 'files', label: '文件管理', icon: 'mdi-folder-outline' },
  { value: 'downloads', label: '云下载', icon: 'mdi-download-circle-outline' },
  { value: 'settings', label: '系统设置', icon: 'mdi-cog-outline' },
]

const typeOptions = [
  { label: '全部', value: 'all' },
  { label: '目录', value: 'dir' },
  { label: '视频', value: 'video' },
  { label: '其他文件', value: 'file' },
]

const sortOptions = [
  { label: '目录优先', value: 'folders' },
  { label: '名称 A-Z', value: 'name' },
  { label: '最近更新', value: 'updated' },
  { label: '体积最大', value: 'size' },
  { label: '续播优先', value: 'resume' },
]

const smallFileFilterOptions = [
  { label: '关闭', value: 0 },
  { label: '1 MB', value: 1 },
  { label: '2 MB', value: 2 },
  { label: '3 MB', value: 3 },
  { label: '5 MB', value: 5 },
  { label: '10 MB', value: 10 },
]

const fileListDensityOptions = [
  { label: '紧凑', value: 'compact' },
  { label: '默认', value: 'default' },
  { label: '宽松', value: 'comfortable' },
]

const downloadFilterOptions = [
  { value: 'active', label: '正在下载' },
  { value: 'failed', label: '下载失败' },
  { value: 'completed', label: '完成记录' },
]

const pageSizeOptions = [
  { title: '20 / 页', value: 20 },
  { title: '50 / 页', value: 50 },
  { title: '100 / 页', value: 100 },
]

const OFFLINE_TARGET_PICKER_VALUE = '__pick_other__'

const rootBreadcrumb = () => ({ id: '0', name: '我的文件' })

const activeSection = ref('files')
const settingsTab = ref('player')

const booting = ref(true)
const directoryLoading = ref(false)
const searchLoading = ref(false)
const loginLoading = ref(false)
const cookieSubmitting = ref(false)
const actionLoading = ref(false)
const playerLoading = ref(false)
const downloadLoading = ref(false)
const downloadSubmitting = ref(false)
const folderPickerLoading = ref(false)
const torrentSelecting = ref(false)

const loggedIn = ref(false)
const user = ref(null)
const settings = ref(createDefaultSettings())
const proxyBase = ref('')

const currentDir = ref('0')
const parentId = ref('')
const currentPath = ref([rootBreadcrumb()])
const items = ref([])
const directoryTotal = ref(0)
const directoryPage = ref(1)
const selectedItem = ref(null)
const detailDrawer = ref(false)

const loginDialog = ref(false)
const loginTab = ref('qr')
const sessionId = ref('')
const qrImage = ref('')
const loginStatus = ref('等待生成二维码')
const cookieInput = ref('')
const pollingTimer = ref(null)

const searchQuery = ref('')
const typeFilter = ref('all')
const sortMode = ref('folders')
const smallFileFilterMB = ref(0)
const searchField = ref(null)
const searchExpanded = ref(false)
const filterMenuOpen = ref(false)
const searchResults = ref([])
const searchTotal = ref(0)
const searchPage = ref(1)
const pageSize = ref(20)
const downloadPage = ref(1)
const downloadPageSize = ref(20)
const searchDebounceTimer = ref(null)
const searchRequestId = ref(0)
const badgeVisibleCounts = ref({})

const jumpDialogVisible = ref(false)
const jumpInput = ref('')
const jumpTargetKey = ref('')

const badgeRowElements = new Map()
let badgeMeasureCanvas = null
let badgeRefreshFrame = 0

const notice = ref({
  show: false,
  color: 'info',
  text: '',
  timeout: 3600,
  key: 0,
})

const downloadDialog = ref(false)
const downloadInput = ref('')
const downloadFilter = ref('active')
const downloadDeleteFiles = ref(false)
const downloadState = ref(createEmptyOfflineState())
const offlineTargetDir = ref(createDirectoryTarget('0', [rootBreadcrumb()]))
const knownDirectoryPaths = ref({
  0: [rootBreadcrumb()],
})

const folderPickerDialog = ref(false)
const folderPickerDirId = ref('0')
const folderPickerPath = ref([rootBreadcrumb()])
const folderPickerFolders = ref([])

const playerOptions = computed(() => settings.value?.players ?? [])

const selectedPlayer = computed(() => {
  const preferred = settings.value?.preferredPlayer ?? 'mpv'
  return playerOptions.value.find((item) => item.id === preferred) ?? playerOptions.value[0] ?? null
})

const breadcrumbItems = computed(() => {
  if (currentPath.value?.length) {
    return currentPath.value
  }
  return [rootBreadcrumb()]
})

const breadcrumbDisplayItems = computed(() =>
  breadcrumbItems.value.map((crumb, index) => ({
    id: crumb.id,
    title: crumb.name,
    disabled: index === breadcrumbItems.value.length - 1,
  })),
)

const folderPickerBreadcrumbDisplayItems = computed(() =>
  folderPickerPath.value.map((crumb, index) => ({
    id: crumb.id,
    title: crumb.name,
    disabled: index === folderPickerPath.value.length - 1,
  })),
)

const currentDirectoryName = computed(() => {
  const tail = breadcrumbItems.value[breadcrumbItems.value.length - 1]
  return tail?.name || '我的文件'
})

const accountDisplayName = computed(() => {
  if (!loggedIn.value || !user.value) {
    return '未登录'
  }
  return user.value.userName || '115 用户'
})
const accountAvatarUrl = computed(() => buildAccountAvatarUrl(user.value?.faceUrl))
const accountVipLevelText = computed(() => {
  if (!loggedIn.value || !user.value) {
    return '--'
  }
  if (user.value.vipLabel) {
    return user.value.vipLabel
  }
  return user.value.isVip ? 'VIP' : '普通用户'
})
const accountVipExpireText = computed(() => {
  if (!loggedIn.value || !user.value) {
    return '--'
  }
  if (!user.value.isVip) {
    return '非 VIP'
  }
  if (user.value.vipForever) {
    return '永久'
  }
  if (user.value.vipExpireAt) {
    return formatDateOnly(user.value.vipExpireAt)
  }
  return '--'
})
const accountSpaceTotalText = computed(() => {
  const total = Number(user.value?.spaceTotal || 0)
  return total > 0 ? formatStorageSize(total) : '--'
})
const accountSpaceUsedText = computed(() => {
  const used = Number(user.value?.spaceUsed || 0)
  return used > 0 ? formatStorageSize(used) : '0 B'
})
const accountSpaceUsageText = computed(() => {
  const total = Number(user.value?.spaceTotal || 0)
  const used = Number(user.value?.spaceUsed || 0)
  if (total > 0 && used >= 0) {
    return `${used > 0 ? formatStorageSize(used) : '0 B'} / ${formatStorageSize(total)}`
  }
  if (used > 0) {
    return formatStorageSize(used)
  }
  return '--'
})
const accountSpaceRemainText = computed(() => {
  const remain = Number(user.value?.spaceRemain || 0)
  return remain > 0 ? formatStorageSize(remain) : '--'
})
const accountSpacePercent = computed(() => {
  const total = Number(user.value?.spaceTotal || 0)
  const used = Math.max(0, Number(user.value?.spaceUsed || 0))
  if (!(total > 0)) {
    return 0
  }
  return Math.min(100, Math.max(0, (used / total) * 100))
})
const accountSpacePercentText = computed(() => {
  if (!(Number(user.value?.spaceTotal || 0) > 0)) {
    return '--'
  }
  return `${accountSpacePercent.value.toFixed(accountSpacePercent.value >= 10 ? 1 : 2)}%`
})
const isGlobalSearchActive = computed(() => searchQuery.value.trim().length > 0)
const isSearchInputVisible = computed(() => searchExpanded.value || isGlobalSearchActive.value)
const fileListDensityValue = computed(() => normalizeFileListDensity(settings.value?.fileListDensity))
const fileListDensityClass = computed(() => `files-density--${fileListDensityValue.value}`)
const fileListAvatarSize = computed(() => {
  if (fileListDensityValue.value === 'compact') return 26
  if (fileListDensityValue.value === 'comfortable') return 30
  return 28
})
const fileListIconSize = computed(() => {
  if (fileListDensityValue.value === 'compact') return 14
  if (fileListDensityValue.value === 'comfortable') return 17
  return 16
})
const sourceItems = computed(() => (isGlobalSearchActive.value ? searchResults.value : items.value))
const activePage = computed(() => (isGlobalSearchActive.value ? searchPage.value : directoryPage.value))
const activeResultTotal = computed(() => (isGlobalSearchActive.value ? searchTotal.value : directoryTotal.value))
const pageCount = computed(() => {
  const total = Number(activeResultTotal.value || 0)
  return Math.max(1, Math.ceil(total / pageSize.value) || 1)
})
const showPagination = computed(() => loggedIn.value && activeResultTotal.value > 0)
const paginationSummaryText = computed(() => {
  const total = Number(activeResultTotal.value || 0)
  if (!total) {
    return '0 项'
  }

  const currentPage = activePage.value
  const start = ((currentPage - 1) * pageSize.value) + 1
  const visibleCount = sourceItems.value.length
  const end = visibleCount > 0
    ? Math.min(total, start + visibleCount - 1)
    : Math.min(total, currentPage * pageSize.value)
  return `第 ${currentPage} 页 · ${start}-${end} / ${total} 项`
})
const searchSummaryText = computed(() => {
  const keyword = searchQuery.value.trim()
  if (!keyword) {
    return ''
  }

  const total = searchTotal.value || searchResults.value.length
  if (total > 0) {
    return `全局搜索 · ${keyword} · ${total} 项 · 第 ${searchPage.value} 页`
  }
  return `全局搜索 · ${keyword}`
})
const fileEmptyText = computed(() => {
  if (isGlobalSearchActive.value) {
    return searchLoading.value ? '正在搜索整个网盘…' : '没有找到匹配结果。'
  }
  return '当前目录没有匹配项。'
})

const filteredItems = computed(() => {
  let list = [...sourceItems.value]

  if (typeFilter.value !== 'all') {
    list = list.filter((item) => {
      if (typeFilter.value === 'dir') return item.isDirectory
      if (typeFilter.value === 'video') return item.isVideo
      if (typeFilter.value === 'file') return !item.isDirectory && !item.isVideo
      return true
    })
  }

  if (smallFileFilterMB.value > 0) {
    list = list.filter((item) => item.isDirectory || Number(item.size || 0) >= smallFileFilterMB.value * 1024 * 1024)
  }

  if (!(isGlobalSearchActive.value && sortMode.value === 'folders')) {
    list.sort((left, right) => compareItems(left, right, sortMode.value))
  }
  return list
})

const selectedSubtitleName = computed(() => {
  if (!selectedItem.value?.subtitlePath) {
    return '未绑定'
  }
  return basename(selectedItem.value.subtitlePath)
})

const selectedResumeText = computed(() => {
  if (!selectedItem.value?.resumeMs) {
    return '暂无续播记录'
  }
  return formatResumeProgressText(selectedItem.value.resumeMs, selectedItem.value.durationSec)
})

const selectedLastPlayedText = computed(() => {
  if (!selectedItem.value?.lastPlayedAt) {
    return '暂无'
  }
  return formatDateTime(selectedItem.value.lastPlayedAt)
})

const downloadTasks = computed(() => downloadState.value?.tasks ?? [])

const filteredDownloadTasks = computed(() =>
  downloadTasks.value.filter((task) => task.statusGroup === downloadFilter.value),
)

const downloadPageCount = computed(() =>
  Math.max(1, Math.ceil(filteredDownloadTasks.value.length / downloadPageSize.value) || 1),
)

const paginatedDownloadTasks = computed(() => {
  const start = (downloadPage.value - 1) * downloadPageSize.value
  return filteredDownloadTasks.value.slice(start, start + downloadPageSize.value)
})

const showDownloadPagination = computed(() => loggedIn.value && filteredDownloadTasks.value.length > 0)

const downloadQuotaCapacity = computed(() => {
  const quota = Math.max(0, Number(downloadState.value?.quota || 0))
  const total = Math.max(0, Number(downloadState.value?.total || 0))
  if (!quota && !total) {
    return 0
  }
  return quota + total
})

const downloadQuotaProgress = computed(() => {
  if (!(downloadQuotaCapacity.value > 0)) {
    return 0
  }
  return Math.min(100, Math.max(0, (Number(downloadState.value?.quota || 0) / downloadQuotaCapacity.value) * 100))
})

const downloadQuotaText = computed(() => {
  if (!(downloadQuotaCapacity.value > 0)) {
    return '0 / 0'
  }
  return `${Math.max(0, Number(downloadState.value?.quota || 0))} / ${downloadQuotaCapacity.value}`
})

const downloadPaginationSummaryText = computed(() => {
  const total = filteredDownloadTasks.value.length
  if (!total) {
    return '0 项'
  }

  const start = ((downloadPage.value - 1) * downloadPageSize.value) + 1
  const end = Math.min(total, start + paginatedDownloadTasks.value.length - 1)
  return `第 ${downloadPage.value} 页 · ${start}-${end} / ${total} 项`
})

const downloadEmptyText = computed(() => {
  if (downloadFilter.value === 'failed') return '当前没有失败任务。'
  if (downloadFilter.value === 'completed') return '当前没有完成记录。'
  return '当前没有进行中的下载任务。'
})

const offlineRecentTargets = computed(() =>
  normalizeOfflineRecentTargets(settings.value?.offlineRecentTargets),
)

const offlineTargetPathText = computed(() =>
  (offlineTargetDir.value?.path ?? []).map((item) => item.name).join(' / ') || '我的文件',
)

const offlineTargetSelectOptions = computed(() => {
  const options = []
  const seen = new Set()
  const activeTarget = normalizeDirectoryTargetValue(offlineTargetDir.value?.id, offlineTargetDir.value?.path)

  if (activeTarget && !offlineRecentTargets.value.some((item) => item.id === activeTarget.id)) {
    options.push(createOfflineTargetOption(activeTarget))
    seen.add(activeTarget.id)
  }

  for (const target of offlineRecentTargets.value) {
    if (seen.has(target.id)) {
      continue
    }
    options.push(createOfflineTargetOption(target))
    seen.add(target.id)
  }

  options.push({
    value: OFFLINE_TARGET_PICKER_VALUE,
    title: '选择其他目录',
    icon: 'mdi-folder-search-outline',
    isPicker: true,
  })

  return options
})

const offlineTargetSelectValue = computed(() => {
  const activeTarget = normalizeDirectoryTargetValue(offlineTargetDir.value?.id, offlineTargetDir.value?.path)
  if (!activeTarget) {
    return OFFLINE_TARGET_PICKER_VALUE
  }

  const matched = offlineTargetSelectOptions.value.find(
    (item) => !item.isPicker && item.target?.id === activeTarget.id,
  )

  return matched?.value || OFFLINE_TARGET_PICKER_VALUE
})

watch(filteredItems, (list) => {
  if (!list.length) {
    selectedItem.value = null
    detailDrawer.value = false
    badgeVisibleCounts.value = {}
    return
  }

  const currentKey = selectedItem.value?.rowKey
  const matched = currentKey ? list.find((item) => item.rowKey === currentKey) : null
  selectedItem.value = matched ?? list[0]
  nextTick(() => {
    queueBadgeVisibilityRefresh()
  })
})

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

watch([activeSection, loggedIn], ([section, signedIn]) => {
  if (section === 'downloads' && signedIn) {
    refreshOfflineTasks().catch((error) => showError(error, '读取云下载失败'))
  } else if (!signedIn) {
    downloadState.value = createEmptyOfflineState()
    searchLoading.value = false
    searchResults.value = []
    searchTotal.value = 0
    searchPage.value = 1
    directoryTotal.value = 0
    directoryPage.value = 1
  }
})

watch(searchQuery, (value) => {
  clearSearchDebounce()
  searchRequestId.value += 1

  const keyword = String(value || '').trim()
  if (!keyword || !loggedIn.value) {
    searchLoading.value = false
    searchResults.value = []
    searchTotal.value = 0
    searchPage.value = 1
    return
  }

  searchPage.value = 1
  searchResults.value = []
  searchTotal.value = 0
  searchDebounceTimer.value = window.setTimeout(() => {
    performGlobalSearch(keyword, searchRequestId.value, 1).catch((error) => showError(error, '搜索失败'))
  }, 260)
})

watch(downloadFilter, () => {
  downloadPage.value = 1
})

watch(filteredDownloadTasks, (tasks) => {
  const maxPage = Math.max(1, Math.ceil(tasks.length / downloadPageSize.value) || 1)
  if (downloadPage.value > maxPage) {
    downloadPage.value = maxPage
  }
})

watch(fileListDensityValue, () => {
  nextTick(() => {
    queueBadgeVisibilityRefresh()
  })
})

onMounted(async () => {
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', handleWindowResize)
  await bootstrapApp()
})

onBeforeUnmount(() => {
  clearPolling()
  clearSearchDebounce()
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleWindowResize)
  if (badgeRefreshFrame) {
    window.cancelAnimationFrame(badgeRefreshFrame)
    badgeRefreshFrame = 0
  }
})

async function bootstrapApp() {
  booting.value = true

  try {
    const boot = await bootstrapWithRetry()
    proxyBase.value = boot?.proxyBase || ''
    loggedIn.value = Boolean(boot?.loggedIn)
    user.value = boot?.user ?? null
    applySettingsView(boot?.settings)
    currentDir.value = boot?.currentId || '0'

    if (loggedIn.value) {
      await loadDirectory(currentDir.value || '0', {}, 1)
    } else {
      resetDirectoryState()
    }
  } catch (error) {
    resetDirectoryState()
    showError(error, '初始化失败')
  } finally {
    booting.value = false
  }
}

async function bootstrapWithRetry() {
  let lastError = null

  for (let attempt = 0; attempt < 8; attempt += 1) {
    try {
      return await Bootstrap()
    } catch (error) {
      lastError = error
      if (!isAppNotReadyError(error) || attempt === 7) {
        break
      }
      await sleep(180 * (attempt + 1))
    }
  }

  throw lastError ?? new Error('初始化失败')
}

async function loadDirectory(dirId, options = {}, page = directoryPage.value) {
  if (!loggedIn.value) {
    return
  }

  directoryLoading.value = true
  const previousKey = selectedItem.value?.rowKey ?? ''
  const nextPage = Math.max(1, Number(page || 1))
  const offset = (nextPage - 1) * pageSize.value

  try {
    const data = await ListDirectory(dirId, offset, pageSize.value)
    const resolvedPath = resolveDirectoryPath(data, dirId, options)
    currentDir.value = data?.dirId || dirId
    parentId.value = data?.parentId || ''
    currentPath.value = resolvedPath
    items.value = (data?.items || []).map(normalizeItem)
    directoryTotal.value = Number(data?.count || items.value.length)
    directoryPage.value = Math.floor(Number(data?.offset || 0) / Number(data?.limit || pageSize.value || 1)) + 1
    rememberKnownPath(currentDir.value, resolvedPath)
    syncSelection(previousKey)
  } catch (error) {
    if (dirId !== '0') {
      showNotice('warning', '目录不可用，已自动回到根目录。')
      await loadDirectory('0', {}, 1)
      return
    }
    showError(error, '读取目录失败')
  } finally {
    directoryLoading.value = false
  }
}

async function loadFolderPicker(dirId) {
  folderPickerLoading.value = true

  try {
    const data = await PreviewDirectory(dirId)
    const resolvedPath = resolveDirectoryPath(data, dirId)
    folderPickerDirId.value = data?.dirId || dirId
    folderPickerPath.value = resolvedPath
    rememberKnownPath(folderPickerDirId.value, resolvedPath)
    folderPickerFolders.value = (data?.items || [])
      .filter((item) => item.isDirectory)
      .map(normalizeItem)
  } catch (error) {
    showError(error, '读取目录失败')
  } finally {
    folderPickerLoading.value = false
  }
}

async function performGlobalSearch(keyword, requestId = searchRequestId.value, page = searchPage.value) {
  if (!loggedIn.value) {
    searchLoading.value = false
    searchResults.value = []
    searchTotal.value = 0
    return
  }

  searchLoading.value = true
  const nextPage = Math.max(1, Number(page || 1))
  const offset = (nextPage - 1) * pageSize.value

  try {
    const data = await SearchFiles(keyword, offset, pageSize.value)
    if (requestId !== searchRequestId.value || keyword !== searchQuery.value.trim()) {
      return
    }

    searchResults.value = (data?.items || []).map(normalizeItem)
    searchTotal.value = Number(data?.count || searchResults.value.length)
    searchPage.value = Math.floor(Number(data?.offset || 0) / Number(data?.limit || pageSize.value || 1)) + 1
    syncSelection(selectedItem.value?.rowKey ?? '')
  } catch (error) {
    if (requestId !== searchRequestId.value) {
      return
    }
    searchResults.value = []
    searchTotal.value = 0
    throw error
  } finally {
    if (requestId === searchRequestId.value) {
      searchLoading.value = false
    }
  }
}

function clearSearchDebounce() {
  if (searchDebounceTimer.value) {
    window.clearTimeout(searchDebounceTimer.value)
    searchDebounceTimer.value = null
  }
}

function syncSelection(preferredKey) {
  const baseList = isGlobalSearchActive.value ? searchResults.value : items.value
  if (!baseList.length) {
    selectedItem.value = null
    detailDrawer.value = false
    return
  }

  const next =
    baseList.find((item) => item.rowKey === preferredKey) ??
    filteredItems.value[0] ??
    baseList[0]

  selectedItem.value = next ?? null
}

function resetDirectoryState() {
  currentDir.value = '0'
  parentId.value = ''
  currentPath.value = [rootBreadcrumb()]
  items.value = []
  directoryTotal.value = 0
  directoryPage.value = 1
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  selectedItem.value = null
  detailDrawer.value = false
  offlineTargetDir.value = createDirectoryTarget('0', [rootBreadcrumb()])
  knownDirectoryPaths.value = {
    0: [rootBreadcrumb()],
  }
}

function rememberKnownPath(dirId, path) {
  const normalizedPath = normalizeBreadcrumbPath(path)
  if (!dirId || !normalizedPath.length) {
    return
  }

  knownDirectoryPaths.value = {
    ...knownDirectoryPaths.value,
    [String(dirId)]: normalizedPath,
  }
}

function normalizeBreadcrumbPath(path) {
  const normalized = Array.isArray(path)
    ? path
      .map((item) => ({
        id: String(item?.id || ''),
        name: String(item?.name || '').trim(),
      }))
      .filter((item) => item.id && item.name)
    : []

  if (!normalized.length || normalized[0].id !== '0') {
    return [rootBreadcrumb(), ...normalized.filter((item) => item.id !== '0')]
  }

  return normalized
}

function resolveDirectoryPath(data, dirId, options = {}) {
  const normalizedDirId = String(data?.dirId || dirId || '0')
  const responsePath = normalizeBreadcrumbPath(data?.path)
  const fallbackPath = normalizeBreadcrumbPath(options?.fallbackPath)
  const cachedPath = normalizeBreadcrumbPath(knownDirectoryPaths.value?.[normalizedDirId])
  const displayName = String(data?.name || options?.fallbackName || '').trim()

  if (normalizedDirId === '0') {
    return [rootBreadcrumb()]
  }

  if (responsePath.length > 1) {
    return responsePath
  }

  if (fallbackPath.length > 1) {
    return fallbackPath
  }

  if (cachedPath.length > 1) {
    return cachedPath
  }

  const parentPath = normalizeBreadcrumbPath(knownDirectoryPaths.value?.[String(data?.parentId || '')])
  if (parentPath.length > 0 && displayName) {
    return [...parentPath, { id: normalizedDirId, name: displayName }]
  }

  if (displayName) {
    return [rootBreadcrumb(), { id: normalizedDirId, name: displayName }]
  }

  return [rootBreadcrumb()]
}

function breadcrumbPathUntil(dirId) {
  const index = breadcrumbItems.value.findIndex((item) => item.id === dirId)
  if (index === -1) {
    return [rootBreadcrumb()]
  }
  return breadcrumbItems.value.slice(0, index + 1)
}

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
    await loadDirectory(currentDir.value || '0', {}, 1)
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
    await loadDirectory(currentDir.value || '0', {}, 1)
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
  actionLoading.value = true

  try {
    await Logout()
    clearPolling()
    loginDialog.value = false
    loggedIn.value = false
    user.value = null
    searchQuery.value = ''
    downloadState.value = createEmptyOfflineState()
    resetDirectoryState()
    showNotice('success', '已退出登录，本地会话已清除。')
  } catch (error) {
    showError(error, '退出登录失败')
  } finally {
    actionLoading.value = false
  }
}

function openDetails(item) {
  selectedItem.value = item
  detailDrawer.value = true
}

async function handlePrimaryAction(item) {
  if (!item) {
    return
  }

  if (item.isDirectory) {
    detailDrawer.value = false
    if (isGlobalSearchActive.value) {
      searchQuery.value = ''
      await nextTick()
      await loadDirectory(item.fileId, {
        fallbackName: item.name,
      }, 1)
      return
    }
    await loadDirectory(item.fileId, {
      fallbackName: item.name,
      fallbackPath: [...breadcrumbItems.value, { id: item.fileId, name: item.name }],
    }, 1)
    return
  }

  if (!item.isVideo) {
    openDetails(item)
    showNotice('info', '当前文件不是常见视频格式，暂不支持直接播放。')
    return
  }

  await playVideo(item)
}

async function playVideo(item, options = {}) {
  if (!item?.isVideo) {
    return
  }

  actionLoading.value = true
  selectedItem.value = item

  try {
    const result = await PlayFile({
      pickCode: item.pickCode,
      name: item.name,
      startMs: options.startMs || 0,
      fromStart: Boolean(options.fromStart),
      subtitle: options.subtitle || item.subtitlePath || '',
    })

    applyPlaybackState(item.pickCode, {
      resumeMs: options.fromStart ? 0 : (result?.startMs || item.resumeMs || 0),
      subtitlePath:
        typeof result?.subtitle === 'string' ? result.subtitle : (item.subtitlePath || ''),
      lastPlayedAt: new Date().toISOString(),
    })

    const segments = [`已交给 ${result?.playerName || '播放器'}`]
    if (result?.resumeUsed && result?.startMs > 0) {
      segments.push(`从 ${formatDurationMs(result.startMs)} 继续`)
    } else if (result?.startMs > 0) {
      segments.push(`从 ${formatDurationMs(result.startMs)} 开始`)
    }
    if (result?.subtitle) {
      segments.push(`字幕 ${basename(result.subtitle)}`)
    }

    showNotice('success', segments.join(' · '), 4200)
  } catch (error) {
    showError(error, '启动播放失败')
  } finally {
    actionLoading.value = false
  }
}

async function chooseSubtitle(item = selectedItem.value) {
  if (!item?.isVideo) {
    return
  }

  actionLoading.value = true

  try {
    const result = await SelectSubtitlePath(item.pickCode)
    applyPlaybackState(item.pickCode, result)
    showNotice('success', result?.subtitleName || '外挂字幕路径已保存。')
  } catch (error) {
    showError(error, '选择字幕失败')
  } finally {
    actionLoading.value = false
  }
}

async function clearSubtitle(item = selectedItem.value) {
  if (!item?.isVideo) {
    return
  }

  actionLoading.value = true

  try {
    const result = await ClearSubtitlePath(item.pickCode)
    applyPlaybackState(item.pickCode, result)
    showNotice('success', `已移除 ${item.displayName || item.name} 的字幕绑定。`)
  } catch (error) {
    showError(error, '清除字幕失败')
  } finally {
    actionLoading.value = false
  }
}

async function clearProgress(item = selectedItem.value) {
  if (!item?.isVideo) {
    return
  }

  actionLoading.value = true

  try {
    const result = await ClearPlaybackProgress(item.pickCode)
    applyPlaybackState(item.pickCode, result)
    showNotice('success', `已清除 ${item.displayName || item.name} 的续播记录。`)
  } catch (error) {
    showError(error, '清除续播失败')
  } finally {
    actionLoading.value = false
  }
}

function applyPlaybackState(pickCode, patch) {
  if (!pickCode) {
    return
  }

  const nextResumeMs = Number(patch?.resumeMs || 0)
  const nextSubtitlePath = patch && 'subtitlePath' in patch ? (patch.subtitlePath || '') : undefined
  const nextLastPlayedAt = patch && 'lastPlayedAt' in patch ? (patch.lastPlayedAt || '') : undefined

  items.value = items.value.map((entry) => {
    if (entry.pickCode !== pickCode) {
      return entry
    }

    return normalizeItem({
      ...entry,
      resumeMs: nextResumeMs,
      subtitlePath: nextSubtitlePath !== undefined ? nextSubtitlePath : entry.subtitlePath,
      lastPlayedAt: nextLastPlayedAt !== undefined ? nextLastPlayedAt : entry.lastPlayedAt,
    })
  })

  if (selectedItem.value?.pickCode === pickCode) {
    const matched = items.value.find((entry) => entry.pickCode === pickCode)
    selectedItem.value = matched ?? selectedItem.value
  }
}

function openJumpDialog(item = selectedItem.value) {
  if (!item?.isVideo) {
    return
  }

  jumpTargetKey.value = item.rowKey
  jumpInput.value = item.resumeMs ? formatDurationMs(item.resumeMs) : ''
  jumpDialogVisible.value = true
}

async function confirmJump() {
  const item = items.value.find((entry) => entry.rowKey === jumpTargetKey.value)
  if (!item) {
    jumpDialogVisible.value = false
    return
  }

  const parsed = parseTimeInput(jumpInput.value)
  if (parsed === null) {
    showNotice('warning', '时间格式不正确，支持 90、01:30、01:02:03。')
    return
  }

  jumpDialogVisible.value = false
  await playVideo(item, { startMs: parsed })
}

async function changePreferredPlayer(playerId) {
  if (!playerId) {
    return
  }

  const target = playerOptions.value.find((item) => item.id === playerId)
  if (target?.disabled) {
    showNotice('warning', '已禁用的播放器不能设为默认。')
    return
  }

  playerLoading.value = true

  try {
    applySettingsView(await SavePreferredPlayer(playerId))
    showNotice('success', `默认播放器已切换为 ${selectedPlayer.value?.name || playerId}。`)
  } catch (error) {
    showError(error, '切换播放器失败')
  } finally {
    playerLoading.value = false
  }
}

async function saveShowTitleBadges(enabled) {
  try {
    applySettingsView(await SaveShowTitleBadgesEnabled(Boolean(enabled)))
    refreshFilePresentation()
  } catch (error) {
    showError(error, '保存徽章显示设置失败')
  }
}

async function saveSmallFileFilter(value) {
  const nextValue = Number(value || 0)
  smallFileFilterMB.value = nextValue

  try {
    applySettingsView(await SaveSmallFileFilterMB(nextValue))
  } catch (error) {
    smallFileFilterMB.value = Number(settings.value?.smallFileFilterMB || 0)
    showError(error, '保存小文件屏蔽规则失败')
  }
}

async function saveFileListDensity(value) {
  const nextValue = normalizeFileListDensity(value)

  try {
    applySettingsView(await SaveFileListDensity(nextValue))
  } catch (error) {
    showError(error, '保存文件列表密度失败')
  }
}

async function choosePlayerPath(playerId = selectedPlayer.value?.id || settings.value?.preferredPlayer || 'mpv') {
  playerLoading.value = true

  try {
    applySettingsView(await SelectPlayerPath(playerId))
    showNotice('success', '播放器路径已保存。')
  } catch (error) {
    showError(error, '设置播放器路径失败')
  } finally {
    playerLoading.value = false
  }
}

async function selectPlayerFromList(player) {
  if (!player || player.disabled) {
    return
  }
  await changePreferredPlayer(player.id)
}

async function togglePlayerDisabled(player) {
  if (!player?.id) {
    return
  }

  playerLoading.value = true

  try {
    const nextDisabled = !player.disabled
    applySettingsView(await SavePlayerDisabled(player.id, nextDisabled))
    showNotice('success', nextDisabled ? `${player.name} 已禁用。` : `${player.name} 已启用。`)
  } catch (error) {
    showError(error, player?.disabled ? '启用播放器失败' : '禁用播放器失败')
  } finally {
    playerLoading.value = false
  }
}

async function deletePlayer(player) {
  if (!player?.id) {
    return
  }

  playerLoading.value = true

  try {
    applySettingsView(await DeletePlayerPath(player.id))
    showNotice('success', `${player.name} 的已保存路径已删除。`)
  } catch (error) {
    showError(error, '删除播放器路径失败')
  } finally {
    playerLoading.value = false
  }
}

async function reloadCurrentDirectory() {
  if (!loggedIn.value) {
    showNotice('info', '请先登录 115 账号。')
    return
  }
  if (isGlobalSearchActive.value) {
    clearSearchDebounce()
    searchRequestId.value += 1
    await performGlobalSearch(searchQuery.value.trim(), searchRequestId.value, searchPage.value)
    return
  }
  await loadDirectory(currentDir.value, {
    fallbackName: breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name || '我的文件',
    fallbackPath: breadcrumbItems.value,
  }, directoryPage.value)
}

async function triggerSearchNow() {
  const keyword = searchQuery.value.trim()
  if (!keyword || !loggedIn.value) {
    return
  }
  clearSearchDebounce()
  searchRequestId.value += 1
  await performGlobalSearch(keyword, searchRequestId.value, searchPage.value)
}

function openSearchInput() {
  if (!loggedIn.value) {
    return
  }

  searchExpanded.value = true
  nextTick(() => {
    searchField.value?.focus?.()
  })
}

function closeSearchInput() {
  clearSearchDebounce()
  searchRequestId.value += 1
  searchQuery.value = ''
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  searchExpanded.value = false
}

function toggleSearchInput() {
  if (!loggedIn.value) {
    return
  }
  if (isSearchInputVisible.value) {
    closeSearchInput()
    return
  }
  openSearchInput()
}

function collapseSearchInput(force = false) {
  if (force || !searchQuery.value.trim()) {
    searchExpanded.value = false
  }
}

function handleSearchBlur() {
  window.setTimeout(() => {
    if (!searchQuery.value.trim()) {
      searchExpanded.value = false
    }
  }, 120)
}

function handleSearchClear() {
  closeSearchInput()
}

async function handlePageChange(page) {
  const nextPage = Math.max(1, Number(page || 1))
  if (isGlobalSearchActive.value) {
    if (nextPage === searchPage.value) {
      return
    }
    searchRequestId.value += 1
    await performGlobalSearch(searchQuery.value.trim(), searchRequestId.value, nextPage)
    return
  }

  if (nextPage === directoryPage.value) {
    return
  }
  await loadDirectory(currentDir.value, {
    fallbackName: breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name || '我的文件',
    fallbackPath: breadcrumbItems.value,
  }, nextPage)
}

async function handlePageSizeChange(value) {
  const nextSize = Number(value || 20)
  if (!nextSize || nextSize === pageSize.value) {
    return
  }

  pageSize.value = nextSize
  if (isGlobalSearchActive.value) {
    searchRequestId.value += 1
    await performGlobalSearch(searchQuery.value.trim(), searchRequestId.value, 1)
    return
  }

  await loadDirectory(currentDir.value, {
    fallbackName: breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name || '我的文件',
    fallbackPath: breadcrumbItems.value,
  }, 1)
}

function handleDownloadPageChange(page) {
  const nextPage = Math.max(1, Number(page || 1))
  if (nextPage === downloadPage.value) {
    return
  }
  downloadPage.value = Math.min(nextPage, downloadPageCount.value)
}

function handleDownloadPageSizeChange(value) {
  const nextSize = Number(value || 20)
  if (!nextSize || nextSize === downloadPageSize.value) {
    return
  }

  downloadPageSize.value = nextSize
  downloadPage.value = 1
}

async function openBreadcrumb(dirId) {
  if (!dirId || dirId === currentDir.value) {
    return
  }
  const fallbackPath = breadcrumbPathUntil(dirId)
  await loadDirectory(dirId, {
    fallbackName: fallbackPath[fallbackPath.length - 1]?.name || '我的文件',
    fallbackPath,
  }, 1)
}

async function refreshOfflineTasks() {
  if (!loggedIn.value) {
    return
  }

  downloadLoading.value = true

  try {
    const data = await ListOfflineTasks()
    downloadState.value = normalizeOfflineState(data)
    downloadPage.value = 1
  } catch (error) {
    showError(error, '读取云下载失败')
  } finally {
    downloadLoading.value = false
  }
}

function openDownloadDialog() {
  if (!loggedIn.value) {
    showNotice('info', '请先登录 115 账号。')
    return
  }

  const recentTarget = offlineRecentTargets.value[0]
  offlineTargetDir.value = recentTarget
    ? createDirectoryTarget(recentTarget.id, recentTarget.path)
    : createDirectoryTarget(currentDir.value || '0', breadcrumbItems.value)
  downloadDialog.value = true
}

function closeDownloadDialog() {
  downloadDialog.value = false
}

async function selectTorrentFile() {
  torrentSelecting.value = true

  try {
    const magnet = await SelectTorrentFileAsMagnet()
    const normalized = String(magnet || '').trim()
    if (!normalized) {
      return
    }

    downloadInput.value = downloadInput.value.trim()
      ? `${downloadInput.value.trim()}\n${normalized}`
      : normalized
    showNotice('success', 'BT 种子已转换为磁力链接并填入输入框。')
  } catch (error) {
    showError(error, '导入 BT 种子失败')
  } finally {
    torrentSelecting.value = false
  }
}

async function submitOfflineTasks() {
  const urls = normalizeMultilineInput(downloadInput.value)
  if (!urls.length) {
    showNotice('warning', '请至少输入一个下载链接。')
    return
  }

  downloadSubmitting.value = true

  try {
    const data = await AddOfflineTasks({
      urls,
      saveDirId: offlineTargetDir.value?.id || '0',
      saveDirPath: offlineTargetDir.value?.path || [rootBreadcrumb()],
    })

    downloadState.value = normalizeOfflineState(data)
    downloadPage.value = 1
    settings.value = {
      ...settings.value,
      offlineRecentTargets: rememberOfflineRecentTargets(
        settings.value?.offlineRecentTargets,
        offlineTargetDir.value,
      ),
    }
    downloadDialog.value = false
    downloadInput.value = ''
    activeSection.value = 'downloads'
    downloadFilter.value = 'active'
    showNotice('success', `已添加 ${urls.length} 个云下载任务。`)
  } catch (error) {
    showError(error, '添加云下载失败')
  } finally {
    downloadSubmitting.value = false
  }
}

async function deleteOfflineTask(task) {
  if (!task?.infoHash) {
    return
  }
  await deleteOfflineHashes([task.infoHash])
}

async function deleteFilteredOfflineTasks() {
  const hashes = filteredDownloadTasks.value.map((task) => task.infoHash).filter(Boolean)
  if (!hashes.length) {
    showNotice('info', '当前分类没有可删除的任务。')
    return
  }

  await deleteOfflineHashes(hashes)
}

async function deleteOfflineHashes(hashes) {
  downloadSubmitting.value = true

  try {
    const data = await DeleteOfflineTasks({
      hashes,
      deleteFiles: downloadDeleteFiles.value,
    })
    downloadState.value = normalizeOfflineState(data)
    downloadPage.value = 1
    showNotice('success', '离线下载任务已删除。')
  } catch (error) {
    showError(error, '删除云下载任务失败')
  } finally {
    downloadSubmitting.value = false
  }
}

async function copyOfflineURL(task) {
  if (!task?.url) {
    showNotice('info', '当前任务没有可复制的链接。')
    return
  }

  try {
    await Clipboard.SetText(task.url)
    showNotice('success', '任务链接已复制到剪贴板。')
  } catch (error) {
    showError(error, '复制任务链接失败')
  }
}

async function openOfflineDirectory(task) {
  if (!task?.dirId) {
    showNotice('warning', '当前任务还没有可打开的目录。')
    return
  }

  activeSection.value = 'files'
  await loadDirectory(task.dirId, {}, 1)
}

function openFolderPicker() {
  folderPickerDialog.value = true
  loadFolderPicker(offlineTargetDir.value?.id || currentDir.value || '0')
}

function closeFolderPicker() {
  folderPickerDialog.value = false
}

async function openFolderPickerBreadcrumb(dirId) {
  if (!dirId || dirId === folderPickerDirId.value) {
    return
  }
  await loadFolderPicker(dirId)
}

async function openFolderPickerDirectory(folder) {
  if (!folder?.fileId) {
    return
  }
  await loadFolderPicker(folder.fileId)
}

function chooseFolderPickerCurrent() {
  offlineTargetDir.value = createDirectoryTarget(folderPickerDirId.value, folderPickerPath.value)
  folderPickerDialog.value = false
}

function handleOfflineTargetSelect(value) {
  const option = offlineTargetSelectOptions.value.find((item) => item.value === value)
  if (!option) {
    return
  }

  if (option.isPicker) {
    openFolderPicker()
    return
  }

  if (option.target) {
    offlineTargetDir.value = createDirectoryTarget(option.target.id, option.target.path)
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

function showError(error, fallback = '操作失败') {
  const message = extractErrorMessage(error)
  showNotice('error', message || fallback, 5200)
}

function handleKeydown(event) {
  if (event.key === 'Escape' && folderPickerDialog.value) {
    event.preventDefault()
    closeFolderPicker()
    return
  }

  if (event.key === 'Escape' && downloadDialog.value) {
    event.preventDefault()
    closeDownloadDialog()
    return
  }

  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'f' && activeSection.value === 'files') {
    event.preventDefault()
    openSearchInput()
    return
  }

  if (event.key === 'Escape' && activeSection.value === 'files' && isSearchInputVisible.value) {
    closeSearchInput()
    return
  }

  if (event.key === 'Escape' && detailDrawer.value) {
    detailDrawer.value = false
  }
}

function createDefaultSettings() {
  return {
    preferredPlayer: 'mpv',
    players: [],
    configPath: '',
    showTitleBadges: true,
    smallFileFilterMB: 0,
    fileListDensity: 'default',
    offlineRecentTargets: [],
  }
}

function applySettingsView(view) {
  settings.value = normalizeSettingsView(view)
  smallFileFilterMB.value = Number(settings.value.smallFileFilterMB || 0)
}

function createEmptyOfflineState() {
  return {
    quota: 0,
    total: 0,
    activeCount: 0,
    failedCount: 0,
    completedCount: 0,
    tasks: [],
  }
}

function createDirectoryTarget(id, path) {
  const normalizedPath = Array.isArray(path) && path.length ? path.map((item) => ({
    id: item.id,
    name: item.name,
  })) : [rootBreadcrumb()]

  return {
    id: String(id || '0'),
    name: normalizedPath[normalizedPath.length - 1]?.name || '我的文件',
    path: normalizedPath,
  }
}

function normalizeSettingsView(view) {
  return {
    preferredPlayer: view?.preferredPlayer || 'mpv',
    players: Array.isArray(view?.players) ? view.players : [],
    configPath: view?.configPath || '',
    showTitleBadges: view?.showTitleBadges !== false,
    smallFileFilterMB: Number(view?.smallFileFilterMB || 0),
    fileListDensity: normalizeFileListDensity(view?.fileListDensity),
    offlineRecentTargets: normalizeOfflineRecentTargets(view?.offlineRecentTargets),
  }
}

function normalizeDirectoryTargetValue(id, path) {
  const normalizedId = String(id || '').trim()
  const rawPath = Array.isArray(path)
    ? path
      .map((item) => ({
        id: String(item?.id || '').trim(),
        name: String(item?.name || '').trim(),
      }))
      .filter((item) => item.id && item.name)
    : []

  if (normalizedId === '0' || (!normalizedId && !rawPath.length)) {
    return createDirectoryTarget('0', [rootBreadcrumb()])
  }

  if (!rawPath.length) {
    return null
  }

  const normalizedPath = rawPath[0]?.id === '0'
    ? rawPath
    : [rootBreadcrumb(), ...rawPath.filter((item) => item.id !== '0')]

  const finalId = normalizedId || normalizedPath[normalizedPath.length - 1]?.id || '0'
  if (finalId === '0') {
    return createDirectoryTarget('0', [rootBreadcrumb()])
  }

  const matchedIndex = normalizedPath.findIndex((item) => item.id === finalId)
  if (matchedIndex === -1) {
    return null
  }

  return createDirectoryTarget(finalId, normalizedPath.slice(0, matchedIndex + 1))
}

function normalizeOfflineRecentTargets(targets) {
  const normalized = []
  const seen = new Set()

  for (const entry of Array.isArray(targets) ? targets : []) {
    const target = normalizeDirectoryTargetValue(entry?.id, entry?.path)
    if (!target || seen.has(target.id)) {
      continue
    }
    seen.add(target.id)
    normalized.push(target)
    if (normalized.length === 3) {
      break
    }
  }

  return normalized
}

function rememberOfflineRecentTargets(targets, target) {
  const currentTarget = normalizeDirectoryTargetValue(target?.id, target?.path)
  if (!currentTarget) {
    return normalizeOfflineRecentTargets(targets)
  }

  return normalizeOfflineRecentTargets([currentTarget, ...(Array.isArray(targets) ? targets : [])])
}

function formatDirectoryTargetPath(path) {
  return (Array.isArray(path) ? path : []).map((item) => item.name).join(' / ') || '我的文件'
}

function createOfflineTargetOption(target) {
  return {
    value: `target:${target.id}`,
    title: formatDirectoryTargetPath(target.path),
    icon: 'mdi-folder-outline',
    target,
  }
}

function normalizeFileListDensity(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'compact' || normalized === 'comfortable') {
    return normalized
  }
  return 'default'
}

const videoExtensions = new Set(['.mp4', '.mkv', '.avi', '.mov', '.wmv', '.flv', '.m4v', '.rmvb', '.ts', '.webm'])
const audioExtensions = new Set(['.mp3', '.flac', '.m4a', '.aac', '.wav', '.ogg', '.opus', '.ape', '.dts'])
const subtitleExtensions = new Set(['.srt', '.ass', '.ssa', '.vtt', '.sub'])
const archiveExtensions = new Set(['.zip', '.rar', '.7z', '.tar', '.gz', '.iso'])

const fileBadgeDefinitions = [
  {
    pattern: /\b(2160P|1080P|720P|4K)\b/gi,
    normalize: (match) => match.toUpperCase(),
    describe: (label) => `${label} 分辨率`,
  },
  {
    pattern: /\b(WEB[- .]?DL|WEB[- .]?RIP|BLURAY|REMUX|BDRIP|HDRIP|HDTV|DVDRIP)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/[ .]/g, ''),
    describe: (label) => `${label} 片源类型`,
  },
  {
    pattern: /\b(HEVC|H\.?265|X265|AV1|X264|H\.?264|AVC)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/\./g, ''),
    describe: (label) => `${label} 视频编码`,
  },
  {
    pattern: /\b(AAC|ACC|AC3|EAC3|DTS(?:-HD)?|TRUEHD|FLAC|ATMOS)\b/gi,
    normalize: (match) => normalizeAudioBadge(match),
    describe: (label) => audioBadgeDescription(label),
  },
  {
    pattern: /\b(HDR10\+?|HDR|DV|DOLBY[ .-]?VISION)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/[ .]/g, ''),
    describe: (label) => `${label} 高动态范围信息`,
  },
  {
    pattern: /\b(CHS|CHT|ENG|JPN|KOR|MANDARIN|CANTONESE)\b/gi,
    normalize: (match) => match.toUpperCase(),
    describe: (label) => subtitleBadgeDescription(label),
  },
  {
    pattern: /简繁英字幕|简繁字幕|中英字幕|中日字幕|中韩字幕|国粤双语|粤国双语|双音轨|多音轨|简体|繁体|中字|双语|内封|外挂|国语|粤语|英语|日语|韩语|普通话/g,
    normalize: (match) => match,
    describe: (label) => textBadgeDescription(label),
  },
  {
    pattern: /杜比视界|杜比全景声/g,
    normalize: (match) => match,
    describe: (label) => textBadgeDescription(label),
  },
]

function normalizeItem(item) {
  const resumeMs = Number(item?.resumeMs || 0)
  const durationSec = Number(item?.durationSec || 0)
  const originalName = String(item?.originalName || item?.name || '')
  const presentation = buildFilePresentation(originalName, Boolean(item?.isDirectory))
  const showTitleBadges = settings.value?.showTitleBadges !== false
  const resumeBadge = buildResumeBadge(resumeMs, durationSec)
  const durationBadge = buildDurationBadge(durationSec, resumeMs)
  const badgeList = [
    ...(resumeBadge ? [resumeBadge] : []),
    ...(durationBadge ? [durationBadge] : []),
    ...(showTitleBadges ? presentation.badges : []),
  ]
  const visibleBadges = badgeList.slice(0, 4)
  const hiddenBadges = badgeList.slice(4)
  return {
    ...item,
    originalName,
    rowKey: item?.fileId || item?.pickCode || item?.name,
    icon: presentation.icon,
    kindLabel: presentation.kindLabel,
    mediaKind: presentation.mediaKind,
    displayName: presentation.title,
    displayNameMain: presentation.titleMain,
    displayNameExtension: presentation.titleExtension,
    badges: badgeList,
    metaBadges: presentation.badges,
    visibleBadges,
    hiddenBadgeCount: hiddenBadges.length,
    hiddenBadgeSummary: hiddenBadges.map((badge) => `${badge.label}：${badge.description}`).join(' · '),
    iconColor: colorForMediaKind(presentation.mediaKind),
    sizeText: item?.isDirectory ? '--' : formatSize(item?.size),
    updatedText: formatDateTime(item?.updatedAt),
    durationSec,
    durationText: durationSec > 0 ? formatDurationMs(durationSec * 1000) : '',
    resumeMs,
    resumeText: resumeMs > 0 ? formatResumeProgressText(resumeMs, durationSec) : '',
  }
}

function subtitleBadgeDescription(label) {
  const normalized = String(label || '').toUpperCase()
  if (normalized === 'CHS' || label === '简体') return '简体中文字幕'
  if (normalized === 'CHT' || label === '繁体') return '繁体中文字幕'
  if (normalized === 'ENG') return '英文字幕或英文音轨'
  if (normalized === 'JPN') return '日文字幕或日文音轨'
  if (normalized === 'KOR') return '韩文字幕或韩文音轨'
  if (normalized === 'MANDARIN' || label === '国语' || label === '普通话') return '普通话音轨'
  if (normalized === 'CANTONESE' || label === '粤语') return '粤语音轨'
  if (label === '英语') return '英语音轨'
  if (label === '日语') return '日语音轨'
  if (label === '韩语') return '韩语音轨'
  if (label === '中字') return '带中文字幕'
  if (label === '中英字幕') return '中英双语字幕'
  if (label === '中日字幕') return '中日双语字幕'
  if (label === '中韩字幕') return '中韩双语字幕'
  if (label === '简繁字幕') return '简繁中文字幕'
  if (label === '简繁英字幕') return '简繁英多语字幕'
  if (label === '双语') return '双语字幕或双语音轨'
  if (label === '内封') return '字幕内封在媒体文件中'
  if (label === '外挂') return '需要外挂字幕文件'
  return `${label} 字幕或语言信息`
}

function normalizeAudioBadge(label) {
  const normalized = String(label || '').toUpperCase().replace(/\s+/g, '')
  if (normalized === 'ACC') return 'AAC'
  return normalized
}

function audioBadgeDescription(label) {
  if (label === 'ATMOS') return '杜比全景声音频'
  if (label === 'TRUEHD') return 'TrueHD 无损音频'
  if (label === 'DTS-HD') return 'DTS-HD 音频'
  if (label === 'FLAC') return 'FLAC 无损音频'
  return `${label} 音频格式`
}

function textBadgeDescription(label) {
  if (label === '双音轨') return '双音轨版本'
  if (label === '多音轨') return '多音轨版本'
  if (label === '国粤双语' || label === '粤国双语') return '普通话与粤语双音轨'
  if (label === '杜比视界') return '杜比视界版本'
  if (label === '杜比全景声') return '杜比全景声音频'
  return subtitleBadgeDescription(label)
}

function formatResumeProgressText(resumeMs, durationSec = 0) {
  if (!resumeMs) {
    return ''
  }

  const current = formatDurationMs(resumeMs)
  const total = Number(durationSec || 0) > 0 ? formatDurationMs(Number(durationSec) * 1000) : ''
  return total ? `${current}/${total}` : current
}

function buildResumeBadge(resumeMs, durationSec = 0) {
  const progressText = formatResumeProgressText(resumeMs, durationSec)
  if (!progressText) {
    return null
  }

  return {
    label: `上次播放：${progressText}`,
    description: `上次退出位置 ${progressText}`,
    color: 'success',
  }
}

function buildDurationBadge(durationSec = 0, resumeMs = 0) {
  const normalizedDuration = Number(durationSec || 0)
  if (normalizedDuration <= 0 || Number(resumeMs || 0) > 0) {
    return null
  }

  const durationText = formatDurationMs(normalizedDuration * 1000)
  if (!durationText) {
    return null
  }

  return {
    label: `时长：${durationText}`,
    description: `视频总时长 ${durationText}`,
    color: 'info',
  }
}

function buildFilePresentation(name, isDirectory) {
  const resolvedName = String(name || '').trim()
  const extension = extractExtension(resolvedName)
  const badgeSource = isDirectory ? resolvedName : stripExtension(resolvedName, extension)
  const badges = extractFileBadges(badgeSource)
  const mediaKind = detectMediaKind(isDirectory, extension)
  const displayTitle = resolvedName || (isDirectory ? '未命名文件夹' : '未命名文件')

  return {
    title: displayTitle,
    titleMain: displayTitle,
    titleExtension: '',
    badges,
    mediaKind,
    icon: iconForMediaKind(mediaKind),
    kindLabel: labelForMediaKind(mediaKind),
  }
}

function extractFileBadges(name) {
  const badges = []
  const seen = new Set()

  for (const definition of fileBadgeDefinitions) {
    for (const match of String(name || '').matchAll(definition.pattern)) {
      const rawLabel = match?.[0]
      const label = String(definition.normalize ? definition.normalize(rawLabel) : rawLabel || '').trim()
      if (!label || seen.has(label)) {
        continue
      }

      seen.add(label)
      badges.push({
        label,
        description: definition.describe ? definition.describe(label) : label,
      })
    }
  }

  return badges
}

function stripBadgeTokens(name) {
  let working = String(name || '')

  for (const definition of fileBadgeDefinitions) {
    working = working.replace(definition.pattern, ' ')
  }

  return working
}

function cleanupDisplayTitle(name) {
  return String(name || '')
    .replace(/[._]+/g, ' ')
    .replace(/[【】\[\]（）()]/g, ' ')
    .replace(/\s*[+&/]+\s*/g, ' ')
    .replace(/\s*[-|]+\s*/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .replace(/\(\s*\)/g, ' ')
    .replace(/\[\s*]/g, ' ')
    .trim()
}

function normalizeAvatarUrl(value) {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return ''
  }
  if (normalized.startsWith('data:')) {
    return normalized
  }
  if (normalized.startsWith('//')) {
    return `https:${normalized}`
  }
  if (/^https?:\/\//i.test(normalized)) {
    return normalized
  }
  if (normalized.startsWith('/')) {
    return `https://115.com${normalized}`
  }
  return `https://${normalized.replace(/^\/+/, '')}`
}

function buildAccountAvatarUrl(value) {
  const normalized = normalizeAvatarUrl(value)
  if (!normalized) {
    return ''
  }
  if (!proxyBase.value || normalized.startsWith('data:')) {
    return normalized
  }
  return `${proxyBase.value}/avatar?url=${encodeURIComponent(normalized)}`
}

function detectMediaKind(isDirectory, extension) {
  if (isDirectory) return 'folder'
  if (videoExtensions.has(extension)) return 'video'
  if (audioExtensions.has(extension)) return 'audio'
  if (subtitleExtensions.has(extension)) return 'subtitle'
  if (archiveExtensions.has(extension)) return 'archive'
  return 'file'
}

function iconForMediaKind(kind) {
  if (kind === 'folder') return 'mdi-folder-outline'
  if (kind === 'video') return 'mdi-movie-open-outline'
  if (kind === 'audio') return 'mdi-music-circle-outline'
  if (kind === 'subtitle') return 'mdi-subtitles-outline'
  if (kind === 'archive') return 'mdi-package-variant-closed'
  return 'mdi-file-outline'
}

function labelForMediaKind(kind) {
  if (kind === 'folder') return '文件夹'
  if (kind === 'video') return '视频'
  if (kind === 'audio') return '音频'
  if (kind === 'subtitle') return '字幕'
  if (kind === 'archive') return '压缩包'
  return '文件'
}

function colorForMediaKind(kind) {
  if (kind === 'folder') return 'warning'
  if (kind === 'video') return 'primary'
  if (kind === 'audio') return 'deep-purple'
  if (kind === 'subtitle') return 'teal'
  if (kind === 'archive') return 'brown'
  return 'grey'
}

function shouldKeepExtension(kind, extension) {
  if (!extension) {
    return false
  }
  return kind !== 'folder'
}

function extractExtension(name) {
  const value = String(name || '')
  const index = value.lastIndexOf('.')
  if (index <= 0) {
    return ''
  }
  return value.slice(index).toLowerCase()
}

function stripExtension(name, extension) {
  if (!extension) {
    return String(name || '')
  }
  return String(name || '').slice(0, -extension.length)
}

function handleWindowResize() {
  queueBadgeVisibilityRefresh()
}

function setBadgeRowRef(rowKey, element) {
  const key = String(rowKey || '')
  if (!key) {
    return
  }

  if (element) {
    badgeRowElements.set(key, element)
  } else {
    badgeRowElements.delete(key)
  }
  queueBadgeVisibilityRefresh()
}

function queueBadgeVisibilityRefresh() {
  if (badgeRefreshFrame) {
    window.cancelAnimationFrame(badgeRefreshFrame)
  }
  badgeRefreshFrame = window.requestAnimationFrame(() => {
    badgeRefreshFrame = 0
    recalculateBadgeVisibility()
  })
}

function recalculateBadgeVisibility() {
  const nextCounts = {}

  for (const item of filteredItems.value) {
    const key = String(item?.rowKey || '')
    const badges = badgeListFor(item)
    if (!key || !badges.length) {
      continue
    }

    const rowElement = badgeRowElements.get(key)
    if (!rowElement) {
      nextCounts[key] = defaultVisibleBadgeCount(badges)
      continue
    }

    const availableWidth = Number(rowElement.clientWidth || 0)
    if (availableWidth <= 0) {
      nextCounts[key] = defaultVisibleBadgeCount(badges)
      continue
    }

    const widths = badges.map((badge) => estimateBadgeWidth(badge?.label))
    const gap = 4
    const totalWidth = widths.reduce((sum, width, index) => sum + width + (index > 0 ? gap : 0), 0)
    if (totalWidth <= availableWidth) {
      nextCounts[key] = badges.length
      continue
    }

    let consumed = 0
    let visibleCount = 0

    for (let index = 0; index < widths.length; index += 1) {
      const remaining = widths.length - index - 1
      const nextWidth = consumed + (visibleCount > 0 ? gap : 0) + widths[index]
      const reserveWidth = remaining > 0 ? gap + estimateBadgeWidth(`+${remaining}`) : 0

      if (nextWidth + reserveWidth <= availableWidth) {
        consumed = nextWidth
        visibleCount = index + 1
        continue
      }
      break
    }

    nextCounts[key] = Math.max(1, visibleCount)
  }

  badgeVisibleCounts.value = nextCounts
}

function defaultVisibleBadgeCount(badges) {
  return Math.min(Array.isArray(badges) ? badges.length : 0, 4)
}

function badgeListFor(item) {
  return Array.isArray(item?.badges) ? item.badges : []
}

function visibleBadgesFor(item) {
  const badges = badgeListFor(item)
  const key = String(item?.rowKey || '')
  const count = Object.prototype.hasOwnProperty.call(badgeVisibleCounts.value, key)
    ? badgeVisibleCounts.value[key]
    : defaultVisibleBadgeCount(badges)
  return badges.slice(0, Math.max(0, count))
}

function hiddenBadgesFor(item) {
  const badges = badgeListFor(item)
  return badges.slice(visibleBadgesFor(item).length)
}

function hiddenBadgeCountFor(item) {
  return hiddenBadgesFor(item).length
}

function hiddenBadgeSummaryFor(item) {
  return hiddenBadgesFor(item)
    .map((badge) => `${badge.label}：${badge.description}`)
    .join(' · ')
}

function estimateBadgeWidth(label) {
  const text = String(label || '')
  if (!badgeMeasureCanvas) {
    badgeMeasureCanvas = document.createElement('canvas')
  }

  const context = badgeMeasureCanvas.getContext('2d')
  if (!context) {
    return Math.max(28, (text.length * 7) + 24)
  }

  context.font = '500 11px Roboto, Arial, sans-serif'
  return Math.ceil(context.measureText(text).width) + 24
}

function refreshFilePresentation() {
  const selectedKey = selectedItem.value?.rowKey
  items.value = items.value.map((item) => normalizeItem({
    ...item,
    name: item.originalName || item.name,
  }))

  if (selectedKey) {
    selectedItem.value = items.value.find((item) => item.rowKey === selectedKey) ?? selectedItem.value
  }

  nextTick(() => {
    queueBadgeVisibilityRefresh()
  })
}

function normalizeOfflineState(data) {
  if (!data) {
    return createEmptyOfflineState()
  }

  return {
    quota: Number(data.quota || 0),
    total: Number(data.total || 0),
    activeCount: Number(data.activeCount || 0),
    failedCount: Number(data.failedCount || 0),
    completedCount: Number(data.completedCount || 0),
    tasks: Array.isArray(data.tasks) ? data.tasks.map((task) => ({
      ...task,
      percent: Number(task.percent || 0),
      percentText: task.percentText || '0%',
      speedText: task.speedText || '--',
      leftTimeText: task.leftTimeText || '--',
    })) : [],
  }
}

function offlineTaskColor(task) {
  if (task?.statusGroup === 'completed') return 'success'
  if (task?.statusGroup === 'failed') return 'error'
  return 'primary'
}

function offlineTaskIcon(task) {
  if (task?.statusGroup === 'completed') return 'mdi-check-circle-outline'
  if (task?.statusGroup === 'failed') return 'mdi-alert-circle-outline'
  return 'mdi-download-outline'
}

function offlineTaskSubtitle(task) {
  if (!task) {
    return '--'
  }
  if (task.statusGroup === 'active') {
    return `${task.status} · ${task.speedText} · 剩余 ${task.leftTimeText}`
  }
  if (task.statusGroup === 'completed') {
    return `${task.status} · 已完成`
  }
  return `${task.status} · ${task.percentText}`
}

function offlineTaskMetaText(task) {
  if (!task) {
    return '--'
  }

  return [formatSize(task.size), task.addTime]
    .map((item) => String(item || '').trim())
    .filter((item) => item && item !== '--')
    .join(' · ') || '--'
}

function offlineTaskProgressText(task) {
  if (!task) {
    return '--'
  }
  if (task.statusGroup === 'active') {
    return `${task.status} · ${task.speedText} · 剩余 ${task.leftTimeText}`
  }
  if (task.statusGroup === 'completed') {
    return `${task.status} · 已完成`
  }
  return `${task.status} · ${task.percentText}`
}

function capabilityTags(player) {
  if (!player) {
    return []
  }

  return [
    {
      label: player.supportsStartPosition ? '支持跳转播放' : '不支持跳转播放',
      color: player.supportsStartPosition ? 'primary' : 'warning',
    },
    {
      label: player.supportsSubtitle ? '支持外挂字幕' : '不支持外挂字幕',
      color: player.supportsSubtitle ? 'primary' : 'warning',
    },
    {
      label: player.supportsManagedResume ? '支持托管续播' : '依赖播放器自身续播',
      color: player.supportsManagedResume ? 'success' : 'info',
    },
  ]
}

function playerStatusText(player) {
  if (!player) {
    return '--'
  }
  if (player.disabled && player.path) {
    return player.path
  }
  if (player.available) {
    return player.path || '已就绪'
  }
  return '未检测到可执行文件'
}

function compareItems(left, right, mode) {
  const leftName = left.displayName || left.name
  const rightName = right.displayName || right.name

  if (mode === 'folders') {
    if (left.isDirectory !== right.isDirectory) {
      return left.isDirectory ? -1 : 1
    }
    return compareText(leftName, rightName)
  }

  if (mode === 'name') {
    return compareText(leftName, rightName)
  }

  if (mode === 'updated') {
    return compareTimestamp(right.updatedAt, left.updatedAt) || compareText(leftName, rightName)
  }

  if (mode === 'size') {
    if ((right.size || 0) !== (left.size || 0)) {
      return (right.size || 0) - (left.size || 0)
    }
    return compareText(leftName, rightName)
  }

  if (mode === 'resume') {
    if ((right.resumeMs || 0) !== (left.resumeMs || 0)) {
      return (right.resumeMs || 0) - (left.resumeMs || 0)
    }
    if (left.isDirectory !== right.isDirectory) {
      return left.isDirectory ? -1 : 1
    }
    return compareText(leftName, rightName)
  }

  return compareText(leftName, rightName)
}

function compareText(left, right) {
  return String(left || '').localeCompare(String(right || ''), 'zh-Hans-CN', { sensitivity: 'base' })
}

function compareTimestamp(left, right) {
  return Date.parse(left || '') - Date.parse(right || '')
}

function normalizeMultilineInput(value) {
  return String(value || '')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseTimeInput(value) {
  const text = String(value || '').trim()
  if (!text) {
    return null
  }

  if (/^\d+$/.test(text)) {
    return Number(text) * 1000
  }

  const parts = text.split(':').map((item) => item.trim())
  if (parts.some((item) => !/^\d+$/.test(item))) {
    return null
  }

  if (parts.length === 2) {
    const minutes = Number(parts[0])
    const seconds = Number(parts[1])
    return ((minutes * 60) + seconds) * 1000
  }

  if (parts.length === 3) {
    const hours = Number(parts[0])
    const minutes = Number(parts[1])
    const seconds = Number(parts[2])
    return (((hours * 60 * 60) + (minutes * 60) + seconds) * 1000)
  }

  return null
}

function formatDurationMs(ms) {
  const totalSeconds = Math.max(0, Math.floor(Number(ms || 0) / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${padTwo(hours)}:${padTwo(minutes)}:${padTwo(seconds)}`
  }
  return `${padTwo(minutes)}:${padTwo(seconds)}`
}

function formatDateTime(value) {
  if (!value) {
    return '--'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '--'
  }

  const year = date.getFullYear()
  const month = padTwo(date.getMonth() + 1)
  const day = padTwo(date.getDate())
  const hours = padTwo(date.getHours())
  const minutes = padTwo(date.getMinutes())
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

function formatDateOnly(value) {
  if (!value) {
    return '--'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '--'
  }

  const year = date.getFullYear()
  const month = padTwo(date.getMonth() + 1)
  const day = padTwo(date.getDate())
  return `${year}-${month}-${day}`
}

function formatSize(value) {
  const size = Number(value || 0)
  if (size <= 0) {
    return '--'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let next = size
  while (next >= 1024 && index < units.length - 1) {
    next /= 1024
    index += 1
  }

  return `${next >= 10 || index === 0 ? next.toFixed(0) : next.toFixed(1)} ${units[index]}`
}

function formatStorageSize(value) {
  const size = Number(value || 0)
  if (size <= 0) {
    return '--'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let index = 0
  let next = size
  while (next >= 1024 && index < units.length - 1) {
    next /= 1024
    index += 1
  }

  const fractionDigits = index >= 3 ? 2 : index >= 1 ? 1 : 0
  return `${next.toFixed(fractionDigits)} ${units[index]}`
}

function padTwo(value) {
  return String(value).padStart(2, '0')
}

function basename(path) {
  return String(path || '').split(/[\\/]/).filter(Boolean).pop() || ''
}

function looksLikeHTMLResponse(message) {
  const normalized = String(message || '').trim().toLowerCase()
  return normalized.startsWith('<!doctype') ||
    normalized.startsWith('<html') ||
    normalized.includes('<meta charset=') ||
    normalized.includes('<body') ||
    normalized.includes('block_url_tips') ||
    normalized.includes('traceid') ||
    normalized.includes('unexpected error')
}

function sanitizeErrorMessage(message) {
  const normalized = String(message || '').trim()
  if (!normalized) {
    return ''
  }

  if (looksLikeHTMLResponse(normalized) || normalized.includes("invalid character '<'")) {
    return '115 云下载接口暂时返回了异常页面，请稍后手动刷新重试。登录状态仍然有效。'
  }

  return normalized
}

function extractErrorMessage(error) {
  if (!error) {
    return ''
  }
  if (typeof error === 'string') {
    return sanitizeErrorMessage(error)
  }
  if (typeof error?.message === 'string') {
    return sanitizeErrorMessage(error.message)
  }
  return sanitizeErrorMessage(String(error))
}

function isAppNotReadyError(error) {
  return extractErrorMessage(error).toLowerCase().includes('app not ready')
}

function sleep(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}
</script>

<template>
  <v-app>
    <v-layout class="app-shell">
      <v-navigation-drawer permanent width="220" class="sidebar">
        <div class="sidebar-layout">
          <v-list nav density="compact" class="sidebar-nav pa-2 pt-3">
            <v-list-item
              v-for="item in navigationItems"
              :key="item.value"
              rounded="lg"
              :active="activeSection === item.value"
              :title="item.label"
              :prepend-icon="item.icon"
              @click="activeSection = item.value"
            />
          </v-list>

          <div class="sidebar-footer pa-2 pt-0">
            <v-divider class="mb-2" />

            <v-menu v-if="loggedIn" location="top start">
              <template #activator="{ props }">
                <v-card
                  class="account-panel"
                  rounded="lg"
                  variant="flat"
                  role="button"
                  tabindex="0"
                  :aria-controls="props['aria-controls']"
                  :aria-expanded="props['aria-expanded']"
                  :aria-haspopup="props['aria-haspopup']"
                  @click="props.onClick"
                >
                  <v-card-item class="account-panel-head pt-2 pb-1">
                    <template #prepend>
                      <div v-if="accountAvatarUrl" class="account-avatar-image-shell account-avatar-image-shell--large" aria-hidden="true">
                        <img class="account-avatar-image" :src="accountAvatarUrl" alt="" />
                      </div>
                      <v-avatar
                        v-else
                        size="42"
                        class="account-avatar"
                        color="primary"
                        variant="tonal"
                      >
                        <v-icon>mdi-account-circle-outline</v-icon>
                      </v-avatar>
                    </template>

                    <v-card-title class="account-panel-title">
                      <div class="account-panel-title-stack">
                        <div class="account-panel-name text-truncate">{{ accountDisplayName }}</div>
                        <div class="account-panel-meta" v-if="loggedIn && accountVipLevelText !== '--'">
                          <span
                            v-if="user?.isVip"
                            class="account-vip-inline-pill"
                          >
                            VIP
                          </span>
                          <span
                            v-else
                            class="account-vip-inline-text"
                          >
                            {{ accountVipLevelText }}
                          </span>
                          <span
                            v-if="user?.isVip && accountVipExpireText !== '--' && accountVipExpireText !== '非 VIP'"
                            class="account-vip-inline-expire-text"
                          >
                            {{ accountVipExpireText }}
                          </span>
                        </div>
                      </div>
                    </v-card-title>
                  </v-card-item>

                  <v-card-text class="account-panel-body">
                    <div class="account-summary-row">
                      <div class="account-summary-main">
                        <div class="account-summary-label">空间占用</div>
                        <div class="account-summary-value">{{ accountSpaceUsageText }}</div>
                      </div>
                      <div class="account-summary-side">{{ accountSpacePercentText }}</div>
                    </div>

                    <v-progress-linear
                      class="account-space-bar"
                      :model-value="accountSpacePercent"
                      color="primary"
                      rounded
                      height="8"
                    />

                  </v-card-text>
                </v-card>
              </template>

              <v-list density="compact" min-width="180">
                <v-list-item
                  prepend-icon="mdi-qrcode"
                  title="重新扫码登录"
                  @click="openLoginDialog('qr')"
                />
                <v-list-item
                  prepend-icon="mdi-cookie-outline"
                  title="Cookie 登录"
                  @click="openLoginDialog('cookie')"
                />
                <v-divider class="my-1" />
                <v-list-item
                  prepend-icon="mdi-logout"
                  title="退出登录"
                  :disabled="actionLoading"
                  @click="handleLogout"
                />
              </v-list>
            </v-menu>

            <v-card
              v-else
              class="account-panel account-panel--guest"
              rounded="lg"
              variant="flat"
              @click="openLoginDialog('qr')"
            >
              <v-card-item class="account-panel-head account-panel-head--guest">
                <template #prepend>
                  <v-avatar size="42" color="primary" variant="tonal">
                    <v-icon>mdi-account-outline</v-icon>
                  </v-avatar>
                </template>

                <v-card-title class="account-panel-title">
                  未登录
                </v-card-title>
                <v-card-subtitle class="account-panel-subtitle">
                  点击登录 115
                </v-card-subtitle>

                <template #append>
                  <v-icon size="18">mdi-login</v-icon>
                </template>
              </v-card-item>
            </v-card>
          </div>
        </div>
      </v-navigation-drawer>

      <v-main class="app-main">
        <div class="workspace">
          <template v-if="booting">
            <v-card class="state-shell" variant="outlined">
              <v-progress-circular indeterminate color="primary" size="42" />
              <div class="text-subtitle-1 mt-4">正在初始化 115 会话</div>
              <div class="text-body-2 text-medium-emphasis">启动时会自动恢复本地 SQLite 中保存的登录状态。</div>
            </v-card>
          </template>

          <template v-else>
            <section v-if="activeSection === 'files'" class="page-section">
              <v-card class="section-card d-flex flex-column">
                <v-toolbar density="compact" flat class="page-toolbar px-2">
                  <div class="breadcrumb-strip">
                    <div class="path-bar">
                      <div v-if="isGlobalSearchActive" class="search-path-indicator">
                        <v-icon size="16" color="medium-emphasis">mdi-magnify</v-icon>
                        <span class="text-truncate">{{ searchSummaryText }}</span>
                      </div>

                      <v-breadcrumbs
                        v-else
                        class="file-breadcrumbs pa-0"
                        :items="breadcrumbDisplayItems"
                        divider="›"
                      >
                        <template #prepend>
                          <v-icon size="16" color="medium-emphasis">mdi-folder-outline</v-icon>
                        </template>

                        <template #title="{ item }">
                          <button
                            type="button"
                            class="file-breadcrumb-link"
                            :class="{ 'file-breadcrumb-link--disabled': item.disabled }"
                            :disabled="item.disabled"
                            @click="openBreadcrumb(item.id)"
                          >
                            {{ item.title }}
                          </button>
                        </template>
                      </v-breadcrumbs>

                      <v-btn
                        icon="mdi-refresh"
                        variant="text"
                        size="x-small"
                        class="path-refresh"
                        :disabled="!loggedIn"
                        :loading="isGlobalSearchActive ? searchLoading : directoryLoading"
                        @click="reloadCurrentDirectory"
                      />
                    </div>
                  </div>

                  <div class="search-slot">
                    <div
                      class="search-shell"
                      :class="{ 'search-shell--expanded': isSearchInputVisible }"
                    >
                      <v-btn
                        :icon="isSearchInputVisible ? 'mdi-close' : 'mdi-magnify'"
                        size="small"
                        variant="text"
                        :disabled="!loggedIn"
                        class="search-trigger"
                        @click="toggleSearchInput"
                      />

                      <div class="search-field-wrap">
                        <v-text-field
                          ref="searchField"
                          v-model="searchQuery"
                          class="compact-search"
                          clearable
                          :disabled="!loggedIn"
                          density="compact"
                          hide-details
                          variant="solo-filled"
                          flat
                          placeholder="全盘搜索"
                          prepend-inner-icon="mdi-magnify"
                          @blur="handleSearchBlur"
                          @click:clear="handleSearchClear"
                          @keydown.enter.prevent="triggerSearchNow"
                        />
                      </div>
                    </div>
                  </div>

                  <v-menu v-model="filterMenuOpen" location="bottom end" :close-on-content-click="false">
                    <template #activator="{ props }">
                      <v-btn
                        class="filter-trigger"
                        :class="{ 'filter-trigger--active': filterMenuOpen }"
                        :aria-controls="props['aria-controls']"
                        :aria-expanded="props['aria-expanded']"
                        :aria-haspopup="props['aria-haspopup']"
                        icon="mdi-tune-variant"
                        size="small"
                        variant="text"
                        @click="props.onClick"
                      />
                    </template>

                    <v-card class="filter-menu" min-width="268">
                      <v-card-text class="filter-menu-body">
                        <v-select
                          v-model="typeFilter"
                          class="filter-select"
                          density="compact"
                          hide-details
                          item-title="label"
                          item-value="value"
                          :items="typeOptions"
                          label="类型"
                          variant="outlined"
                        />

                        <v-select
                          v-model="sortMode"
                          class="filter-select"
                          density="compact"
                          hide-details
                          item-title="label"
                          item-value="value"
                          :items="sortOptions"
                          label="排序"
                          variant="outlined"
                        />

                        <v-select
                          :model-value="smallFileFilterMB"
                          class="filter-select"
                          density="compact"
                          hide-details
                          item-title="label"
                          item-value="value"
                          :items="smallFileFilterOptions"
                          label="小文件屏蔽规则"
                          variant="outlined"
                          @update:model-value="saveSmallFileFilter"
                        />

                        <v-select
                          :model-value="settings.fileListDensity"
                          class="filter-select"
                          density="compact"
                          hide-details
                          item-title="label"
                          item-value="value"
                          :items="fileListDensityOptions"
                          label="文件列表密度"
                          variant="outlined"
                          @update:model-value="saveFileListDensity"
                        />

                        <v-divider class="filter-divider" />

                        <v-checkbox
                          :model-value="settings.showTitleBadges"
                          class="filter-checkbox"
                          color="primary"
                          density="compact"
                          hide-details
                          label="显示徽章信息"
                          @update:model-value="saveShowTitleBadges"
                        />

                      </v-card-text>
                    </v-card>
                  </v-menu>
                </v-toolbar>

                <v-progress-linear
                  :active="directoryLoading || actionLoading || searchLoading"
                  :indeterminate="directoryLoading || actionLoading || searchLoading"
                  height="2"
                />

                <template v-if="!loggedIn">
                  <div class="state-shell">
                    <v-icon size="52" color="primary">mdi-folder-search-outline</v-icon>
                    <div class="text-subtitle-1">登录后即可浏览 115 文件</div>
                    <div class="text-body-2 text-medium-emphasis">
                      支持扫码登录和 Cookie 登录，登录状态会写入应用目录下的 SQLite 文件。
                    </div>
                    <div class="d-flex ga-2 flex-wrap">
                      <v-btn color="primary" prepend-icon="mdi-qrcode" @click="openLoginDialog('qr')">
                        扫码登录
                      </v-btn>
                      <v-btn variant="tonal" prepend-icon="mdi-cookie-outline" @click="openLoginDialog('cookie')">
                        Cookie 登录
                      </v-btn>
                    </div>
                  </div>
                </template>

                <template v-else>
                  <div class="table-scroll files-scroll" :class="fileListDensityClass">
                    <table class="files-table">
                      <thead>
                        <tr>
                          <th class="name-column">名称</th>
                          <th class="size-column text-no-wrap">大小</th>
                          <th class="time-column text-no-wrap">更新时间</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-if="!filteredItems.length">
                          <td colspan="3" class="empty-row">
                            {{ fileEmptyText }}
                          </td>
                        </tr>

                        <tr
                          v-for="item in filteredItems"
                          :key="item.rowKey"
                          class="file-row"
                          :class="{ 'selected-row': selectedItem?.rowKey === item.rowKey }"
                          @click="openDetails(item)"
                          @dblclick="handlePrimaryAction(item)"
                        >
                          <td class="name-column">
                            <div class="name-cell">
                              <v-avatar
                                :size="fileListAvatarSize"
                                class="name-avatar"
                                variant="tonal"
                                :color="item.iconColor"
                              >
                                <v-icon :size="fileListIconSize" :color="item.iconColor">{{ item.icon }}</v-icon>
                              </v-avatar>

                              <div class="name-text">
                                <div class="file-title">
                                  <span class="file-title-main text-truncate">{{ item.displayName || item.name }}</span>
                                </div>
                                <div class="file-meta-row">
                                  <span class="file-kind-label">{{ item.kindLabel }}</span>

                                  <div
                                    v-if="badgeListFor(item).length"
                                    class="file-badge-row"
                                    :ref="(el) => setBadgeRowRef(item.rowKey, el)"
                                  >
                                    <v-tooltip
                                      v-for="badge in visibleBadgesFor(item)"
                                      :key="`${item.rowKey}-${badge.label}`"
                                      location="bottom"
                                    >
                                      <template #activator="{ props }">
                                        <v-chip
                                          v-bind="props"
                                          size="x-small"
                                          variant="tonal"
                                          :color="badge.color || 'primary'"
                                          class="file-badge"
                                        >
                                          {{ badge.label }}
                                        </v-chip>
                                      </template>
                                      <span>{{ badge.description }}</span>
                                    </v-tooltip>

                                    <v-tooltip v-if="hiddenBadgeCountFor(item) > 0" location="bottom">
                                      <template #activator="{ props }">
                                        <v-chip
                                          v-bind="props"
                                          size="x-small"
                                          variant="tonal"
                                          color="default"
                                          class="file-badge"
                                        >
                                          +{{ hiddenBadgeCountFor(item) }}
                                        </v-chip>
                                      </template>
                                      <span>{{ hiddenBadgeSummaryFor(item) }}</span>
                                    </v-tooltip>
                                  </div>
                                </div>
                              </div>
                            </div>
                          </td>

                          <td class="size-column text-no-wrap text-caption">
                            {{ item.sizeText }}
                          </td>

                          <td class="time-column text-no-wrap text-caption">
                            {{ item.updatedText }}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  <div v-if="showPagination" class="pagination-bar">
                    <div class="pagination-summary text-caption text-medium-emphasis">
                      {{ paginationSummaryText }}
                    </div>

                    <div class="pagination-controls">
                      <v-select
                        :model-value="pageSize"
                        class="page-size-select"
                        density="compact"
                        hide-details
                        item-title="title"
                        item-value="value"
                        menu-icon="mdi-chevron-down"
                        :items="pageSizeOptions"
                        variant="plain"
                        @update:model-value="handlePageSizeChange"
                      />

                      <v-pagination
                        :length="pageCount"
                        :model-value="activePage"
                        active-color="primary"
                        density="comfortable"
                        rounded="circle"
                        size="small"
                        total-visible="5"
                        @update:model-value="handlePageChange"
                      />
                    </div>
                  </div>
                </template>
              </v-card>
            </section>

            <section v-else-if="activeSection === 'downloads'" class="page-section">
              <v-card class="section-card d-flex flex-column">
                <v-toolbar density="compact" flat class="page-toolbar download-toolbar px-2">
                  <v-btn
                    class="download-add-btn"
                    color="primary"
                    prepend-icon="mdi-link-plus"
                    size="small"
                    variant="flat"
                    :disabled="!loggedIn"
                    @click="openDownloadDialog"
                  >
                    添加
                  </v-btn>

                  <v-tabs
                    v-model="downloadFilter"
                    color="primary"
                    density="compact"
                    height="100%"
                    class="ml-4 download-tabs"
                  >
                    <v-tab
                      v-for="entry in downloadFilterOptions"
                      :key="entry.value"
                      :value="entry.value"
                      class="download-tab"
                    >
                      {{ entry.label }}
                    </v-tab>
                  </v-tabs>

                  <v-spacer />

                  <div class="download-quota mr-2">
                    <div class="download-quota-text text-caption text-medium-emphasis">
                      <span class="download-quota-label">额度</span>
                      <span class="download-quota-value">{{ downloadQuotaText }}</span>
                    </div>
                    <v-progress-linear
                      class="download-quota-bar"
                      :model-value="downloadQuotaProgress"
                      color="primary"
                      bg-color="rgba(var(--v-theme-on-surface), 0.08)"
                      height="6"
                      rounded
                    />
                  </div>

                  <v-btn
                    icon="mdi-refresh"
                    variant="text"
                    size="small"
                    class="download-refresh-btn"
                    :disabled="!loggedIn"
                    :loading="downloadLoading"
                    @click="refreshOfflineTasks"
                  />
                </v-toolbar>

                <v-progress-linear
                  :active="downloadLoading || downloadSubmitting"
                  :indeterminate="downloadLoading || downloadSubmitting"
                  height="2"
                />

                <template v-if="!loggedIn">
                  <div class="state-shell">
                    <v-icon size="52" color="primary">mdi-download-circle-outline</v-icon>
                    <div class="text-subtitle-1">请先登录 115 账号</div>
                    <div class="text-body-2 text-medium-emphasis">
                      登录后即可查看离线任务、选择保存目录并直接打开对应网盘目录。
                    </div>
                  </div>
                </template>

                <template v-else>
                  <div class="table-scroll files-scroll" :class="fileListDensityClass">
                    <table class="files-table downloads-table">
                      <thead>
                        <tr>
                          <th class="name-column">文件名</th>
                          <th class="download-progress-column">进度</th>
                          <th class="download-action-column text-right text-no-wrap">操作</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-if="!paginatedDownloadTasks.length">
                          <td colspan="3" class="empty-row">
                            {{ downloadEmptyText }}
                          </td>
                        </tr>

                        <tr v-for="task in paginatedDownloadTasks" :key="task.infoHash" class="file-row">
                          <td class="name-column">
                            <div class="name-cell">
                              <v-avatar
                                :size="fileListAvatarSize"
                                variant="tonal"
                                :color="offlineTaskColor(task)"
                              >
                                <v-icon :size="fileListIconSize" :color="offlineTaskColor(task)">
                                  {{ offlineTaskIcon(task) }}
                                </v-icon>
                              </v-avatar>

                              <div class="name-text">
                                <div class="file-title">
                                  <span class="file-title-main text-truncate">{{ task.name }}</span>
                                </div>
                                <div class="file-subtitle text-truncate">
                                  {{ offlineTaskMetaText(task) }}
                                </div>
                              </div>
                            </div>
                          </td>

                          <td class="download-progress-column">
                            <div class="download-progress-cell">
                              <div class="download-progress-summary text-caption text-medium-emphasis">
                                <span>{{ task.percentText }}</span>
                                <span class="download-progress-tail">{{ offlineTaskProgressText(task) }}</span>
                              </div>
                              <v-progress-linear
                                :model-value="task.statusGroup === 'completed' ? 100 : task.percent"
                                :color="task.statusGroup === 'failed' ? 'error' : 'primary'"
                                height="8"
                                rounded
                              />
                            </div>
                          </td>

                          <td class="download-action-column text-right text-no-wrap">
                            <div class="download-actions">
                              <v-btn
                                v-if="task.dirId"
                                icon="mdi-folder-open-outline"
                                size="small"
                                variant="text"
                                title="打开目录"
                                @click.stop="openOfflineDirectory(task)"
                              />
                              <v-btn
                                icon="mdi-content-copy"
                                size="small"
                                variant="text"
                                title="复制任务链接"
                                @click.stop="copyOfflineURL(task)"
                              />
                              <v-btn
                                icon="mdi-delete-outline"
                                size="small"
                                variant="text"
                                color="error"
                                title="删除任务"
                                @click.stop="deleteOfflineTask(task)"
                              />
                            </div>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  <div v-if="showDownloadPagination" class="pagination-bar">
                    <div class="pagination-summary text-caption text-medium-emphasis">
                      {{ downloadPaginationSummaryText }}
                    </div>

                    <div class="pagination-controls">
                      <v-select
                        :model-value="pageSize"
                        class="page-size-select"
                        density="compact"
                        hide-details
                        item-title="title"
                        item-value="value"
                        menu-icon="mdi-chevron-down"
                        :items="pageSizeOptions"
                        variant="plain"
                        @update:model-value="handleDownloadPageSizeChange"
                      />

                      <v-pagination
                        :length="downloadPageCount"
                        :model-value="downloadPage"
                        active-color="primary"
                        density="comfortable"
                        rounded="circle"
                        size="small"
                        total-visible="5"
                        @update:model-value="handleDownloadPageChange"
                      />
                    </div>
                  </div>
                </template>
              </v-card>
            </section>

            <section v-else class="page-section">
              <v-card class="section-card">
                <v-tabs v-model="settingsTab" color="primary" class="px-4 pt-2">
                  <v-tab value="player">播放器</v-tab>
                  <v-tab value="future" disabled>更多设置</v-tab>
                </v-tabs>

                <v-divider />

                <v-window v-model="settingsTab">
                  <v-window-item value="player">
                    <v-card-text class="pa-4">
                      <div class="settings-panel">
                        <div class="text-subtitle-1 font-weight-medium">播放器设置</div>
                        <div class="text-caption text-medium-emphasis mt-1 mb-2">
                          点击列表项即可切换默认播放器。
                        </div>

                        <v-list class="player-list" density="compact" lines="two">
                          <v-list-item
                            v-for="player in playerOptions"
                            :key="player.id"
                            rounded="lg"
                            class="player-list-item"
                            :class="{ 'player-list-item--disabled': player.disabled }"
                            :active="player.id === settings.preferredPlayer && !player.disabled"
                            @click="selectPlayerFromList(player)"
                          >
                            <template #prepend>
                              <v-icon :color="player.disabled ? 'medium-emphasis' : (player.available ? 'success' : 'warning')">
                                {{
                                  player.disabled
                                    ? 'mdi-pause-circle-outline'
                                    : player.available
                                      ? 'mdi-check-circle-outline'
                                      : 'mdi-alert-circle-outline'
                                }}
                              </v-icon>
                            </template>

                            <template #title>
                              <div class="player-row-title">
                                <span class="player-title">{{ player.name }}</span>
                                <div class="player-inline-badges">
                                  <v-chip
                                    v-if="player.id === settings.preferredPlayer && !player.disabled"
                                    size="x-small"
                                    color="primary"
                                    variant="tonal"
                                  >
                                    当前默认
                                  </v-chip>
                                  <v-chip
                                    size="x-small"
                                    :color="player.disabled ? 'default' : (player.available ? 'success' : 'warning')"
                                    variant="tonal"
                                  >
                                    {{ player.disabled ? '已禁用' : (player.available ? '可用' : '未就绪') }}
                                  </v-chip>
                                  <v-chip
                                    v-for="feature in capabilityTags(player)"
                                    :key="`${player.id}-${feature.label}`"
                                    size="x-small"
                                    :color="feature.color"
                                    variant="tonal"
                                  >
                                    {{ feature.label }}
                                  </v-chip>
                                </div>
                              </div>
                            </template>

                            <template #subtitle>
                              <div class="player-subtitle text-truncate">
                                {{ playerStatusText(player) }}
                              </div>
                            </template>

                            <template #append>
                              <div class="player-actions">
                                <v-btn
                                  icon="mdi-folder-open-outline"
                                  size="small"
                                  variant="text"
                                  title="选择路径"
                                  :loading="playerLoading"
                                  @click.stop="choosePlayerPath(player.id)"
                                />
                                <v-btn
                                  :icon="player.disabled ? 'mdi-play-circle-outline' : 'mdi-pause-circle-outline'"
                                  size="small"
                                  variant="text"
                                  :title="player.disabled ? '启用播放器' : '禁用播放器'"
                                  :loading="playerLoading"
                                  @click.stop="togglePlayerDisabled(player)"
                                />
                                <v-btn
                                  icon="mdi-delete-outline"
                                  size="small"
                                  variant="text"
                                  color="error"
                                  title="删除已保存路径"
                                  :loading="playerLoading"
                                  @click.stop="deletePlayer(player)"
                                />
                              </div>
                            </template>
                          </v-list-item>
                        </v-list>
                      </div>
                    </v-card-text>
                  </v-window-item>
                </v-window>
              </v-card>
            </section>
          </template>

          <v-snackbar
            :key="notice.key"
            v-model="notice.show"
            class="notice-snackbar"
            :color="notice.color"
            :timeout="notice.timeout"
            location="bottom end"
            timer="white"
            variant="flat"
            absolute
            contained
            rounded="lg"
            max-width="420"
          >
            {{ notice.text }}

            <template #actions>
              <v-btn
                icon="mdi-close"
                size="x-small"
                variant="text"
                @click="notice.show = false"
              />
            </template>
          </v-snackbar>
        </div>
      </v-main>

      <v-navigation-drawer
        v-model="detailDrawer"
        class="detail-drawer"
        location="right"
        temporary
        :scrim="false"
        width="320"
      >
        <template v-if="selectedItem">
          <div class="detail-header">
            <div class="d-flex align-center ga-3 min-w-0">
              <v-avatar
                size="36"
                variant="tonal"
                :color="selectedItem.iconColor"
              >
                <v-icon :color="selectedItem.iconColor">{{ selectedItem.icon }}</v-icon>
              </v-avatar>

              <div class="min-w-0 flex-1-1">
                <div class="text-subtitle-2 font-weight-medium text-truncate">
                  {{ selectedItem.displayName || selectedItem.name }}
                </div>
                <div class="text-caption text-medium-emphasis">
                  {{ selectedItem.kindLabel }}
                </div>
              </div>
            </div>

            <v-btn icon="mdi-close" variant="text" size="small" @click="detailDrawer = false" />
          </div>

          <v-divider />

          <div class="detail-body">
            <div class="d-flex flex-column ga-2">
              <v-btn
                v-if="selectedItem.isDirectory"
                color="primary"
                block
                prepend-icon="mdi-folder-open-outline"
                @click="handlePrimaryAction(selectedItem)"
              >
                打开目录
              </v-btn>

              <template v-else-if="selectedItem.isVideo">
                <v-btn
                  color="primary"
                  block
                  prepend-icon="mdi-play-circle-outline"
                  @click="playVideo(selectedItem)"
                >
                  立即播放
                </v-btn>
                <v-btn
                  variant="tonal"
                  block
                  prepend-icon="mdi-replay"
                  @click="playVideo(selectedItem, { fromStart: true })"
                >
                  从头播放
                </v-btn>
                <v-btn
                  variant="text"
                  block
                  prepend-icon="mdi-timeline-clock-outline"
                  @click="openJumpDialog(selectedItem)"
                >
                  跳转播放
                </v-btn>
              </template>
            </div>

            <v-list class="mt-4" density="compact" lines="two">
              <v-list-item title="类型" :subtitle="selectedItem.kindLabel" />
              <v-list-item title="大小" :subtitle="selectedItem.sizeText" />
              <v-list-item title="更新时间" :subtitle="selectedItem.updatedText" />
              <v-list-item v-if="selectedItem.pickCode" title="PickCode" :subtitle="selectedItem.pickCode" />

                        <template v-if="selectedItem.isVideo">
                          <v-list-item title="时长" :subtitle="selectedItem.durationText || '--'" />
                          <v-list-item title="续播位置" :subtitle="selectedResumeText" />
                          <v-list-item title="外挂字幕" :subtitle="selectedSubtitleName" />
                          <v-list-item title="上次播放" :subtitle="selectedLastPlayedText" />
                        </template>
            </v-list>

            <template v-if="selectedItem.isVideo">
              <v-divider class="my-2" />
              <div class="d-flex flex-column ga-2">
                <v-btn
                  variant="text"
                  prepend-icon="mdi-subtitles-outline"
                  @click="chooseSubtitle(selectedItem)"
                >
                  绑定外挂字幕
                </v-btn>
                <v-btn
                  variant="text"
                  prepend-icon="mdi-subtitles-off-outline"
                  :disabled="!selectedItem.subtitlePath"
                  @click="clearSubtitle(selectedItem)"
                >
                  清除字幕绑定
                </v-btn>
                <v-btn
                  variant="text"
                  prepend-icon="mdi-history"
                  :disabled="!selectedItem.resumeMs"
                  @click="clearProgress(selectedItem)"
                >
                  清除续播记录
                </v-btn>
              </div>
            </template>
          </div>
        </template>

        <template v-else>
          <div class="state-shell detail-empty">
            <v-icon size="40" color="medium-emphasis">mdi-cursor-default-click-outline</v-icon>
            <div class="text-body-2 text-medium-emphasis">
              点击列表项可在这里查看详情，双击即可直接打开目录或播放视频。
            </div>
          </div>
        </template>
      </v-navigation-drawer>
    </v-layout>

    <v-dialog v-model="loginDialog" max-width="440">
      <v-card>
        <div class="login-dialog-header">
          <v-tabs v-model="loginTab" color="primary" class="login-dialog-tabs">
            <v-tab value="qr">扫码登录</v-tab>
            <v-tab value="cookie">Cookie 登录</v-tab>
          </v-tabs>
          <v-btn icon="mdi-close" variant="text" size="small" @click="closeLoginDialog" />
        </div>
        <v-divider />

        <v-window v-model="loginTab">
          <v-window-item value="qr">
            <v-card-text>
              <div class="text-body-2 text-medium-emphasis">
                使用 115 App 扫描二维码即可恢复本地会话。如果你已经在浏览器里登录，也可以切到 Cookie 登录。
              </div>

              <div class="qr-shell mt-4">
                <v-progress-circular v-if="loginLoading && !qrImage" indeterminate color="primary" />
                <v-img
                  v-else-if="qrImage"
                  :src="qrImage"
                  width="220"
                  height="220"
                  cover
                  class="rounded-lg"
                />
                <v-icon v-else size="64" color="medium-emphasis">mdi-qrcode</v-icon>
              </div>

              <v-alert class="mt-4" variant="tonal" type="info">
                {{ loginStatus }}
              </v-alert>
            </v-card-text>

            <v-card-actions class="px-6 pb-4">
              <v-spacer />
              <v-btn color="primary" :loading="loginLoading" @click="startLogin">
                刷新二维码
              </v-btn>
            </v-card-actions>
          </v-window-item>

          <v-window-item value="cookie">
            <v-card-text>
              <div class="text-body-2 text-medium-emphasis mb-4">
                粘贴浏览器中 115 的 Cookie，建议至少包含 `UID`、`CID`、`SEID` 和 `KID`。
              </div>

              <v-textarea
                v-model="cookieInput"
                class="scroll-textarea scroll-textarea--cookie"
                rows="4"
                variant="outlined"
                label="Cookie"
                placeholder="UID=...; CID=...; SEID=...; KID=..."
              />
            </v-card-text>

            <v-card-actions class="px-6 pb-4">
              <v-btn variant="text" prepend-icon="mdi-clipboard-text-outline" @click="pasteCookieFromClipboard">
                从剪贴板粘贴
              </v-btn>
              <v-spacer />
              <v-btn
                color="primary"
                prepend-icon="mdi-cookie-check-outline"
                :loading="cookieSubmitting"
                @click="submitCookieLogin"
              >
                使用 Cookie 登录
              </v-btn>
            </v-card-actions>
          </v-window-item>
        </v-window>
      </v-card>
    </v-dialog>

    <v-dialog v-model="downloadDialog" max-width="680" persistent>
      <v-card>
        <v-card-title class="download-dialog-title">
          <span>添加云下载</span>
          <v-btn
            icon="mdi-close"
            size="small"
            variant="text"
            @click="closeDownloadDialog"
          />
        </v-card-title>
        <v-divider />
        <v-card-text>
          <v-textarea
            v-model="downloadInput"
            class="scroll-textarea scroll-textarea--download"
            hide-details
            persistent-placeholder
            rows="7"
            variant="outlined"
            placeholder="输入下载链接，多个链接换行分隔（支持http/https/磁力链接/ed2k）"
          />

          <div class="target-select-row">
            <v-select
              class="offline-target-select"
              density="compact"
              hide-details
              item-title="title"
              item-value="value"
              label="保存目录"
              menu-icon="mdi-chevron-down"
              :items="offlineTargetSelectOptions"
              :model-value="offlineTargetSelectValue"
              variant="outlined"
              @update:model-value="handleOfflineTargetSelect"
            >
              <template #selection="{ item }">
                <span class="text-body-2 text-truncate">{{ item?.raw?.title || offlineTargetPathText }}</span>
              </template>

              <template #item="{ props, item }">
                <v-list-item
                  v-bind="props"
                  :prepend-icon="item.raw.icon"
                  :title="item.raw.title"
                />
              </template>
            </v-select>
          </div>
        </v-card-text>
        <v-card-actions class="download-dialog-actions">
          <v-btn
            variant="tonal"
            prepend-icon="mdi-file-upload-outline"
            :loading="torrentSelecting"
            @click="selectTorrentFile"
          >
            新建BT任务
          </v-btn>
          <v-btn
            color="primary"
            prepend-icon="mdi-download-outline"
            :loading="downloadSubmitting"
            @click="submitOfflineTasks"
          >
            确认下载
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="folderPickerDialog" max-width="700">
      <v-card>
        <v-card-title>选择保存目录</v-card-title>
        <v-divider />

        <v-toolbar density="compact" flat class="px-2 folder-picker-toolbar">
          <div class="breadcrumb-strip">
            <div class="path-bar folder-picker-path-bar">
              <v-breadcrumbs
                class="file-breadcrumbs pa-0"
                :items="folderPickerBreadcrumbDisplayItems"
                divider="›"
              >
                <template #prepend>
                  <v-icon size="16" color="medium-emphasis">mdi-folder-outline</v-icon>
                </template>

                <template #title="{ item }">
                  <button
                    type="button"
                    class="file-breadcrumb-link"
                    :class="{ 'file-breadcrumb-link--disabled': item.disabled }"
                    :disabled="item.disabled"
                    @click="openFolderPickerBreadcrumb(item.id)"
                  >
                    {{ item.title }}
                  </button>
                </template>
              </v-breadcrumbs>
            </div>
          </div>
          <v-btn
            color="primary"
            variant="tonal"
            size="small"
            prepend-icon="mdi-check"
            @click="chooseFolderPickerCurrent"
          >
            选择
          </v-btn>
        </v-toolbar>

        <v-progress-linear :active="folderPickerLoading" :indeterminate="folderPickerLoading" height="2" />

        <v-card-text class="pa-0">
          <v-list density="compact">
            <v-list-item
              v-for="folder in folderPickerFolders"
              :key="folder.rowKey"
              prepend-icon="mdi-folder-outline"
              :title="folder.name"
              :subtitle="folder.updatedText"
              @click="openFolderPickerDirectory(folder)"
            />

            <v-list-item v-if="!folderPickerLoading && !folderPickerFolders.length" title="当前目录没有子文件夹" />
          </v-list>
        </v-card-text>

        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="closeFolderPicker">取消</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="jumpDialogVisible" max-width="420">
      <v-card>
        <v-card-title>跳转播放</v-card-title>
        <v-divider />
        <v-card-text>
          <v-text-field
            v-model="jumpInput"
            label="时间点"
            placeholder="例如 90、01:30、01:02:03"
          />
          <div class="text-caption text-medium-emphasis">
            输入秒数或时分秒，播放器会从该时间点开始拉起。
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="jumpDialogVisible = false">取消</v-btn>
          <v-btn color="primary" @click="confirmJump">开始播放</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-app>
</template>

<style scoped>
.app-shell {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.app-main {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.app-main :deep(.v-main__scroller) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.sidebar {
  border-right: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  overflow: hidden;
}

.sidebar :deep(.v-navigation-drawer__content) {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sidebar-layout {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.sidebar-nav {
  flex: 0 0 auto;
}

.sidebar-footer {
  margin-top: auto;
}

.account-panel {
  min-height: 0;
  display: flex;
  flex-direction: column;
  cursor: pointer;
  background: transparent !important;
  box-shadow: none !important;
}

.account-panel-head {
  align-items: center;
}

.account-panel-head--guest {
  padding: 8px 10px !important;
}

.account-panel-title {
  padding-inline-end: 0;
  min-width: 0;
  height: 42px;
  display: flex;
  align-items: center;
}

.account-panel-title-stack {
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  gap: 2px;
  width: 100%;
  overflow: hidden;
}

.account-panel-name {
  min-width: 0;
  width: 100%;
  font-size: 14px;
  line-height: 1.28;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.account-panel-meta {
  min-width: 0;
  width: 100%;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}

.account-vip-inline-pill {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  padding: 1px 9px;
  border-radius: 999px;
  border: 1px solid rgba(198, 151, 14, 0.24);
  background: #fff3bf;
  color: #9c6b00;
  font-size: 10px;
  line-height: 1.4;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.account-vip-inline-text,
.account-vip-inline-expire-text {
  min-width: 0;
  font-size: 11px;
  line-height: 1.35;
  color: rgba(var(--v-theme-on-surface), 0.72);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-panel-body {
  padding-top: 4px;
  padding-bottom: 5px !important;
  flex: 0 0 auto;
  display: block;
}

.account-summary-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.account-summary-main {
  min-width: 0;
  flex: 1 1 auto;
}

.account-summary-label {
  font-size: 11px;
  line-height: 1.2;
  color: rgba(var(--v-theme-on-surface), 0.58);
}

.account-summary-value {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.35;
  font-weight: 600;
  white-space: normal;
  word-break: break-word;
}

.account-summary-side {
  flex: 0 0 auto;
  padding-top: 2px;
  font-size: 12px;
  line-height: 1.2;
  font-weight: 600;
  color: rgba(var(--v-theme-primary), 1);
}

.account-space-bar {
  margin-top: 6px;
  margin-bottom: 0;
}

.account-panel-caption {
  margin-top: 8px;
}

.account-avatar {
  overflow: hidden;
}

.account-avatar-image-shell {
  display: block;
  position: relative;
  width: 30px;
  height: 30px;
  overflow: hidden;
  border-radius: 999px;
  background: #fff;
  box-sizing: border-box;
  flex: 0 0 auto;
  -webkit-mask-image: -webkit-radial-gradient(white, black);
}

.account-avatar-image-shell--large {
  width: 42px;
  height: 42px;
}

.account-avatar-image {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
  backface-visibility: hidden;
  transform: translateZ(0);
  filter: none !important;
  opacity: 1 !important;
  mix-blend-mode: normal !important;
}

.workspace {
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
  box-sizing: border-box;
  padding: 0 0 6px;
  overflow: hidden;
}

.page-section,
.section-card {
  min-height: 0;
}

.page-section {
  flex: 1 1 auto;
  display: flex;
  box-sizing: border-box;
  padding-bottom: 2px;
  overflow: hidden;
}

.section-card {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  background: transparent !important;
  border-radius: 0 !important;
  box-shadow: none !important;
  min-height: 0;
}

.section-card :deep(.v-window) {
  flex: 1 1 auto;
  min-height: 0;
}

.page-toolbar {
  gap: 8px;
}

.page-toolbar :deep(.v-toolbar__content) {
  gap: 10px;
  padding-inline: 8px !important;
}

.flex-1-1 {
  flex: 1 1 auto;
}

.breadcrumb-strip {
  min-width: 0;
  flex: 1 1 auto;
  display: flex;
  align-items: center;
}

.path-bar {
  min-width: 0;
  flex: 1 1 auto;
  min-height: 32px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px 0 8px;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.1);
  border-radius: 8px;
  background: rgba(var(--v-theme-surface), 1);
}

.file-breadcrumbs {
  min-width: 0;
  overflow: hidden;
  flex: 1 1 auto;
  width: auto;
  min-height: 30px;
  display: flex;
  justify-content: flex-start;
}

.file-breadcrumbs :deep(.v-breadcrumbs__prepend),
.file-breadcrumbs :deep(.v-breadcrumbs__divider),
.file-breadcrumbs :deep(.v-breadcrumbs-item) {
  display: flex;
  align-items: center;
}

.file-breadcrumbs :deep(.v-breadcrumbs__prepend) {
  padding-inline-end: 6px;
}

.file-breadcrumbs :deep(.v-breadcrumbs-item) {
  min-width: 0;
}

.file-breadcrumb-link {
  max-width: 220px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-breadcrumb-link--disabled {
  cursor: default;
  color: rgba(var(--v-theme-on-surface), 0.88);
}

.path-refresh {
  flex: 0 0 auto;
  margin-inline-start: auto;
}

.search-path-indicator {
  min-width: 0;
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: rgba(var(--v-theme-on-surface), 0.78);
}

.search-slot {
  min-width: 0;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.search-shell {
  width: 32px;
  min-width: 32px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  overflow: hidden;
  transition: width 0.18s ease;
}

.search-shell--expanded {
  width: 194px;
}

.search-field-wrap {
  width: 0;
  min-width: 0;
  margin-inline-start: 0;
  opacity: 0;
  overflow: hidden;
  pointer-events: none;
  transition:
    width 0.18s ease,
    margin-inline-start 0.18s ease,
    opacity 0.14s ease;
}

.search-shell--expanded .search-field-wrap {
  width: 156px;
  margin-inline-start: 4px;
  opacity: 1;
  pointer-events: auto;
}

.search-trigger {
  flex: 0 0 auto;
}

.compact-search {
  width: 156px;
  max-width: 156px;
}

.compact-search :deep(.v-field) {
  min-height: 32px;
  border-radius: 8px;
  box-shadow: none;
}

.compact-search :deep(.v-field__prepend-inner),
.compact-search :deep(.v-field__append-inner),
.compact-search :deep(.v-field__clearable) {
  padding-top: 0;
}

.compact-search :deep(.v-field__input) {
  min-height: 32px;
  padding-top: 0;
  padding-bottom: 0;
  font-size: 13px;
}

.filter-menu {
  border-radius: 10px;
}

.filter-trigger {
  color: rgba(var(--v-theme-on-surface), 0.68) !important;
  transition: color 0.18s ease;
}

.filter-trigger--active {
  color: rgb(var(--v-theme-primary)) !important;
}

.filter-menu-body {
  padding: 12px 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.filter-select :deep(.v-field) {
  border-radius: 8px;
}

.filter-select :deep(.v-field__input) {
  min-height: 34px;
  padding-top: 0;
  padding-bottom: 0;
  font-size: 13px;
}

.filter-divider {
  margin: 2px 0;
}

.filter-checkbox {
  margin: 0;
}

.filter-checkbox :deep(.v-selection-control) {
  min-height: 30px;
}

.filter-checkbox :deep(.v-label) {
  font-size: 13px;
  opacity: 1;
}

.scroll-textarea :deep(.v-field__field) {
  align-items: stretch;
}

.scroll-textarea :deep(.v-field__input) {
  padding-top: 6px;
  padding-bottom: 6px;
  -webkit-mask-image: none;
  mask-image: none;
}

.scroll-textarea :deep(textarea) {
  overflow-y: auto !important;
  resize: none;
}

.scroll-textarea--cookie :deep(textarea) {
  min-height: 130px;
  max-height: 130px;
}

.scroll-textarea--download :deep(textarea) {
  min-height: 168px;
  max-height: 168px;
}

.table-scroll {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  box-sizing: border-box;
  overflow: hidden;
}

.files-scroll {
  --files-row-gap: 4px;
  --files-header-height: 34px;
  --files-header-padding-x: 14px;
  --files-header-font-size: 12px;
  --files-cell-padding-y: 11px;
  --files-cell-padding-x: 14px;
  --files-name-gap: 10px;
  --files-title-font-size: 13px;
  --files-title-line-height: 1.25;
  --files-meta-gap: 6px;
  --files-meta-margin-top: 2px;
  --files-kind-font-size: 11px;
  --files-badge-gap: 4px;
  flex: 1 1 0;
  height: 0;
  min-height: 0;
  display: block;
  overflow: auto;
  box-sizing: border-box;
  padding-bottom: 12px;
  scrollbar-gutter: stable both-edges;
}

.files-scroll.files-density--compact {
  --files-row-gap: 2px;
  --files-header-height: 30px;
  --files-header-padding-x: 12px;
  --files-header-font-size: 11px;
  --files-cell-padding-y: 8px;
  --files-cell-padding-x: 12px;
  --files-name-gap: 8px;
  --files-title-font-size: 12px;
  --files-title-line-height: 1.2;
  --files-meta-gap: 5px;
  --files-meta-margin-top: 1px;
  --files-kind-font-size: 10px;
  --files-badge-gap: 3px;
}

.files-scroll.files-density--comfortable {
  --files-row-gap: 6px;
  --files-header-height: 38px;
  --files-header-padding-x: 16px;
  --files-header-font-size: 13px;
  --files-cell-padding-y: 13px;
  --files-cell-padding-x: 16px;
  --files-name-gap: 12px;
  --files-title-font-size: 14px;
  --files-title-line-height: 1.3;
  --files-meta-gap: 7px;
  --files-meta-margin-top: 3px;
  --files-kind-font-size: 12px;
  --files-badge-gap: 5px;
}

.table-scroll :deep(.v-table) {
  flex: 1 1 auto;
  min-height: 0;
  height: 100%;
}

.table-scroll :deep(.v-table__wrapper) {
  min-height: 0;
  overflow-x: auto;
  overflow-y: auto;
  box-sizing: border-box;
  padding-bottom: 10px;
}

.notice-snackbar {
  margin: 12px !important;
  padding: 0 !important;
  z-index: 6;
}

.notice-snackbar :deep(.v-snackbar__wrapper) {
  min-width: 280px;
  max-width: min(420px, calc(100vw - 48px));
}

.notice-snackbar :deep(.v-snackbar__content) {
  padding: 12px 14px 16px;
  line-height: 1.45;
}

.notice-snackbar :deep(.v-snackbar__actions) {
  align-self: stretch;
  margin-inline-end: 6px;
}

.notice-snackbar :deep(.v-snackbar__timer) {
  top: auto;
  bottom: 0;
}

.name-column {
  width: 100%;
  min-width: 0;
}

.files-table {
  table-layout: fixed;
  width: 100%;
  border-collapse: separate;
  border-spacing: 0 var(--files-row-gap);
}

.files-table th {
  height: var(--files-header-height);
  padding: 0 var(--files-header-padding-x);
  vertical-align: middle;
  font-size: var(--files-header-font-size);
  font-weight: 600;
  color: rgba(var(--v-theme-on-surface), 0.76);
  position: sticky;
  top: 0;
  z-index: 1;
  background: rgba(var(--v-theme-surface), 0.96);
  backdrop-filter: blur(6px);
}

.files-table td {
  padding: var(--files-cell-padding-y) var(--files-cell-padding-x);
  vertical-align: middle;
  background: rgba(var(--v-theme-surface), 1);
}

.files-table tbody tr td:first-child {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
}

.files-table tbody tr td:last-child {
  border-top-right-radius: 8px;
  border-bottom-right-radius: 8px;
}

.files-table .name-column {
  width: auto;
}

.files-table .size-column {
  width: 86px;
}

.files-table .time-column {
  width: 136px;
}

.downloads-table .download-progress-column {
  width: 196px;
}

.downloads-table .download-action-column {
  width: 136px;
}

.downloads-table th.download-action-column,
.downloads-table td.download-action-column {
  padding-inline: 10px 12px;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: var(--files-name-gap);
  min-width: 0;
}

.name-avatar {
  flex: 0 0 auto;
}

.name-text {
  min-width: 0;
  width: 100%;
}

.file-title {
  display: flex;
  align-items: baseline;
  gap: 0;
  font-size: var(--files-title-font-size);
  line-height: var(--files-title-line-height);
  font-weight: 500;
  color: rgba(var(--v-theme-on-surface), 0.94);
}

.file-title-main {
  min-width: 0;
  flex: 1 1 auto;
}

.file-title-extension {
  flex: 0 0 auto;
}

.file-subtitle {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.2;
  color: rgba(var(--v-theme-on-surface), 0.58);
}

.file-meta-row {
  display: flex;
  align-items: center;
  gap: var(--files-meta-gap);
  min-width: 0;
  margin-top: var(--files-meta-margin-top);
  overflow: hidden;
}

.file-kind-label {
  flex: 0 0 auto;
  font-size: var(--files-kind-font-size);
  line-height: 1.2;
  color: rgba(var(--v-theme-on-surface), 0.56);
}

.file-badge-row {
  display: flex;
  align-items: center;
  flex: 1 1 auto;
  gap: var(--files-badge-gap);
  min-width: 0;
  overflow: hidden;
}

.file-badge {
  flex: 0 0 auto;
}

.pagination-bar {
  flex: 0 0 auto;
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 12px 8px;
  border-top: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}

.pagination-summary {
  white-space: nowrap;
}

.pagination-controls {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.page-size-select {
  width: 92px;
  min-width: 92px;
}

.page-size-select :deep(.v-input__control) {
  min-height: 30px;
}

.page-size-select :deep(.v-field) {
  align-items: center;
  padding-inline-end: 0;
  border-radius: 8px;
}

.page-size-select :deep(.v-field__field) {
  display: flex;
  align-items: center;
}

.page-size-select :deep(.v-field__input) {
  display: flex;
  align-items: center;
  min-height: 28px;
  padding-top: 0;
  padding-bottom: 0;
  font-size: 12px;
}

.page-size-select :deep(.v-select__selection) {
  display: inline-flex;
  align-items: center;
  line-height: 1;
}

.page-size-select :deep(.v-field__append-inner) {
  display: flex;
  align-items: center;
  padding-top: 0;
}

.page-size-select :deep(.v-select__menu-icon) {
  margin-top: 0;
  transform-origin: center;
}

.file-row {
  cursor: pointer;
}

.files-table .selected-row td {
  background: rgba(var(--v-theme-primary), 0.08);
}

.player-list {
  padding: 0;
}

.settings-panel {
  height: 100%;
}

.player-list-item {
  margin-bottom: 6px;
  cursor: pointer;
}

.player-list-item--disabled {
  opacity: 0.82;
}

.player-row-title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.player-inline-badges {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.player-title {
  font-size: 14px;
  font-weight: 500;
}

.player-subtitle {
  margin-top: 2px;
  line-height: 1.35;
  white-space: normal;
}

.player-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-inline-start: 10px;
}

.empty-row {
  padding: 32px 16px;
  text-align: center;
  color: rgba(var(--v-theme-on-surface), 0.6);
}

.state-shell,
.qr-shell,
.detail-empty {
  height: 100%;
  min-height: 240px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  text-align: center;
}

.downloads-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.download-progress-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.download-progress-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.download-progress-tail {
  min-width: 0;
  flex: 1 1 auto;
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.download-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  min-width: 104px;
  gap: 4px;
}

.drop-zone {
  --wails-drop-target: drop;
  padding: 18px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  text-align: center;
  border: 1px dashed rgba(var(--v-theme-primary), 0.35);
}

.target-select-row {
  margin-top: 8px;
  min-width: 0;
  width: 100%;
}

.download-dialog-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.download-dialog-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 24px 18px !important;
}

.login-dialog-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 0 16px;
}

.login-dialog-tabs {
  min-width: 0;
  flex: 1 1 auto;
}

.login-dialog-tabs :deep(.v-slide-group__content) {
  gap: 4px;
}

.login-dialog-tabs :deep(.v-tab) {
  min-width: 0;
  padding-inline: 12px;
}

.offline-target-select {
  width: 100%;
  min-width: 0;
}

.offline-target-select :deep(.v-field) {
  border-radius: 8px;
}

.offline-target-select :deep(.v-field__input) {
  min-height: 34px;
  padding-top: 0;
  padding-bottom: 0;
}

.offline-target-select :deep(.v-select__selection) {
  min-width: 0;
  max-width: 100%;
}

.offline-target-select :deep(.v-list-item-title) {
  font-size: 13px;
}

.detail-header {
  padding: 14px 14px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.detail-body {
  padding: 16px;
}

.text-break {
  word-break: break-all;
}

.download-tabs {
  min-width: 0;
  display: flex;
  align-self: stretch;
}

.download-toolbar :deep(.v-toolbar__content) {
  min-height: 36px !important;
  padding-block: 0 !important;
  align-items: center;
}

.download-add-btn {
  min-width: 0;
  padding-inline: 10px !important;
  align-self: center;
}

.download-refresh-btn {
  width: 30px;
  height: 30px;
  align-self: center;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.download-quota {
  width: 112px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 3px;
  align-self: center;
}

.download-quota-text {
  line-height: 1.1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  text-align: left;
}

.download-quota-label {
  font-weight: 700;
  flex: 0 0 auto;
}

.download-quota-value {
  min-width: 0;
  flex: 1 1 auto;
  text-align: right;
}

.download-quota-bar {
  margin-inline-start: auto;
  width: 100%;
}

.download-quota-bar :deep(.v-progress-linear__background) {
  opacity: 1 !important;
  background: rgba(var(--v-theme-primary), 0.14) !important;
}

.download-tabs :deep(.v-slide-group),
.download-tabs :deep(.v-slide-group__container) {
  height: 100%;
}

.download-tabs :deep(.v-slide-group__content) {
  min-height: 36px;
  height: 100%;
  align-items: stretch;
}

.download-tabs :deep(.v-tab) {
  min-height: 36px;
  height: 100%;
  padding-inline: 12px;
  font-size: 13px;
  text-transform: none;
}

.download-tabs :deep(.v-tab__slider) {
  bottom: 0;
}

.download-tab {
  min-width: 0;
}

@media (max-width: 960px) {
  .search-shell--expanded {
    width: 170px;
  }

  .search-shell--expanded .search-field-wrap {
    width: 132px;
  }

  .compact-search {
    width: 132px;
    max-width: 132px;
  }

  .file-breadcrumb-link {
    max-width: 120px;
  }
}
</style>
