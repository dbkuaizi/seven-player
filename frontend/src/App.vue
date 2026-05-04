<script setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import {
  AddOfflineTasks,
  Bootstrap,
  ClearPlaybackProgress,
  ClearSubtitlePath,
  DeleteOfflineTasks,
  ListDirectory,
  ListOfflineTasks,
  PlayFile,
  PrepareBuiltinPlayback,
  PreviewDirectory,
  SavePlaybackProgress,
  SearchFiles,
  SelectSubtitlePath,
  SelectTorrentFileAsMagnet,
} from "../bindings/panplayer/app";
import { Clipboard } from "@wailsio/runtime";
import AppSidebar from "./components/app/AppSidebar.vue";
import NoticeSnackbar from "./components/app/NoticeSnackbar.vue";
import LoginDialog from "./components/auth/LoginDialog.vue";
import DownloadDialog from "./components/downloads/DownloadDialog.vue";
import DownloadsPage from "./components/downloads/DownloadsPage.vue";
import FolderPickerDialog from "./components/downloads/FolderPickerDialog.vue";
import FileDetailDrawer from "./components/files/FileDetailDrawer.vue";
import FileManagerPage from "./components/files/FileManagerPage.vue";
import BuiltinPlayerDialog from "./components/player/BuiltinPlayerDialog.vue";
import JumpDialog from "./components/player/JumpDialog.vue";
import SettingsPage from "./components/settings/SettingsPage.vue";
import {
  OFFLINE_TARGET_PICKER_VALUE,
  downloadFilterOptions,
  fileListDensityOptions,
  navigationItems,
  pageSizeOptions,
  smallFileFilterOptions,
  sortOptions,
  typeOptions,
} from "./constants/options";
import { useAccountSummary } from "./composables/useAccountSummary";
import { useLoginSession } from "./composables/useLoginSession";
import { useNotice } from "./composables/useNotice";
import { useSettingsState } from "./composables/useSettingsState";
import {
  createDirectoryTarget,
  createOfflineTargetOption,
  normalizeDirectoryTargetValue,
  normalizeOfflineRecentTargets,
  rememberOfflineRecentTargets,
  rootBreadcrumb,
} from "./utils/directoryTarget";
import {
  breadcrumbPathUntil,
  normalizeBreadcrumbPath,
  resolveDirectoryPath,
} from "./utils/breadcrumbs";
import { isAppNotReadyError } from "./utils/error";
import { sanitizeErrorMessage } from "./utils/error";
import {
  compareItems,
  formatResumeProgressText,
  normalizeFileItem,
} from "./utils/filePresentation";
import { formatDateTime, formatDurationMs } from "./utils/format";
import {
  createEmptyOfflineState,
  normalizeOfflineState,
  offlineTaskMetaText,
} from "./utils/offlineState";
import { createEmptyBuiltinPlayer } from "./utils/playback";
import { normalizeFileListDensity } from "./utils/settings";
import {
  basename,
  normalizeMultilineInput,
  parseTimeInput,
} from "./utils/text";

const activeSection = ref("files");

const booting = ref(true);
const directoryLoading = ref(false);
const searchLoading = ref(false);
const actionLoading = ref(false);
const downloadLoading = ref(false);
const downloadSubmitting = ref(false);
const folderPickerLoading = ref(false);
const torrentSelecting = ref(false);

const loggedIn = ref(false);
const user = ref(null);
const proxyBase = ref("");

const currentDir = ref("0");
const parentId = ref("");
const currentPath = ref([rootBreadcrumb()]);
const items = ref([]);
const directoryTotal = ref(0);
const directoryPage = ref(1);
const selectedItem = ref(null);
const detailDrawer = ref(false);

const searchQuery = ref("");
const typeFilter = ref("all");
const sortMode = ref("folders");
const searchExpanded = ref(false);
const filterMenuOpen = ref(false);
const searchResults = ref([]);
const searchTotal = ref(0);
const searchPage = ref(1);
const pageSize = ref(20);
const downloadPage = ref(1);
const downloadPageSize = ref(20);
const searchDebounceTimer = ref(null);
const searchRequestId = ref(0);

const jumpDialogVisible = ref(false);
const jumpInput = ref("");
const jumpTargetKey = ref("");

const builtinPlayerDialog = ref(false);
const builtinPlayerLoading = ref(false);
const builtinPlayer = ref(createEmptyBuiltinPlayer());
const builtinPlayerSeekKey = ref("");

const { notice, showNotice, showError, closeNotice } = useNotice();

const {
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
  handleLogout: logoutSession,
  clearPolling,
} = useLoginSession({
  loggedIn,
  user,
  showNotice,
  showError,
  onLoggedIn: () => loadDirectory(currentDir.value || "0", {}, 1),
  onLoggedOut: () => {
    searchQuery.value = "";
    downloadState.value = createEmptyOfflineState();
    resetDirectoryState();
  },
});

const {
  settingsTab,
  settings,
  playerLoading,
  smallFileFilterMB,
  playerOptions,
  applySettingsView,
  saveShowTitleBadges,
  saveSmallFileFilter,
  saveFileListDensity,
  choosePlayerPath,
  selectPlayerFromList,
  togglePlayerDisabled,
  deletePlayer,
} = useSettingsState({
  showNotice,
  showError,
  refreshFilePresentation,
});

const downloadDialog = ref(false);
const downloadInput = ref("");
const downloadFilter = ref("active");
const downloadDeleteFiles = ref(false);
const downloadState = ref(createEmptyOfflineState());
const offlineTargetDir = ref(createDirectoryTarget("0", [rootBreadcrumb()]));
const knownDirectoryPaths = ref({
  0: [rootBreadcrumb()],
});

const folderPickerDialog = ref(false);
const folderPickerDirId = ref("0");
const folderPickerPath = ref([rootBreadcrumb()]);
const folderPickerFolders = ref([]);

const breadcrumbItems = computed(() => {
  if (currentPath.value?.length) {
    return currentPath.value;
  }
  return [rootBreadcrumb()];
});

const breadcrumbDisplayItems = computed(() =>
  breadcrumbItems.value.map((crumb, index) => ({
    id: crumb.id,
    title: crumb.name,
    disabled: index === breadcrumbItems.value.length - 1,
  })),
);

const folderPickerBreadcrumbDisplayItems = computed(() =>
  folderPickerPath.value.map((crumb, index) => ({
    id: crumb.id,
    title: crumb.name,
    disabled: index === folderPickerPath.value.length - 1,
  })),
);

const {
  accountDisplayName,
  accountAvatarUrl,
  accountVipLevelText,
  accountVipExpireText,
  accountSpaceUsageText,
  accountSpacePercent,
  accountSpacePercentText,
} = useAccountSummary({ loggedIn, user, proxyBase });
const isGlobalSearchActive = computed(
  () => searchQuery.value.trim().length > 0,
);
const isSearchInputVisible = computed(
  () => searchExpanded.value || isGlobalSearchActive.value,
);
const fileListDensityValue = computed(() =>
  normalizeFileListDensity(settings.value?.fileListDensity),
);
const fileListDensityClass = computed(
  () => `files-density--${fileListDensityValue.value}`,
);
const fileListAvatarSize = computed(() => {
  if (fileListDensityValue.value === "compact") return 26;
  if (fileListDensityValue.value === "comfortable") return 30;
  return 28;
});
const fileListIconSize = computed(() => {
  if (fileListDensityValue.value === "compact") return 14;
  if (fileListDensityValue.value === "comfortable") return 17;
  return 16;
});
const sourceItems = computed(() =>
  isGlobalSearchActive.value ? searchResults.value : items.value,
);
const activePage = computed(() =>
  isGlobalSearchActive.value ? searchPage.value : directoryPage.value,
);
const activeResultTotal = computed(() =>
  isGlobalSearchActive.value ? searchTotal.value : directoryTotal.value,
);
const pageCount = computed(() => {
  const total = Number(activeResultTotal.value || 0);
  return Math.max(1, Math.ceil(total / pageSize.value) || 1);
});
const showPagination = computed(
  () => loggedIn.value && activeResultTotal.value > 0,
);
const paginationSummaryText = computed(() => {
  const total = Number(activeResultTotal.value || 0);
  if (!total) {
    return "0 项";
  }

  const currentPage = activePage.value;
  const start = (currentPage - 1) * pageSize.value + 1;
  const visibleCount = sourceItems.value.length;
  const end =
    visibleCount > 0
      ? Math.min(total, start + visibleCount - 1)
      : Math.min(total, currentPage * pageSize.value);
  return `第 ${currentPage} 页 · ${start}-${end} / ${total} 项`;
});
const searchSummaryText = computed(() => {
  const keyword = searchQuery.value.trim();
  if (!keyword) {
    return "";
  }

  const total = searchTotal.value || searchResults.value.length;
  if (total > 0) {
    return `全局搜索 · ${keyword} · ${total} 项 · 第 ${searchPage.value} 页`;
  }
  return `全局搜索 · ${keyword}`;
});
const fileEmptyText = computed(() => {
  if (isGlobalSearchActive.value) {
    return searchLoading.value ? "正在搜索整个网盘…" : "没有找到匹配结果。";
  }
  return "当前目录没有匹配项。";
});

const filteredItems = computed(() => {
  let list = [...sourceItems.value];

  if (typeFilter.value !== "all") {
    list = list.filter((item) => {
      if (typeFilter.value === "dir") return item.isDirectory;
      if (typeFilter.value === "video") return item.isVideo;
      if (typeFilter.value === "file")
        return !item.isDirectory && !item.isVideo;
      return true;
    });
  }

  if (smallFileFilterMB.value > 0) {
    list = list.filter(
      (item) =>
        item.isDirectory ||
        Number(item.size || 0) >= smallFileFilterMB.value * 1024 * 1024,
    );
  }

  if (!(isGlobalSearchActive.value && sortMode.value === "folders")) {
    list.sort((left, right) => compareItems(left, right, sortMode.value));
  }
  return list;
});

const selectedSubtitleName = computed(() => {
  if (!selectedItem.value?.subtitlePath) {
    return "未绑定";
  }
  return basename(selectedItem.value.subtitlePath);
});

const selectedResumeText = computed(() => {
  if (!selectedItem.value?.resumeMs) {
    return "暂无续播记录";
  }
  return formatResumeProgressText(
    selectedItem.value.resumeMs,
    selectedItem.value.durationSec,
  );
});

const selectedLastPlayedText = computed(() => {
  if (!selectedItem.value?.lastPlayedAt) {
    return "暂无";
  }
  return formatDateTime(selectedItem.value.lastPlayedAt);
});

const builtinTextTracks = computed(() => {
  if (!builtinPlayer.value?.subtitleUrl || !builtinPlayer.value?.subtitleType) {
    return [];
  }

  return [
    {
      src: builtinPlayer.value.subtitleUrl,
      type: builtinPlayer.value.subtitleType,
      kind: "subtitles",
      label: builtinPlayer.value.subtitleName || "外挂字幕",
      language: "zh-CN",
      default: true,
    },
  ];
});

const downloadTasks = computed(() => downloadState.value?.tasks ?? []);

const filteredDownloadTasks = computed(() =>
  downloadTasks.value.filter(
    (task) => task.statusGroup === downloadFilter.value,
  ),
);

const downloadPageCount = computed(() =>
  Math.max(
    1,
    Math.ceil(filteredDownloadTasks.value.length / downloadPageSize.value) || 1,
  ),
);

const paginatedDownloadTasks = computed(() => {
  const start = (downloadPage.value - 1) * downloadPageSize.value;
  return filteredDownloadTasks.value.slice(
    start,
    start + downloadPageSize.value,
  );
});

const presentedDownloadTasks = computed(() =>
  paginatedDownloadTasks.value.map((task) => ({
    ...task,
    metaText: offlineTaskMetaText(task),
  })),
);

const showDownloadPagination = computed(
  () => loggedIn.value && filteredDownloadTasks.value.length > 0,
);

const downloadQuotaCapacity = computed(() => {
  const quota = Math.max(0, Number(downloadState.value?.quota || 0));
  const total = Math.max(0, Number(downloadState.value?.total || 0));
  const capacity = Math.max(quota, total);
  if (!(capacity > 0)) {
    return 0;
  }
  return capacity;
});

const downloadQuotaProgress = computed(() => {
  if (!(downloadQuotaCapacity.value > 0)) {
    return 0;
  }
  return Math.min(
    100,
    Math.max(
      0,
      (Number(downloadState.value?.quota || 0) / downloadQuotaCapacity.value) *
        100,
    ),
  );
});

const downloadQuotaText = computed(() => {
  if (!(downloadQuotaCapacity.value > 0)) {
    return "0 / 0";
  }
  return `${Math.max(0, Number(downloadState.value?.quota || 0))} / ${downloadQuotaCapacity.value}`;
});

const downloadPaginationSummaryText = computed(() => {
  const total = filteredDownloadTasks.value.length;
  if (!total) {
    return "0 项";
  }

  const start = (downloadPage.value - 1) * downloadPageSize.value + 1;
  const end = Math.min(total, start + paginatedDownloadTasks.value.length - 1);
  return `第 ${downloadPage.value} 页 · ${start}-${end} / ${total} 项`;
});

const downloadEmptyText = computed(() => {
  if (downloadFilter.value === "failed") return "当前没有失败任务。";
  if (downloadFilter.value === "completed") return "当前没有完成记录。";
  return "当前没有进行中的下载任务。";
});

const offlineRecentTargets = computed(() =>
  normalizeOfflineRecentTargets(settings.value?.offlineRecentTargets),
);

const offlineTargetPathText = computed(
  () =>
    (offlineTargetDir.value?.path ?? []).map((item) => item.name).join(" / ") ||
    "我的文件",
);

const offlineTargetSelectOptions = computed(() => {
  const options = [];
  const seen = new Set();
  const activeTarget = normalizeDirectoryTargetValue(
    offlineTargetDir.value?.id,
    offlineTargetDir.value?.path,
  );

  if (
    activeTarget &&
    !offlineRecentTargets.value.some((item) => item.id === activeTarget.id)
  ) {
    options.push(createOfflineTargetOption(activeTarget));
    seen.add(activeTarget.id);
  }

  for (const target of offlineRecentTargets.value) {
    if (seen.has(target.id)) {
      continue;
    }
    options.push(createOfflineTargetOption(target));
    seen.add(target.id);
  }

  options.push({
    value: OFFLINE_TARGET_PICKER_VALUE,
    title: "选择其他目录",
    icon: "mdi-folder-search-outline",
    isPicker: true,
  });

  return options;
});

const offlineTargetSelectValue = computed(() => {
  const activeTarget = normalizeDirectoryTargetValue(
    offlineTargetDir.value?.id,
    offlineTargetDir.value?.path,
  );
  if (!activeTarget) {
    return OFFLINE_TARGET_PICKER_VALUE;
  }

  const matched = offlineTargetSelectOptions.value.find(
    (item) => !item.isPicker && item.target?.id === activeTarget.id,
  );

  return matched?.value || OFFLINE_TARGET_PICKER_VALUE;
});

watch(filteredItems, (list) => {
  if (!list.length) {
    selectedItem.value = null;
    detailDrawer.value = false;
    return;
  }

  const currentKey = selectedItem.value?.rowKey;
  const matched = currentKey
    ? list.find((item) => item.rowKey === currentKey)
    : null;
  selectedItem.value = matched ?? list[0];
});

watch([activeSection, loggedIn], ([section, signedIn]) => {
  if (section === "downloads" && signedIn) {
    refreshOfflineTasks().catch((error) => showError(error, "读取云下载失败"));
  } else if (!signedIn) {
    downloadState.value = createEmptyOfflineState();
    searchLoading.value = false;
    searchResults.value = [];
    searchTotal.value = 0;
    searchPage.value = 1;
    directoryTotal.value = 0;
    directoryPage.value = 1;
  }
});

watch(searchQuery, (value) => {
  clearSearchDebounce();
  searchRequestId.value += 1;

  const keyword = String(value || "").trim();
  if (!keyword || !loggedIn.value) {
    searchLoading.value = false;
    searchResults.value = [];
    searchTotal.value = 0;
    searchPage.value = 1;
    return;
  }

  searchPage.value = 1;
  searchResults.value = [];
  searchTotal.value = 0;
  searchDebounceTimer.value = window.setTimeout(() => {
    performGlobalSearch(keyword, searchRequestId.value, 1).catch((error) =>
      showError(error, "搜索失败"),
    );
  }, 260);
});

watch(downloadFilter, () => {
  downloadPage.value = 1;
});

watch(filteredDownloadTasks, (tasks) => {
  const maxPage = Math.max(
    1,
    Math.ceil(tasks.length / downloadPageSize.value) || 1,
  );
  if (downloadPage.value > maxPage) {
    downloadPage.value = maxPage;
  }
});

onMounted(async () => {
  window.addEventListener("keydown", handleKeydown);
  await bootstrapApp();
});

onBeforeUnmount(() => {
  clearPolling();
  clearSearchDebounce();
  window.removeEventListener("keydown", handleKeydown);
});

async function bootstrapApp() {
  booting.value = true;

  try {
    const boot = await bootstrapWithRetry();
    proxyBase.value = boot?.proxyBase || "";
    loggedIn.value = Boolean(boot?.loggedIn);
    user.value = boot?.user ?? null;
    applySettingsView(boot?.settings);
    currentDir.value = boot?.currentId || "0";

    if (loggedIn.value) {
      await loadDirectory(currentDir.value || "0", {}, 1);
    } else {
      resetDirectoryState();
    }
  } catch (error) {
    resetDirectoryState();
    showError(error, "初始化失败");
  } finally {
    booting.value = false;
  }
}

async function bootstrapWithRetry() {
  let lastError = null;

  for (let attempt = 0; attempt < 8; attempt += 1) {
    try {
      return await Bootstrap();
    } catch (error) {
      lastError = error;
      if (!isAppNotReadyError(error) || attempt === 7) {
        break;
      }
      await sleep(180 * (attempt + 1));
    }
  }

  throw lastError ?? new Error("初始化失败");
}

async function loadDirectory(dirId, options = {}, page = directoryPage.value) {
  if (!loggedIn.value) {
    return;
  }

  directoryLoading.value = true;
  const previousKey = selectedItem.value?.rowKey ?? "";
  const nextPage = Math.max(1, Number(page || 1));
  const offset = (nextPage - 1) * pageSize.value;

  try {
    const data = await ListDirectory(dirId, offset, pageSize.value);
    const resolvedPath = resolveDirectoryPath(
      data,
      dirId,
      options,
      knownDirectoryPaths.value,
    );
    currentDir.value = data?.dirId || dirId;
    parentId.value = data?.parentId || "";
    currentPath.value = resolvedPath;
    items.value = (data?.items || []).map(normalizeItem);
    directoryTotal.value = Number(data?.count || items.value.length);
    directoryPage.value =
      Math.floor(
        Number(data?.offset || 0) / Number(data?.limit || pageSize.value || 1),
      ) + 1;
    rememberKnownPath(currentDir.value, resolvedPath);
    syncSelection(previousKey);
  } catch (error) {
    const message = sanitizeErrorMessage(error?.message || error);
    const temporaryRemoteError =
      message.includes("115 接口暂时返回了异常页面") ||
      message.includes("远程接口暂时返回了异常页面");

    if (dirId !== "0" && !temporaryRemoteError) {
      showNotice("warning", "目录不可用，已自动回到根目录。");
      await loadDirectory("0", {}, 1);
      return;
    }
    showError(error, "读取目录失败");
  } finally {
    directoryLoading.value = false;
  }
}

async function loadFolderPicker(dirId) {
  folderPickerLoading.value = true;

  try {
    const data = await PreviewDirectory(dirId);
    const resolvedPath = resolveDirectoryPath(
      data,
      dirId,
      {},
      knownDirectoryPaths.value,
    );
    folderPickerDirId.value = data?.dirId || dirId;
    folderPickerPath.value = resolvedPath;
    rememberKnownPath(folderPickerDirId.value, resolvedPath);
    folderPickerFolders.value = (data?.items || [])
      .filter((item) => item.isDirectory)
      .map(normalizeItem);
  } catch (error) {
    const message = sanitizeErrorMessage(error?.message || error);
    if (
      message.includes("115 接口暂时返回了异常页面") ||
      message.includes("远程接口暂时返回了异常页面")
    ) {
      showNotice("warning", "115 目录接口暂时波动，请稍后重试。");
    } else {
      showError(error, "读取目录失败");
    }
  } finally {
    folderPickerLoading.value = false;
  }
}

async function performGlobalSearch(
  keyword,
  requestId = searchRequestId.value,
  page = searchPage.value,
) {
  if (!loggedIn.value) {
    searchLoading.value = false;
    searchResults.value = [];
    searchTotal.value = 0;
    return;
  }

  searchLoading.value = true;
  const nextPage = Math.max(1, Number(page || 1));
  const offset = (nextPage - 1) * pageSize.value;

  try {
    const data = await SearchFiles(keyword, offset, pageSize.value);
    if (
      requestId !== searchRequestId.value ||
      keyword !== searchQuery.value.trim()
    ) {
      return;
    }

    searchResults.value = (data?.items || []).map(normalizeItem);
    searchTotal.value = Number(data?.count || searchResults.value.length);
    searchPage.value =
      Math.floor(
        Number(data?.offset || 0) / Number(data?.limit || pageSize.value || 1),
      ) + 1;
    syncSelection(selectedItem.value?.rowKey ?? "");
  } catch (error) {
    if (requestId !== searchRequestId.value) {
      return;
    }
    searchResults.value = [];
    searchTotal.value = 0;
    throw error;
  } finally {
    if (requestId === searchRequestId.value) {
      searchLoading.value = false;
    }
  }
}

function clearSearchDebounce() {
  if (searchDebounceTimer.value) {
    window.clearTimeout(searchDebounceTimer.value);
    searchDebounceTimer.value = null;
  }
}

function syncSelection(preferredKey) {
  const baseList = isGlobalSearchActive.value
    ? searchResults.value
    : items.value;
  if (!baseList.length) {
    selectedItem.value = null;
    detailDrawer.value = false;
    return;
  }

  const next =
    baseList.find((item) => item.rowKey === preferredKey) ??
    filteredItems.value[0] ??
    baseList[0];

  selectedItem.value = next ?? null;
}

function resetDirectoryState() {
  currentDir.value = "0";
  parentId.value = "";
  currentPath.value = [rootBreadcrumb()];
  items.value = [];
  directoryTotal.value = 0;
  directoryPage.value = 1;
  searchResults.value = [];
  searchTotal.value = 0;
  searchPage.value = 1;
  selectedItem.value = null;
  detailDrawer.value = false;
  offlineTargetDir.value = createDirectoryTarget("0", [rootBreadcrumb()]);
  knownDirectoryPaths.value = {
    0: [rootBreadcrumb()],
  };
}

function rememberKnownPath(dirId, path) {
  const normalizedPath = normalizeBreadcrumbPath(path);
  if (!dirId || !normalizedPath.length) {
    return;
  }

  knownDirectoryPaths.value = {
    ...knownDirectoryPaths.value,
    [String(dirId)]: normalizedPath,
  };
}

async function handleLogout() {
  actionLoading.value = true;

  try {
    await logoutSession();
  } finally {
    actionLoading.value = false;
  }
}

function openDetails(item) {
  selectedItem.value = item;
  detailDrawer.value = true;
}

async function handlePrimaryAction(item) {
  if (!item) {
    return;
  }

  if (item.isDirectory) {
    detailDrawer.value = false;
    if (isGlobalSearchActive.value) {
      searchQuery.value = "";
      await nextTick();
      await loadDirectory(
        item.fileId,
        {
          fallbackName: item.name,
        },
        1,
      );
      return;
    }
    await loadDirectory(
      item.fileId,
      {
        fallbackName: item.name,
        fallbackPath: [
          ...breadcrumbItems.value,
          { id: item.fileId, name: item.name },
        ],
      },
      1,
    );
    return;
  }

  if (!item.isVideo) {
    openDetails(item);
    showNotice("info", "当前文件不是常见视频格式，暂不支持直接播放。");
    return;
  }

  await playVideo(item);
}

async function playVideo(item, options = {}) {
  if (!item?.isVideo) {
    return;
  }

  actionLoading.value = true;
  selectedItem.value = item;

  try {
    const result = await PlayFile({
      pickCode: item.pickCode,
      name: item.name,
      startMs: options.startMs || 0,
      fromStart: Boolean(options.fromStart),
      subtitle: options.subtitle || item.subtitlePath || "",
    });

    applyPlaybackState(item.pickCode, {
      resumeMs: options.fromStart ? 0 : result?.startMs || item.resumeMs || 0,
      subtitlePath:
        typeof result?.subtitle === "string"
          ? result.subtitle
          : item.subtitlePath || "",
      lastPlayedAt: new Date().toISOString(),
    });

    const segments = [`已交给 ${result?.playerName || "播放器"}`];
    if (result?.resumeUsed && result?.startMs > 0) {
      segments.push(`从 ${formatDurationMs(result.startMs)} 继续`);
    } else if (result?.startMs > 0) {
      segments.push(`从 ${formatDurationMs(result.startMs)} 开始`);
    }
    if (result?.subtitle) {
      segments.push(`字幕 ${basename(result.subtitle)}`);
    }

    showNotice("success", segments.join(" · "), 4200);
  } catch (error) {
    const fallbackOpened = await openBuiltinPlayer(item, options, {
      fallback: true,
      loading: false,
    });
    if (fallbackOpened) {
      showNotice("warning", "本地播放器启动失败，已切换到内置播放器。", 5200);
    } else {
      showError(error, "启动播放失败");
    }
  } finally {
    actionLoading.value = false;
  }
}

async function openBuiltinPlayer(
  item = selectedItem.value,
  options = {},
  meta = {},
) {
  if (!item?.isVideo) {
    return false;
  }

  if (meta.loading !== false) {
    actionLoading.value = true;
  }
  builtinPlayerLoading.value = true;
  selectedItem.value = item;

  try {
    const result = await PrepareBuiltinPlayback({
      pickCode: item.pickCode,
      name: item.name,
      startMs: options.startMs || 0,
      fromStart: Boolean(options.fromStart),
      subtitle: options.subtitle || item.subtitlePath || "",
    });

    builtinPlayer.value = {
      pickCode: item.pickCode,
      rowKey: item.rowKey,
      title: result?.title || item.displayName || item.name || "PanPlayer 115",
      url: result?.url || "",
      startMs: Number(result?.startMs || 0),
      resumeUsed: Boolean(result?.resumeUsed),
      subtitleUrl: result?.subtitleUrl || "",
      subtitleName: result?.subtitleName || "",
      subtitlePath: result?.subtitlePath || item.subtitlePath || "",
      subtitleType: result?.subtitleType || "",
      subtitleUsable: Boolean(result?.subtitleUsable),
      currentMs: Number(result?.startMs || 0),
      ended: false,
    };
    builtinPlayerSeekKey.value = "";
    builtinPlayerDialog.value = true;

    applyPlaybackState(item.pickCode, {
      resumeMs: options.fromStart ? 0 : result?.startMs || item.resumeMs || 0,
      subtitlePath:
        typeof result?.subtitlePath === "string"
          ? result.subtitlePath
          : item.subtitlePath || "",
      lastPlayedAt: new Date().toISOString(),
    });

    if (result?.subtitlePath && !result?.subtitleUsable) {
      showNotice("warning", "内置播放器暂不支持当前字幕格式。");
    } else if (!meta.fallback && !meta.silent) {
      showNotice("success", "已打开内置播放器。");
    }

    return true;
  } catch (error) {
    if (!meta.fallback) {
      showError(error, "内置播放器启动失败");
    }
    return false;
  } finally {
    builtinPlayerLoading.value = false;
    if (meta.loading !== false) {
      actionLoading.value = false;
    }
  }
}

function handleBuiltinCanPlay(player) {
  const startMS = Number(builtinPlayer.value?.startMs || 0);
  const key = `${builtinPlayer.value?.pickCode || ""}:${startMS}`;
  if (!player || startMS <= 0 || builtinPlayerSeekKey.value === key) {
    return;
  }

  player.currentTime = startMS / 1000;
  builtinPlayerSeekKey.value = key;
}

function handleBuiltinTimeUpdate(event) {
  const currentTime = Number(event?.detail?.currentTime || 0);
  if (!Number.isFinite(currentTime)) {
    return;
  }
  builtinPlayer.value = {
    ...builtinPlayer.value,
    currentMs: Math.max(0, Math.floor(currentTime * 1000)),
  };
}

function handleBuiltinEnded() {
  builtinPlayer.value = {
    ...builtinPlayer.value,
    currentMs: 0,
    ended: true,
  };
  persistBuiltinProgress();
}

function handleBuiltinError() {
  showNotice(
    "warning",
    "内置播放器无法播放当前视频，请尝试 mpv 或 VLC。",
    5200,
  );
}

function closeBuiltinPlayer(playerElement = null) {
  persistBuiltinProgress();
  if (playerElement?.pause) {
    playerElement.pause();
  }
  builtinPlayerDialog.value = false;
  builtinPlayer.value = createEmptyBuiltinPlayer();
}

async function playExternalFromBuiltin() {
  const item = selectedItem.value;
  closeBuiltinPlayer();
  await playVideo(item);
}

async function persistBuiltinProgress() {
  const source = builtinPlayer.value;
  if (!source?.pickCode) {
    return;
  }

  const positionMS = source.ended
    ? 0
    : Math.max(0, Number(source.currentMs || 0));
  try {
    const result = await SavePlaybackProgress(
      source.pickCode,
      source.title || "",
      positionMS,
    );
    applyPlaybackState(source.pickCode, result);
  } catch (error) {
    showError(error, "保存播放进度失败");
  }
}

async function chooseSubtitle(item = selectedItem.value) {
  if (!item?.isVideo) {
    return;
  }

  actionLoading.value = true;

  try {
    const result = await SelectSubtitlePath(item.pickCode);
    applyPlaybackState(item.pickCode, result);
    showNotice("success", result?.subtitleName || "外挂字幕路径已保存。");
    if (
      builtinPlayerDialog.value &&
      builtinPlayer.value.pickCode === item.pickCode
    ) {
      const refreshedItem =
        selectedItem.value?.pickCode === item.pickCode
          ? selectedItem.value
          : normalizeItem({
              ...item,
              subtitlePath: result?.subtitlePath || "",
            });
      await openBuiltinPlayer(
        refreshedItem,
        {
          startMs: builtinPlayer.value.currentMs || 0,
        },
        {
          loading: false,
          silent: true,
        },
      );
    }
  } catch (error) {
    showError(error, "选择字幕失败");
  } finally {
    actionLoading.value = false;
  }
}

async function clearSubtitle(item = selectedItem.value) {
  if (!item?.isVideo) {
    return;
  }

  actionLoading.value = true;

  try {
    const result = await ClearSubtitlePath(item.pickCode);
    applyPlaybackState(item.pickCode, result);
    showNotice(
      "success",
      `已移除 ${item.displayName || item.name} 的字幕绑定。`,
    );
    if (
      builtinPlayerDialog.value &&
      builtinPlayer.value.pickCode === item.pickCode
    ) {
      builtinPlayer.value = {
        ...builtinPlayer.value,
        subtitleUrl: "",
        subtitleName: "",
        subtitlePath: "",
        subtitleType: "",
        subtitleUsable: false,
      };
    }
  } catch (error) {
    showError(error, "清除字幕失败");
  } finally {
    actionLoading.value = false;
  }
}

async function clearProgress(item = selectedItem.value) {
  if (!item?.isVideo) {
    return;
  }

  actionLoading.value = true;

  try {
    const result = await ClearPlaybackProgress(item.pickCode);
    applyPlaybackState(item.pickCode, result);
    showNotice(
      "success",
      `已清除 ${item.displayName || item.name} 的续播记录。`,
    );
  } catch (error) {
    showError(error, "清除续播失败");
  } finally {
    actionLoading.value = false;
  }
}

function applyPlaybackState(pickCode, patch) {
  if (!pickCode) {
    return;
  }

  const nextResumeMs = Number(patch?.resumeMs || 0);
  const nextSubtitlePath =
    patch && "subtitlePath" in patch ? patch.subtitlePath || "" : undefined;
  const nextLastPlayedAt =
    patch && "lastPlayedAt" in patch ? patch.lastPlayedAt || "" : undefined;

  items.value = items.value.map((entry) => {
    if (entry.pickCode !== pickCode) {
      return entry;
    }

    return normalizeItem({
      ...entry,
      resumeMs: nextResumeMs,
      subtitlePath:
        nextSubtitlePath !== undefined ? nextSubtitlePath : entry.subtitlePath,
      lastPlayedAt:
        nextLastPlayedAt !== undefined ? nextLastPlayedAt : entry.lastPlayedAt,
    });
  });

  if (selectedItem.value?.pickCode === pickCode) {
    const matched = items.value.find((entry) => entry.pickCode === pickCode);
    selectedItem.value = matched ?? selectedItem.value;
  }
}

function openJumpDialog(item = selectedItem.value) {
  if (!item?.isVideo) {
    return;
  }

  jumpTargetKey.value = item.rowKey;
  jumpInput.value = item.resumeMs ? formatDurationMs(item.resumeMs) : "";
  jumpDialogVisible.value = true;
}

async function confirmJump() {
  const item = items.value.find(
    (entry) => entry.rowKey === jumpTargetKey.value,
  );
  if (!item) {
    jumpDialogVisible.value = false;
    return;
  }

  const parsed = parseTimeInput(jumpInput.value);
  if (parsed === null) {
    showNotice("warning", "时间格式不正确，支持 90、01:30、01:02:03。");
    return;
  }

  jumpDialogVisible.value = false;
  await playVideo(item, { startMs: parsed });
}

async function reloadCurrentDirectory() {
  if (!loggedIn.value) {
    showNotice("info", "请先登录 115 账号。");
    return;
  }
  if (isGlobalSearchActive.value) {
    clearSearchDebounce();
    searchRequestId.value += 1;
    await performGlobalSearch(
      searchQuery.value.trim(),
      searchRequestId.value,
      searchPage.value,
    );
    return;
  }
  await loadDirectory(
    currentDir.value,
    {
      fallbackName:
        breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name ||
        "我的文件",
      fallbackPath: breadcrumbItems.value,
    },
    directoryPage.value,
  );
}

async function triggerSearchNow() {
  const keyword = searchQuery.value.trim();
  if (!keyword || !loggedIn.value) {
    return;
  }
  clearSearchDebounce();
  searchRequestId.value += 1;
  await performGlobalSearch(keyword, searchRequestId.value, searchPage.value);
}

function openSearchInput() {
  if (!loggedIn.value) {
    return;
  }

  searchExpanded.value = true;
}

function closeSearchInput() {
  clearSearchDebounce();
  searchRequestId.value += 1;
  searchQuery.value = "";
  searchResults.value = [];
  searchTotal.value = 0;
  searchPage.value = 1;
  searchExpanded.value = false;
}

function toggleSearchInput() {
  if (!loggedIn.value) {
    return;
  }
  if (isSearchInputVisible.value) {
    closeSearchInput();
    return;
  }
  openSearchInput();
}

function handleSearchBlur() {
  window.setTimeout(() => {
    if (!searchQuery.value.trim()) {
      searchExpanded.value = false;
    }
  }, 120);
}

function handleSearchClear() {
  closeSearchInput();
}

async function handlePageChange(page) {
  const nextPage = Math.max(1, Number(page || 1));
  if (isGlobalSearchActive.value) {
    if (nextPage === searchPage.value) {
      return;
    }
    searchRequestId.value += 1;
    await performGlobalSearch(
      searchQuery.value.trim(),
      searchRequestId.value,
      nextPage,
    );
    return;
  }

  if (nextPage === directoryPage.value) {
    return;
  }
  await loadDirectory(
    currentDir.value,
    {
      fallbackName:
        breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name ||
        "我的文件",
      fallbackPath: breadcrumbItems.value,
    },
    nextPage,
  );
}

async function handlePageSizeChange(value) {
  const nextSize = Number(value || 20);
  if (!nextSize || nextSize === pageSize.value) {
    return;
  }

  pageSize.value = nextSize;
  if (isGlobalSearchActive.value) {
    searchRequestId.value += 1;
    await performGlobalSearch(
      searchQuery.value.trim(),
      searchRequestId.value,
      1,
    );
    return;
  }

  await loadDirectory(
    currentDir.value,
    {
      fallbackName:
        breadcrumbItems.value[breadcrumbItems.value.length - 1]?.name ||
        "我的文件",
      fallbackPath: breadcrumbItems.value,
    },
    1,
  );
}

function handleFileListDensityInput(value) {
  saveFileListDensity(value).catch((error) =>
    showError(error, "保存文件列表密度失败"),
  );
}

function handleDownloadPageChange(page) {
  const nextPage = Math.max(1, Number(page || 1));
  if (nextPage === downloadPage.value) {
    return;
  }
  downloadPage.value = Math.min(nextPage, downloadPageCount.value);
}

function handleDownloadPageSizeChange(value) {
  const nextSize = Number(value || 20);
  if (!nextSize || nextSize === downloadPageSize.value) {
    return;
  }

  downloadPageSize.value = nextSize;
  downloadPage.value = 1;
}

async function openBreadcrumb(dirId) {
  if (!dirId || dirId === currentDir.value) {
    return;
  }
  const fallbackPath = breadcrumbPathUntil(breadcrumbItems.value, dirId);
  await loadDirectory(
    dirId,
    {
      fallbackName: fallbackPath[fallbackPath.length - 1]?.name || "我的文件",
      fallbackPath,
    },
    1,
  );
}

async function refreshOfflineTasks() {
  if (!loggedIn.value) {
    return;
  }

  downloadLoading.value = true;

  try {
    const data = await ListOfflineTasks();
    downloadState.value = normalizeOfflineState(data);
    downloadPage.value = 1;
  } catch (error) {
    showError(error, "读取云下载失败");
  } finally {
    downloadLoading.value = false;
  }
}

function openDownloadDialog() {
  if (!loggedIn.value) {
    showNotice("info", "请先登录 115 账号。");
    return;
  }

  const recentTarget = offlineRecentTargets.value[0];
  offlineTargetDir.value = recentTarget
    ? createDirectoryTarget(recentTarget.id, recentTarget.path)
    : createDirectoryTarget(currentDir.value || "0", breadcrumbItems.value);
  downloadDialog.value = true;
}

function closeDownloadDialog() {
  downloadDialog.value = false;
}

async function selectTorrentFile() {
  torrentSelecting.value = true;

  try {
    const magnet = await SelectTorrentFileAsMagnet();
    const normalized = String(magnet || "").trim();
    if (!normalized) {
      return;
    }

    downloadInput.value = downloadInput.value.trim()
      ? `${downloadInput.value.trim()}\n${normalized}`
      : normalized;
    showNotice("success", "BT 种子已转换为磁力链接并填入输入框。");
  } catch (error) {
    showError(error, "导入 BT 种子失败");
  } finally {
    torrentSelecting.value = false;
  }
}

async function submitOfflineTasks() {
  const urls = normalizeMultilineInput(downloadInput.value);
  if (!urls.length) {
    showNotice("warning", "请至少输入一个下载链接。");
    return;
  }

  downloadSubmitting.value = true;

  try {
    const data = await AddOfflineTasks({
      urls,
      saveDirId: offlineTargetDir.value?.id || "0",
      saveDirPath: offlineTargetDir.value?.path || [rootBreadcrumb()],
    });

    downloadState.value = normalizeOfflineState(data);
    downloadPage.value = 1;
    settings.value = {
      ...settings.value,
      offlineRecentTargets: rememberOfflineRecentTargets(
        settings.value?.offlineRecentTargets,
        offlineTargetDir.value,
      ),
    };
    downloadDialog.value = false;
    downloadInput.value = "";
    activeSection.value = "downloads";
    downloadFilter.value = "active";
    showNotice("success", `已添加 ${urls.length} 个云下载任务。`);
  } catch (error) {
    showError(error, "添加云下载失败");
  } finally {
    downloadSubmitting.value = false;
  }
}

async function deleteOfflineTask(task) {
  if (!task?.infoHash) {
    return;
  }
  await deleteOfflineHashes([task.infoHash]);
}

async function deleteOfflineHashes(hashes) {
  downloadSubmitting.value = true;

  try {
    const data = await DeleteOfflineTasks({
      hashes,
      deleteFiles: downloadDeleteFiles.value,
    });
    downloadState.value = normalizeOfflineState(data);
    downloadPage.value = 1;
    showNotice("success", "离线下载任务已删除。");
  } catch (error) {
    showError(error, "删除云下载任务失败");
  } finally {
    downloadSubmitting.value = false;
  }
}

async function copyOfflineURL(task) {
  if (!task?.url) {
    showNotice("info", "当前任务没有可复制的链接。");
    return;
  }

  try {
    await Clipboard.SetText(task.url);
    showNotice("success", "任务链接已复制到剪贴板。");
  } catch (error) {
    showError(error, "复制任务链接失败");
  }
}

async function openOfflineDirectory(task) {
  if (!task?.dirId) {
    showNotice("warning", "当前任务还没有可打开的目录。");
    return;
  }

  activeSection.value = "files";
  await loadDirectory(task.dirId, {}, 1);
}

function openFolderPicker() {
  folderPickerDialog.value = true;
  loadFolderPicker(offlineTargetDir.value?.id || currentDir.value || "0");
}

function closeFolderPicker() {
  folderPickerDialog.value = false;
}

async function openFolderPickerBreadcrumb(dirId) {
  if (!dirId || dirId === folderPickerDirId.value) {
    return;
  }
  await loadFolderPicker(dirId);
}

async function openFolderPickerDirectory(folder) {
  if (!folder?.fileId) {
    return;
  }
  await loadFolderPicker(folder.fileId);
}

function chooseFolderPickerCurrent() {
  offlineTargetDir.value = createDirectoryTarget(
    folderPickerDirId.value,
    folderPickerPath.value,
  );
  folderPickerDialog.value = false;
}

function handleOfflineTargetSelect(value) {
  const option = offlineTargetSelectOptions.value.find(
    (item) => item.value === value,
  );
  if (!option) {
    return;
  }

  if (option.isPicker) {
    openFolderPicker();
    return;
  }

  if (option.target) {
    offlineTargetDir.value = createDirectoryTarget(
      option.target.id,
      option.target.path,
    );
  }
}

function handleKeydown(event) {
  if (event.key === "Escape" && folderPickerDialog.value) {
    event.preventDefault();
    closeFolderPicker();
    return;
  }

  if (event.key === "Escape" && downloadDialog.value) {
    event.preventDefault();
    closeDownloadDialog();
    return;
  }

  if (event.key === "Escape" && builtinPlayerDialog.value) {
    event.preventDefault();
    closeBuiltinPlayer();
    return;
  }

  if (
    (event.ctrlKey || event.metaKey) &&
    event.key.toLowerCase() === "f" &&
    activeSection.value === "files"
  ) {
    event.preventDefault();
    openSearchInput();
    return;
  }

  if (
    event.key === "Escape" &&
    activeSection.value === "files" &&
    isSearchInputVisible.value
  ) {
    closeSearchInput();
    return;
  }

  if (event.key === "Escape" && detailDrawer.value) {
    detailDrawer.value = false;
  }
}

function normalizeItem(item) {
  return normalizeFileItem(item, {
    showTitleBadges: settings.value?.showTitleBadges,
  });
}

function refreshFilePresentation() {
  const selectedKey = selectedItem.value?.rowKey;
  items.value = items.value.map((item) =>
    normalizeItem({
      ...item,
      name: item.originalName || item.name,
    }),
  );

  if (selectedKey) {
    selectedItem.value =
      items.value.find((item) => item.rowKey === selectedKey) ??
      selectedItem.value;
  }
}

function sleep(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
</script>

<template>
  <v-app>
    <v-layout class="app-shell">
      <AppSidebar
        v-model:active-section="activeSection"
        :navigation-items="navigationItems"
        :logged-in="loggedIn"
        :user="user"
        :account-avatar-url="accountAvatarUrl"
        :account-display-name="accountDisplayName"
        :account-vip-level-text="accountVipLevelText"
        :account-vip-expire-text="accountVipExpireText"
        :account-space-usage-text="accountSpaceUsageText"
        :account-space-percent-text="accountSpacePercentText"
        :account-space-percent="accountSpacePercent"
        :action-loading="actionLoading"
        @login="openLoginDialog"
        @logout="handleLogout"
      />

      <v-main class="app-main">
        <div class="workspace">
          <template v-if="booting">
            <v-card class="state-shell" variant="outlined">
              <v-progress-circular indeterminate color="primary" size="42" />
              <div class="text-subtitle-1 mt-4">正在初始化 115 会话</div>
              <div class="text-body-2 text-medium-emphasis">
                启动时会自动恢复本地 SQLite 中保存的登录状态。
              </div>
            </v-card>
          </template>

          <template v-else>
            <FileManagerPage
              v-if="activeSection === 'files'"
              v-model:search-query="searchQuery"
              v-model:type-filter="typeFilter"
              v-model:sort-mode="sortMode"
              v-model:filter-menu-open="filterMenuOpen"
              :logged-in="loggedIn"
              :directory-loading="directoryLoading"
              :action-loading="actionLoading"
              :search-loading="searchLoading"
              :is-global-search-active="isGlobalSearchActive"
              :search-summary-text="searchSummaryText"
              :breadcrumb-display-items="breadcrumbDisplayItems"
              :search-input-visible="isSearchInputVisible"
              :small-file-filter-m-b="smallFileFilterMB"
              :settings="settings"
              :type-options="typeOptions"
              :sort-options="sortOptions"
              :small-file-filter-options="smallFileFilterOptions"
              :file-list-density-options="fileListDensityOptions"
              :items="filteredItems"
              :file-empty-text="fileEmptyText"
              :selected-item="selectedItem"
              :file-list-density-class="fileListDensityClass"
              :file-list-avatar-size="fileListAvatarSize"
              :file-list-icon-size="fileListIconSize"
              :show-pagination="showPagination"
              :pagination-summary-text="paginationSummaryText"
              :page-size="pageSize"
              :page-count="pageCount"
              :active-page="activePage"
              :page-size-options="pageSizeOptions"
              @open-login="openLoginDialog"
              @open-breadcrumb="openBreadcrumb"
              @reload="reloadCurrentDirectory"
              @toggle-search="toggleSearchInput"
              @search-blur="handleSearchBlur"
              @search-clear="handleSearchClear"
              @trigger-search="triggerSearchNow"
              @save-small-file-filter="saveSmallFileFilter"
              @save-file-list-density="handleFileListDensityInput"
              @save-show-title-badges="saveShowTitleBadges"
              @open-details="openDetails"
              @primary-action="handlePrimaryAction"
              @page-size-change="handlePageSizeChange"
              @page-change="handlePageChange"
            />

            <DownloadsPage
              v-else-if="activeSection === 'downloads'"
              v-model:download-filter="downloadFilter"
              :logged-in="loggedIn"
              :download-loading="downloadLoading"
              :download-submitting="downloadSubmitting"
              :download-filter-options="downloadFilterOptions"
              :download-quota-text="downloadQuotaText"
              :download-quota-progress="downloadQuotaProgress"
              :file-list-density-class="fileListDensityClass"
              :file-list-avatar-size="fileListAvatarSize"
              :file-list-icon-size="fileListIconSize"
              :tasks="presentedDownloadTasks"
              :download-empty-text="downloadEmptyText"
              :show-download-pagination="showDownloadPagination"
              :download-pagination-summary-text="downloadPaginationSummaryText"
              :page-size="downloadPageSize"
              :page-size-options="pageSizeOptions"
              :download-page-count="downloadPageCount"
              :download-page="downloadPage"
              @open-download-dialog="openDownloadDialog"
              @refresh="refreshOfflineTasks"
              @open-directory="openOfflineDirectory"
              @copy-url="copyOfflineURL"
              @delete-task="deleteOfflineTask"
              @page-size-change="handleDownloadPageSizeChange"
              @page-change="handleDownloadPageChange"
            />

            <SettingsPage
              v-else-if="activeSection === 'settings'"
              v-model:settings-tab="settingsTab"
              :settings="settings"
              :player-options="playerOptions"
              :player-loading="playerLoading"
              @select-player="selectPlayerFromList"
              @choose-player-path="choosePlayerPath"
              @toggle-player-disabled="togglePlayerDisabled"
              @delete-player="deletePlayer"
            />
          </template>

          <NoticeSnackbar :notice="notice" @close="closeNotice" />
        </div>
      </v-main>

      <FileDetailDrawer
        v-model="detailDrawer"
        :selected-item="selectedItem"
        :selected-resume-text="selectedResumeText"
        :selected-subtitle-name="selectedSubtitleName"
        :selected-last-played-text="selectedLastPlayedText"
        @primary-action="handlePrimaryAction"
        @play="playVideo"
        @builtin-play="openBuiltinPlayer"
        @play-from-start="(item) => playVideo(item, { fromStart: true })"
        @jump="openJumpDialog"
        @choose-subtitle="chooseSubtitle"
        @clear-subtitle="clearSubtitle"
        @clear-progress="clearProgress"
      />
    </v-layout>

    <LoginDialog
      v-model="loginDialog"
      v-model:login-tab="loginTab"
      v-model:cookie-input="cookieInput"
      :login-loading="loginLoading"
      :qr-image="qrImage"
      :login-status="loginStatus"
      :cookie-submitting="cookieSubmitting"
      @close="closeLoginDialog"
      @start-login="startLogin"
      @paste-cookie="pasteCookieFromClipboard"
      @submit-cookie="submitCookieLogin"
    />

    <DownloadDialog
      v-model="downloadDialog"
      v-model:download-input="downloadInput"
      :download-submitting="downloadSubmitting"
      :torrent-selecting="torrentSelecting"
      :offline-target-select-options="offlineTargetSelectOptions"
      :offline-target-select-value="offlineTargetSelectValue"
      :offline-target-path-text="offlineTargetPathText"
      @close="closeDownloadDialog"
      @select-torrent="selectTorrentFile"
      @submit="submitOfflineTasks"
      @select-target="handleOfflineTargetSelect"
    />

    <FolderPickerDialog
      v-model="folderPickerDialog"
      :breadcrumb-items="folderPickerBreadcrumbDisplayItems"
      :loading="folderPickerLoading"
      :folders="folderPickerFolders"
      @close="closeFolderPicker"
      @choose-current="chooseFolderPickerCurrent"
      @open-breadcrumb="openFolderPickerBreadcrumb"
      @open-directory="openFolderPickerDirectory"
    />

    <BuiltinPlayerDialog
      v-model="builtinPlayerDialog"
      :player="builtinPlayer"
      :loading="builtinPlayerLoading"
      :text-tracks="builtinTextTracks"
      :selected-item="selectedItem"
      @close="closeBuiltinPlayer"
      @choose-subtitle="chooseSubtitle"
      @external-play="playExternalFromBuiltin"
      @can-play="handleBuiltinCanPlay"
      @time-update="handleBuiltinTimeUpdate"
      @ended="handleBuiltinEnded"
      @error="handleBuiltinError"
    />

    <JumpDialog
      v-model="jumpDialogVisible"
      v-model:jump-input="jumpInput"
      @confirm="confirmJump"
    />
  </v-app>
</template>
