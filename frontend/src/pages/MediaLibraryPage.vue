<script setup>
import { computed, onMounted, ref, watch } from "vue";
import {
  GetLibraryDetail,
  GetLibrarySnapshot,
} from "../../bindings/panplayer/app";
import LibraryGrid from "../components/library/LibraryGrid.vue";
import { sanitizeErrorMessage } from "../utils/error";
import {
  buildLibraryImageStyle,
  escapeLibraryAssetUrl,
  normalizeLibraryAssetUrl,
} from "../utils/libraryAssets";

const props = defineProps({
  scraperStatus: {
    type: Object,
    default: null,
  },
  scraperActionLoading: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["play", "start-scraper", "pause-scraper"]);

const primarySectionIds = [
  "movies",
  "series",
  "variety",
  "anime",
  "documentary",
];

const librarySections = ref([]);
const libraryItems = ref([]);
const selectedItem = ref(null);
const selectedDetail = ref(null);
const activeSection = ref("all");
const activeChild = ref("");
const activeChildSection = ref("");
const filterOpen = ref(false);
const filterExpanded = ref(false);
const filterDragging = ref(false);
const filterStripRef = ref(null);
const historyOnly = ref(false);
const activeSeason = ref(1);
const loading = ref(false);
const detailLoading = ref(false);

let filterDragPointerId = null;
let filterDragStartX = 0;
let filterDragScrollLeft = 0;
let suppressNextFilterClick = false;

const topNavSections = computed(() => [
  { id: "all", label: "全部", icon: "mdi-view-grid-outline" },
  ...librarySections.value.filter((section) =>
    primarySectionIds.includes(section.id),
  ),
]);

const selectedSection = computed(() => {
  if (activeSection.value === "all") {
    return null;
  }
  return (
    librarySections.value.find(
      (section) => section.id === activeSection.value,
    ) || null
  );
});

const filterGroups = computed(() => {
  if (activeSection.value === "all") {
    return librarySections.value.filter((section) =>
      primarySectionIds.includes(section.id),
    );
  }
  return selectedSection.value ? [selectedSection.value] : [];
});

const selectedMedia = computed(
  () => selectedDetail.value || selectedItem.value || null,
);

const selectedSectionLabel = computed(() => {
  const item = selectedMedia.value;
  if (!item) {
    return "";
  }
  return (
    librarySections.value.find((section) => section.id === item.section)
      ?.label || "影视"
  );
});

const selectedDetailTags = computed(() => {
  const item = selectedMedia.value;
  if (!item) {
    return [];
  }
  return [
    item.child,
    item.quality,
    item.source,
    item.audio,
    item.status,
  ].filter(Boolean);
});

const selectedSeasonItems = computed(() => {
  const detail = selectedDetail.value;
  if (!detail?.files?.length) {
    return [];
  }
  const seasons = [
    ...new Set(
      detail.files
        .map((item) => Number(item.seasonNumber || 1))
        .filter((item) => item > 0),
    ),
  ];
  if (!seasons.length) {
    return [];
  }
  return seasons.map((season) => ({
    value: season,
    title: `第${season}季`,
  }));
});

const selectedEpisodes = computed(() => {
  const detail = selectedDetail.value;
  if (!detail?.files?.length) {
    return [];
  }

  const hasSeason = selectedSeasonItems.value.length > 0;
  const seasonFiles = detail.files.filter(
    (item) =>
      (!hasSeason || Number(item.seasonNumber || 1) === activeSeason.value) &&
      item,
  );
  const filtered = seasonFiles.filter((item) => {
    const duration = Number(item.durationSec);
    if (Number.isFinite(duration)) {
      return duration > 0;
    }
    return true;
  });
  const baseFiles = filtered.length ? filtered : seasonFiles;
  const deduped = [];
  const episodeMap = new Map();
  const specials = [];

  baseFiles.forEach((item) => {
    const episodeNumber = Number(item.episodeNumber || 0);
    if (episodeNumber <= 0) {
      if (item.episodeTitle || item.name) {
        specials.push(item);
      }
      return;
    }
    const key = `${Number(item.seasonNumber || 1)}-${episodeNumber}`;
    if (!episodeMap.has(key)) {
      episodeMap.set(key, item);
      deduped.push(item);
    }
  });

  const ordered =
    deduped.length > 0 ? [...deduped, ...specials] : [...baseFiles];
  return ordered.map((item, index) => ({
    ...item,
    cardSubtitle: item.durationText || "",
    cardTitle:
      item.episodeTitle ||
      (item.episodeNumber > 0
        ? `第 ${item.episodeNumber} ${selectedMedia.value?.section === "variety" ? "期" : "集"}`
        : `第 ${index + 1} ${selectedMedia.value?.section === "variety" ? "期" : "集"}`),
  }));
});

const selectedCastItems = computed(() =>
  (
    (selectedMedia.value?.castMembers?.length
      ? selectedMedia.value.castMembers
      : (selectedMedia.value?.cast || []).map((name) => ({ name }))) || []
  ).map((person, index) => ({
    id: `${selectedMedia.value?.id || "detail"}-cast-${index}`,
    name: person.name,
    avatarUrl: person.avatarUrl || "",
    role:
      person.character ||
      person.role ||
      (selectedMedia.value?.section === "variety"
        ? index === 0
          ? "常驻嘉宾"
          : "嘉宾"
        : index === 0
          ? "主演"
          : "演员"),
    tone: ["amber", "steel", "rose", "green", "violet", "cyan"][index % 6],
  })),
);

const selectedResourcePaths = computed(() => {
  const values = [];
  const seen = new Set();

  const appendPath = (path) => {
    const normalized = (Array.isArray(path) ? path : [])
      .map((item) => item?.name || "")
      .filter(Boolean)
      .join(" / ");
    if (!normalized || seen.has(normalized)) {
      return;
    }
    seen.add(normalized);
    values.push(normalized);
  };

  appendPath(selectedMedia.value?.defaultPath);
  (selectedDetail.value?.files || []).forEach((item) => appendPath(item.path));

  return values;
});

const detailHeroStyle = computed(() => {
  const media = selectedMedia.value;
  if (!media) {
    return undefined;
  }
  const url = normalizeLibraryAssetUrl(media.backdropUrl || media.posterUrl);
  if (!url) {
    return undefined;
  }
  return {
    backgroundImage: [
      "linear-gradient(90deg, rgba(8, 15, 28, 0.86) 0%, rgba(8, 15, 28, 0.68) 34%, rgba(8, 15, 28, 0.38) 58%, rgba(8, 15, 28, 0.86) 100%)",
      "linear-gradient(to top, rgba(var(--v-theme-surface), 1) 0%, rgba(var(--v-theme-surface), 0.08) 46%, rgba(var(--v-theme-surface), 0) 100%)",
      `url("${escapeLibraryAssetUrl(url)}")`,
    ].join(", "),
  };
});

const filteredItems = computed(() => {
  return libraryItems.value.filter((item) => {
    if (historyOnly.value && item.progress <= 0) {
      return false;
    }

    if (
      !historyOnly.value &&
      activeSection.value !== "all" &&
      item.section !== activeSection.value
    ) {
      return false;
    }

    const child = activeChild.value;
    if (!child || child.startsWith("全部")) {
      return true;
    }
    if (child === "继续观看") {
      return item.progress > 0;
    }
    if (child === "已入库") {
      return item.status === "已入库" || item.status === "更新完成";
    }
    if (child === "高分收藏") {
      return Number(item.rating || 0) >= 8.5;
    }
    if (activeChildSection.value && item.section !== activeChildSection.value) {
      return false;
    }
    return item.child === child;
  });
});

const scraperRunning = computed(() => {
  const status = props.scraperStatus?.status;
  return status === "running" || status === "queued";
});

const visibleScraperStatus = computed(() => {
  const status = props.scraperStatus?.status;
  return (
    status === "running" ||
    status === "queued" ||
    status === "failed" ||
    status === "paused"
  );
});

const scraperProgressValue = computed(() => {
  const status = props.scraperStatus;
  if (!status) {
    return 0;
  }
  if (status.status === "completed") {
    return 100;
  }
  if (Number(status.discoveredFiles || 0) > 0) {
    return Math.min(
      100,
      Math.max(
        0,
        Math.round(
          (Number(status.processedFiles || 0) /
            Number(status.discoveredFiles || 1)) *
            100,
        ),
      ),
    );
  }
  if (status.totalDirectories > 0) {
    return Math.min(
      100,
      Math.max(
        0,
        Math.round(
          (Number(status.scannedDirectories || 0) /
            Number(status.totalDirectories || 1)) *
            100,
        ),
      ),
    );
  }
  return 0;
});

const scraperSummaryText = computed(() => {
  const status = props.scraperStatus;
  if (!status) {
    return "尚未开始刮削";
  }
  if (status.status === "running") {
    const totalFiles = Number(status.discoveredFiles || 0);
    if (totalFiles > 0) {
      return `${status.message || "正在刮削"} · ${status.processedFiles || 0}/${totalFiles}`;
    }
    return status.message || "正在刮削";
  }
  if (status.status === "failed") {
    return sanitizeErrorMessage(
      status.lastError || status.message || "刮削失败",
    );
  }
  if (status.status === "paused") {
    return sanitizeErrorMessage(status.message || "刮削已暂停");
  }
  return sanitizeErrorMessage(status.message || "刮削已完成");
});

watch(
  selectedSeasonItems,
  (items) => {
    if (!items.length) {
      activeSeason.value = 1;
      return;
    }
    const available = new Set(items.map((item) => item.value));
    if (!available.has(activeSeason.value)) {
      activeSeason.value = items[0].value;
    }
  },
  { immediate: true },
);

watch(
  () => props.scraperStatus,
  async (status, previous) => {
    const prevStatus = previous?.status;
    const nextStatus = status?.status;
    const previousRunning = prevStatus === "running" || prevStatus === "queued";
    const nextRunning = nextStatus === "running" || nextStatus === "queued";
    if (!previousRunning || nextRunning) {
      return;
    }
    await loadLibrarySnapshot();
    if (selectedItem.value?.id) {
      await openDetail(selectedItem.value);
    }
  },
  { deep: true },
);

onMounted(async () => {
  await loadLibrarySnapshot();
});

async function loadLibrarySnapshot() {
  loading.value = true;
  try {
    const data = await GetLibrarySnapshot();
    librarySections.value =
      Array.isArray(data?.sections) && data.sections.length
        ? data.sections
        : [];
    libraryItems.value = Array.isArray(data?.items) ? data.items : [];
    if (selectedItem.value?.id) {
      const next =
        libraryItems.value.find((item) => item.id === selectedItem.value.id) ||
        null;
      selectedItem.value = next;
      if (!next) {
        selectedDetail.value = null;
      }
    }
  } finally {
    loading.value = false;
  }
}

function selectSection(sectionId) {
  if (sectionId === "all") {
    activeSection.value = "all";
    activeChild.value = "";
    activeChildSection.value = "";
    historyOnly.value = false;
    return;
  }

  const section = librarySections.value.find((item) => item.id === sectionId);
  if (!section) {
    return;
  }
  activeSection.value = section.id;
  activeChild.value = "";
  activeChildSection.value = "";
  historyOnly.value = false;
}

function selectChild(sectionId, child) {
  activeSection.value = sectionId;
  activeChild.value = child;
  activeChildSection.value = sectionId;
  historyOnly.value = false;
}

function toggleHistory() {
  historyOnly.value = !historyOnly.value;
  if (historyOnly.value) {
    activeChild.value = "";
    activeChildSection.value = "";
  }
}

function isChildActive(sectionId, child) {
  return (
    activeSection.value === sectionId &&
    activeChild.value === child &&
    !historyOnly.value
  );
}

async function openDetail(item) {
  if (!item?.id) {
    return;
  }
  selectedItem.value = item;
  activeSeason.value = 1;
  detailLoading.value = true;
  try {
    selectedDetail.value = await GetLibraryDetail(item.id);
  } finally {
    detailLoading.value = false;
  }
}

function closeDetail() {
  selectedItem.value = null;
  selectedDetail.value = null;
}

function playSelected() {
  if (!selectedItem.value) {
    return;
  }
  emit("play", {
    titleId: selectedItem.value.id,
    fileId:
      selectedDetail.value?.defaultFileId ||
      selectedItem.value.defaultFileId ||
      "",
    startMs: 0,
  });
}

function playEpisode(episode) {
  if (!selectedItem.value || !episode?.fileId) {
    return;
  }
  emit("play", {
    titleId: selectedItem.value.id,
    fileId: episode.fileId,
    startMs: 0,
  });
}

function toggleFilterExpanded() {
  filterExpanded.value = !filterExpanded.value;
}

function handleFilterWheel(event) {
  if (filterExpanded.value) {
    return;
  }

  const el = filterStripRef.value;
  if (!el || el.scrollWidth <= el.clientWidth) {
    return;
  }

  const delta =
    Math.abs(event.deltaX) > Math.abs(event.deltaY)
      ? event.deltaX
      : event.deltaY;
  if (!delta) {
    return;
  }

  const before = el.scrollLeft;
  el.scrollLeft += delta;
  if (el.scrollLeft !== before) {
    event.preventDefault();
  }
}

function handleFilterPointerDown(event) {
  if (filterExpanded.value || event.button !== 0) {
    return;
  }

  const el = filterStripRef.value;
  if (!el || el.scrollWidth <= el.clientWidth) {
    return;
  }

  filterDragPointerId = event.pointerId;
  filterDragStartX = event.clientX;
  filterDragScrollLeft = el.scrollLeft;
  filterDragging.value = true;
  suppressNextFilterClick = false;
  el.setPointerCapture?.(event.pointerId);
}

function handleFilterPointerMove(event) {
  if (!filterDragging.value || event.pointerId !== filterDragPointerId) {
    return;
  }

  const el = filterStripRef.value;
  if (!el) {
    return;
  }

  const offset = event.clientX - filterDragStartX;
  if (Math.abs(offset) > 4) {
    suppressNextFilterClick = true;
    event.preventDefault();
  }
  el.scrollLeft = filterDragScrollLeft - offset;
}

function handleFilterPointerEnd(event) {
  if (!filterDragging.value || event.pointerId !== filterDragPointerId) {
    return;
  }

  filterStripRef.value?.releasePointerCapture?.(event.pointerId);
  filterDragging.value = false;
  filterDragPointerId = null;
}

function handleFilterClickCapture(event) {
  if (!suppressNextFilterClick) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();
  suppressNextFilterClick = false;
}
</script>

<template>
  <section class="page-section library-page">
    <div class="library-shell">
      <main class="library-content">
        <div class="library-nav-row">
          <div class="library-primary-nav" aria-label="影视库一级分类">
            <template v-if="selectedItem">
              <v-btn
                class="library-nav-button library-back-button"
                prepend-icon="mdi-arrow-left"
                rounded="pill"
                size="small"
                variant="text"
                @click="closeDetail"
              >
                返回
              </v-btn>
            </template>

            <template v-else>
              <v-btn
                v-for="section in topNavSections"
                :key="section.id"
                class="library-nav-button"
                :class="{
                  'library-nav-button--active':
                    activeSection === section.id && !historyOnly,
                }"
                :color="
                  activeSection === section.id && !historyOnly
                    ? 'primary'
                    : undefined
                "
                :prepend-icon="section.icon"
                :variant="
                  activeSection === section.id && !historyOnly ? 'flat' : 'text'
                "
                rounded="pill"
                size="small"
                @click="selectSection(section.id)"
              >
                {{ section.label }}
              </v-btn>
            </template>
          </div>

          <div v-if="!selectedItem" class="library-nav-actions">
            <v-tooltip text="观看历史" location="bottom">
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  class="library-action-button"
                  :color="historyOnly ? 'primary' : undefined"
                  icon="mdi-history"
                  :variant="historyOnly ? 'tonal' : 'text'"
                  size="small"
                  @click="toggleHistory"
                />
              </template>
            </v-tooltip>

            <v-tooltip text="分类筛选" location="bottom">
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  class="library-action-button"
                  :color="filterOpen ? 'primary' : undefined"
                  :icon="
                    filterOpen
                      ? 'mdi-filter-variant-minus'
                      : 'mdi-filter-variant'
                  "
                  :variant="filterOpen ? 'tonal' : 'text'"
                  size="small"
                  @click="filterOpen = !filterOpen"
                />
              </template>
            </v-tooltip>
          </div>
        </div>

        <div
          v-if="visibleScraperStatus"
          class="library-scraper-status"
          :class="{ 'library-scraper-status--running': scraperRunning }"
        >
          <div class="library-scraper-header">
            <div class="library-scraper-copy">
              <div class="library-scraper-title">{{ scraperSummaryText }}</div>
              <div class="library-scraper-meta" v-if="props.scraperStatus">
                已发现 {{ props.scraperStatus.discoveredFiles || 0 }} 个文件 ·
                已入库 {{ props.scraperStatus.updatedTitles || 0 }} 个条目
              </div>
            </div>
            <div class="library-scraper-actions">
              <v-btn
                v-if="scraperRunning"
                color="primary"
                prepend-icon="mdi-pause-circle-outline"
                rounded="pill"
                size="small"
                variant="tonal"
                :loading="props.scraperActionLoading"
                @click="emit('pause-scraper')"
              >
                暂停
              </v-btn>
              <v-btn
                v-else
                color="primary"
                prepend-icon="mdi-play-circle-outline"
                rounded="pill"
                size="small"
                variant="tonal"
                :loading="props.scraperActionLoading"
                @click="emit('start-scraper')"
              >
                开始刮削
              </v-btn>
            </div>
          </div>
          <v-progress-linear
            class="library-scraper-progress"
            :indeterminate="scraperRunning && scraperProgressValue === 0"
            :model-value="scraperProgressValue"
            color="primary"
            height="4"
            rounded
          />
        </div>

        <v-expand-transition>
          <div
            v-if="filterOpen && !selectedItem"
            class="library-filter-panel"
            :class="{ 'library-filter-panel--expanded': filterExpanded }"
          >
            <div
              ref="filterStripRef"
              class="library-filter-groups"
              :class="{
                'library-filter-groups--expanded': filterExpanded,
                'library-filter-groups--dragging': filterDragging,
              }"
              @click.capture="handleFilterClickCapture"
              @pointercancel="handleFilterPointerEnd"
              @pointerdown="handleFilterPointerDown"
              @pointerleave="handleFilterPointerEnd"
              @pointermove="handleFilterPointerMove"
              @pointerup="handleFilterPointerEnd"
              @wheel="handleFilterWheel"
            >
              <div
                v-for="group in filterGroups"
                :key="group.id"
                class="library-filter-group"
              >
                <div class="library-filter-chips">
                  <v-chip
                    v-for="child in group.children"
                    :key="`${group.id}-${child}`"
                    class="library-filter-chip"
                    :color="
                      isChildActive(group.id, child) ? 'primary' : undefined
                    "
                    :variant="isChildActive(group.id, child) ? 'flat' : 'tonal'"
                    size="small"
                    @click="selectChild(group.id, child)"
                  >
                    {{ child }}
                  </v-chip>
                </div>
              </div>
            </div>

            <v-tooltip
              :text="filterExpanded ? '收起分类' : '展开多行'"
              location="bottom"
            >
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  class="library-filter-expand-button"
                  :icon="filterExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"
                  variant="text"
                  size="small"
                  @click="toggleFilterExpanded"
                />
              </template>
            </v-tooltip>
          </div>
        </v-expand-transition>

        <div
          class="library-scroll"
          :class="{ 'library-scroll--detail': selectedItem }"
        >
          <template v-if="selectedItem">
            <div class="library-detail-page">
              <section
                class="library-detail-hero"
                :class="`library-detail-hero--${selectedItem.posterTone}`"
              >
                <div
                  class="library-detail-hero-poster"
                  :style="detailHeroStyle"
                />
                <div class="library-detail-hero-art" />
                <div class="library-detail-hero-content">
                  <div class="library-detail-kicker">
                    {{ selectedSectionLabel }} · {{ selectedMedia.child }}
                  </div>
                  <h2>{{ selectedMedia.title }}</h2>
                  <div class="library-detail-original">
                    {{ selectedMedia.originalTitle }}
                  </div>

                  <div class="library-detail-actions">
                    <v-btn
                      color="primary"
                      prepend-icon="mdi-play"
                      rounded="pill"
                      size="small"
                      @click="playSelected"
                    >
                      播放
                    </v-btn>
                  </div>

                  <div class="library-detail-meta-line">
                    <span
                      class="library-detail-score"
                      v-if="selectedMedia.rating"
                      >★ {{ Number(selectedMedia.rating).toFixed(1) }}</span
                    >
                    <span v-if="selectedMedia.year">{{
                      selectedMedia.year
                    }}</span>
                    <span>{{ selectedMedia.duration }}</span>
                    <span v-if="selectedMedia.quality">{{
                      selectedMedia.quality
                    }}</span>
                    <span v-if="selectedMedia.source">{{
                      selectedMedia.source
                    }}</span>
                  </div>

                  <p>{{ selectedMedia.summary || "暂未获取到影片简介。" }}</p>
                  <div class="library-detail-people">
                    <span>导演：{{ selectedMedia.director || "待补充" }}</span>
                    <span
                      >主演：{{
                        selectedMedia.cast?.join(" / ") || "待补充"
                      }}</span
                    >
                  </div>
                </div>
              </section>

              <section class="library-detail-section">
                <div class="library-detail-section-title">影片信息</div>
                <div class="library-detail-tags">
                  <v-chip
                    v-for="tag in selectedDetailTags"
                    :key="tag"
                    size="small"
                    variant="tonal"
                  >
                    {{ tag }}
                  </v-chip>
                </div>
              </section>

              <section
                v-if="selectedDetail?.files?.length"
                class="library-detail-section"
              >
                <div class="library-detail-section-title">选集</div>
                <v-tabs
                  v-if="selectedSeasonItems.length"
                  v-model="activeSeason"
                  class="library-season-tabs"
                  density="compact"
                  show-arrows
                >
                  <v-tab
                    v-for="season in selectedSeasonItems"
                    :key="season.value"
                    :value="season.value"
                  >
                    {{ season.title }}
                  </v-tab>
                </v-tabs>
                <div class="library-episode-strip">
                  <button
                    v-for="episode in selectedEpisodes"
                    :key="episode.fileId"
                    class="library-episode-card"
                    type="button"
                    @click="playEpisode(episode)"
                  >
                    <span
                      class="library-episode-thumb"
                      :class="`library-episode-thumb--${selectedMedia.posterTone}`"
                      :style="buildLibraryImageStyle(episode.thumbnailUrl)"
                    />
                    <span class="library-episode-title">{{
                      episode.cardTitle
                    }}</span>
                    <small
                      v-if="episode.cardSubtitle"
                      class="library-episode-meta"
                      >{{ episode.cardSubtitle }}</small
                    >
                    <small
                      v-if="episode.resumeText"
                      class="library-episode-resume"
                      >{{ episode.resumeText }}</small
                    >
                  </button>
                </div>
              </section>

              <section
                v-if="selectedCastItems.length"
                class="library-detail-section"
              >
                <div class="library-detail-section-title">相关演员</div>
                <div class="library-cast-strip">
                  <div
                    v-for="person in selectedCastItems"
                    :key="person.id"
                    class="library-cast-card"
                  >
                    <div
                      class="library-cast-avatar"
                      :class="`library-cast-avatar--${person.tone}`"
                    >
                      <img
                        v-if="person.avatarUrl"
                        :src="person.avatarUrl"
                        :alt="person.name"
                      />
                      <span v-else>{{ person.name.slice(0, 1) }}</span>
                    </div>
                    <div class="library-cast-name">{{ person.name }}</div>
                    <div class="library-cast-role">{{ person.role }}</div>
                  </div>
                </div>
              </section>

              <section
                v-if="selectedResourcePaths.length"
                class="library-detail-section"
              >
                <div class="library-detail-section-title">资源目录</div>
                <div class="library-resource-paths">
                  <div
                    v-for="path in selectedResourcePaths"
                    :key="path"
                    class="library-resource-path"
                  >
                    {{ path }}
                  </div>
                </div>
              </section>
            </div>
          </template>

          <div v-else-if="loading" class="library-state">
            <v-progress-circular indeterminate color="primary" size="32" />
            <div class="text-body-2 text-medium-emphasis mt-3">
              正在加载影视库
            </div>
          </div>

          <div v-else-if="!filteredItems.length" class="library-state">
            <v-icon
              icon="mdi-view-grid-plus-outline"
              size="34"
              color="medium-emphasis"
            />
            <div class="text-body-2 text-medium-emphasis mt-3">
              当前没有可展示的影视条目，先运行一次刮削吧。
            </div>
          </div>

          <LibraryGrid
            v-else
            :items="filteredItems"
            :show-progress="historyOnly"
            @select="openDetail"
          />
        </div>
      </main>
    </div>
  </section>
</template>

<style scoped>
.library-page {
  padding: 0;
}

.library-shell {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  display: flex;
  overflow: hidden;
  background: rgb(var(--v-theme-surface));
}

.library-content {
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
}

.library-nav-row {
  flex: 0 0 auto;
  min-height: 44px;
  padding: 6px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}

.library-scraper-status {
  flex: 0 0 auto;
  padding: 8px 12px 10px;
  display: grid;
  gap: 7px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
  background: linear-gradient(
    180deg,
    rgba(var(--v-theme-primary), 0.02),
    transparent
  );
}

.library-scraper-status--running {
  background: linear-gradient(
    180deg,
    rgba(var(--v-theme-primary), 0.06),
    transparent
  );
}

.library-scraper-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.library-scraper-copy {
  min-width: 0;
  flex: 1 1 auto;
}

.library-scraper-actions {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
}

.library-scraper-title {
  font-size: 12px;
  line-height: 1.2;
  font-weight: 650;
}

.library-scraper-meta {
  margin-top: 2px;
  font-size: 11px;
  line-height: 1.2;
  color: rgba(var(--v-theme-on-surface), 0.58);
}

.library-scraper-progress {
  width: min(420px, 100%);
}

.library-primary-nav {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
}

.library-primary-nav::-webkit-scrollbar {
  display: none;
}

.library-nav-button {
  flex: 0 0 auto;
  min-width: 0;
  height: 30px !important;
  padding-inline: 10px !important;
  letter-spacing: 0;
  font-weight: 650;
}

.library-nav-button:not(.library-nav-button--active) {
  color: rgba(var(--v-theme-on-surface), 0.74);
}

.library-nav-actions {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.library-action-button {
  width: 30px !important;
  height: 30px !important;
}

.library-filter-panel {
  flex: 0 0 auto;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  background:
    linear-gradient(180deg, rgba(var(--v-theme-primary), 0.035), transparent),
    rgba(var(--v-theme-surface), 1);
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.library-filter-panel--expanded {
  align-items: flex-start;
}

.library-filter-groups {
  min-width: 0;
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  cursor: grab;
  user-select: none;
  touch-action: pan-y;
}

.library-filter-groups::-webkit-scrollbar {
  display: none;
}

.library-filter-groups--expanded {
  flex-wrap: wrap;
  overflow: visible;
  cursor: default;
  user-select: auto;
}

.library-filter-groups--expanded .library-filter-group {
  min-width: 0;
  flex: 0 1 auto;
}

.library-filter-groups--expanded .library-filter-chips {
  min-width: 0;
  flex-wrap: wrap;
}

.library-filter-groups--dragging {
  cursor: grabbing;
}

.library-filter-group {
  flex: 0 0 auto;
  min-width: max-content;
  display: flex;
  align-items: center;
}

.library-filter-chips {
  min-width: max-content;
  display: flex;
  align-items: center;
  gap: 6px;
}

.library-filter-chip {
  flex: 0 0 auto;
  cursor: pointer;
}

.library-filter-expand-button {
  flex: 0 0 auto;
  width: 28px !important;
  height: 28px !important;
  margin-top: -1px;
}

.library-scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 12px 14px 18px;
}

.library-scroll--detail {
  padding: 0;
}

.library-state {
  min-height: 100%;
  display: grid;
  place-content: center;
  justify-items: center;
}

.library-detail-page {
  min-height: 100%;
  background: rgb(var(--v-theme-surface));
}

.library-detail-hero {
  position: relative;
  min-height: min(56vh, 440px);
  display: flex;
  align-items: flex-end;
  padding: 0 36px 34px;
  overflow: hidden;
  color: rgba(255, 255, 255, 0.96);
  background:
    radial-gradient(
      circle at 78% 18%,
      rgba(255, 255, 255, 0.2),
      transparent 22%
    ),
    linear-gradient(135deg, #334155, #111827);
}

.library-detail-hero::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(0, 0, 0, 0.5), transparent 62%),
    linear-gradient(
      to top,
      rgba(var(--v-theme-surface), 1) 0,
      rgba(var(--v-theme-surface), 0.86) 12%,
      rgba(var(--v-theme-surface), 0) 44%
    );
  z-index: 1;
}

.library-detail-hero-poster {
  position: absolute;
  inset: 0;
  background-size: cover, cover, cover;
  background-position:
    center,
    center,
    center top 24%;
  filter: saturate(1.08) contrast(1.03);
  transform: scale(1.02);
  opacity: 0.98;
}

.library-detail-hero-art {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(
      circle at 78% 22%,
      rgba(255, 255, 255, 0.16),
      transparent 14%
    ),
    radial-gradient(
      circle at 58% 18%,
      rgba(255, 255, 255, 0.12),
      transparent 18%
    ),
    linear-gradient(
      115deg,
      transparent 0 52%,
      rgba(255, 255, 255, 0.1) 53% 54%,
      transparent 55% 100%
    );
  opacity: 0.92;
}

.library-detail-hero--amber {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #92400e, #111827);
}
.library-detail-hero--steel {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.18),
      transparent 28%
    ),
    linear-gradient(135deg, #475569, #020617);
}
.library-detail-hero--red {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.18),
      transparent 28%
    ),
    linear-gradient(135deg, #991b1b, #111827);
}
.library-detail-hero--rose {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #9f1239, #111827);
}
.library-detail-hero--green {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.18),
      transparent 28%
    ),
    linear-gradient(135deg, #166534, #111827);
}
.library-detail-hero--violet {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #5b21b6, #111827);
}
.library-detail-hero--mint {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #0f766e, #111827);
}
.library-detail-hero--sky {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #0369a1, #111827);
}
.library-detail-hero--ink {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.16),
      transparent 28%
    ),
    linear-gradient(135deg, #334155, #020617);
}
.library-detail-hero--ocean {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #0f766e, #082f49);
}
.library-detail-hero--cyan {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.2),
      transparent 28%
    ),
    linear-gradient(135deg, #0e7490, #111827);
}
.library-detail-hero--paper {
  background:
    radial-gradient(
      circle at 82% 14%,
      rgba(255, 255, 255, 0.24),
      transparent 28%
    ),
    linear-gradient(135deg, #78716c, #1c1917);
}

.library-detail-hero-content {
  position: relative;
  z-index: 2;
  width: min(720px, 72%);
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.library-detail-kicker {
  font-size: 12px;
  line-height: 1.2;
  opacity: 0.78;
}

.library-detail-hero h2 {
  margin: 7px 0 0;
  font-size: 30px;
  line-height: 1.1;
  font-weight: 850;
  letter-spacing: -0.02em;
}

.library-detail-original {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.2;
  opacity: 0.72;
}

.library-detail-actions {
  margin-top: 18px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.library-detail-meta-line {
  margin-top: 13px;
  display: flex;
  align-items: center;
  gap: 9px;
  flex-wrap: wrap;
  font-size: 12px;
  line-height: 1.2;
  color: rgba(255, 255, 255, 0.74);
}

.library-detail-meta-line span + span::before {
  content: "";
  display: inline-block;
  width: 3px;
  height: 3px;
  margin-right: 9px;
  border-radius: 50%;
  vertical-align: middle;
  background: rgba(255, 255, 255, 0.42);
}

.library-detail-score {
  color: #facc15;
  font-weight: 700;
}

.library-detail-meta-line .library-detail-score::before {
  display: none;
}

.library-detail-hero p {
  margin: 12px 0 0;
  max-width: 560px;
  font-size: 13px;
  line-height: 1.7;
  color: rgba(255, 255, 255, 0.82);
}

.library-detail-people {
  margin-top: 8px;
  display: grid;
  gap: 4px;
  font-size: 12px;
  line-height: 1.35;
  color: rgba(255, 255, 255, 0.68);
}

.library-detail-section {
  padding: 22px 36px 0;
  display: grid;
  gap: 12px;
}

.library-detail-section-title {
  font-size: 14px;
  line-height: 1.2;
  font-weight: 750;
}

.library-detail-tags,
.library-cast-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.library-season-tabs {
  min-width: 0;
  max-width: 100%;
  min-height: 32px;
  height: 32px;
}

.library-season-tabs :deep(.v-slide-group__content) {
  gap: 6px;
}

.library-season-tabs :deep(.v-tab) {
  min-width: 0;
  height: 30px;
  min-height: 30px;
  padding-inline: 12px;
  border-radius: 999px;
  font-size: 12px;
  letter-spacing: 0;
  text-transform: none;
}

.library-season-tabs :deep(.v-tab--selected) {
  background: rgba(var(--v-theme-primary), 0.1);
}

.library-episode-strip {
  display: flex;
  gap: 14px;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: none;
}

.library-episode-strip::-webkit-scrollbar {
  display: none;
}

.library-episode-card {
  flex: 0 0 170px;
  display: grid;
  gap: 7px;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
  transition:
    transform 0.18s ease,
    opacity 0.18s ease;
}

.library-episode-card:hover {
  transform: translateY(-3px);
}

.library-episode-card:hover .library-episode-thumb {
  box-shadow: 0 14px 24px rgba(15, 23, 42, 0.18);
}

.library-episode-thumb {
  display: block;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  background-size: cover;
  background-position: center;
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.28),
      transparent 22%
    ),
    linear-gradient(145deg, #334155, #111827);
  box-shadow: 0 10px 18px rgba(15, 23, 42, 0.12);
  transition: box-shadow 0.18s ease;
}

.library-episode-thumb--amber {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.3),
      transparent 22%
    ),
    linear-gradient(145deg, #f59e0b, #78350f);
}
.library-episode-thumb--steel {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.24),
      transparent 22%
    ),
    linear-gradient(145deg, #64748b, #020617);
}
.library-episode-thumb--red {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.26),
      transparent 22%
    ),
    linear-gradient(145deg, #ef4444, #450a0a);
}
.library-episode-thumb--rose {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.26),
      transparent 22%
    ),
    linear-gradient(145deg, #fb7185, #4c0519);
}
.library-episode-thumb--green {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.24),
      transparent 22%
    ),
    linear-gradient(145deg, #16a34a, #052e16);
}
.library-episode-thumb--violet {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.26),
      transparent 22%
    ),
    linear-gradient(145deg, #8b5cf6, #2e1065);
}
.library-episode-thumb--mint {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.3),
      transparent 22%
    ),
    linear-gradient(145deg, #5eead4, #134e4a);
}
.library-episode-thumb--sky {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.3),
      transparent 22%
    ),
    linear-gradient(145deg, #38bdf8, #082f49);
}
.library-episode-thumb--ink {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.2),
      transparent 22%
    ),
    linear-gradient(145deg, #475569, #0f172a);
}
.library-episode-thumb--ocean {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.28),
      transparent 22%
    ),
    linear-gradient(145deg, #0ea5e9, #064e3b);
}
.library-episode-thumb--cyan {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.28),
      transparent 22%
    ),
    linear-gradient(145deg, #22d3ee, #164e63);
}
.library-episode-thumb--paper {
  background:
    radial-gradient(
      circle at 20% 18%,
      rgba(255, 255, 255, 0.32),
      transparent 22%
    ),
    linear-gradient(145deg, #d6d3d1, #57534e);
}

.library-episode-title {
  font-size: 12px;
  line-height: 1.3;
  font-weight: 650;
}

.library-episode-meta {
  color: rgba(var(--v-theme-on-surface), 0.56);
  font-size: 11px;
  line-height: 1.2;
}

.library-episode-resume {
  color: rgba(var(--v-theme-on-surface), 0.58);
  font-size: 11px;
  line-height: 1.2;
}

.library-cast-card {
  flex: 0 0 96px;
  display: grid;
  justify-items: center;
  gap: 7px;
  text-align: center;
}

.library-cast-avatar {
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  color: #fff;
  font-size: 22px;
  font-weight: 800;
  background: linear-gradient(145deg, #64748b, #111827);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.14);
}

.library-cast-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  filter: saturate(1.04) brightness(1.04);
}

.library-cast-avatar--amber {
  background: linear-gradient(145deg, #f59e0b, #78350f);
}
.library-cast-avatar--steel {
  background: linear-gradient(145deg, #64748b, #020617);
}
.library-cast-avatar--rose {
  background: linear-gradient(145deg, #fb7185, #4c0519);
}
.library-cast-avatar--green {
  background: linear-gradient(145deg, #16a34a, #052e16);
}
.library-cast-avatar--violet {
  background: linear-gradient(145deg, #8b5cf6, #2e1065);
}
.library-cast-avatar--cyan {
  background: linear-gradient(145deg, #22d3ee, #164e63);
}

.library-cast-name {
  max-width: 96px;
  font-size: 12px;
  line-height: 1.25;
  font-weight: 650;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.library-cast-role {
  font-size: 11px;
  line-height: 1.2;
  color: rgba(var(--v-theme-on-surface), 0.52);
  min-height: 24px;
}

.library-resource-paths {
  display: grid;
  gap: 8px;
}

.library-resource-path {
  font-size: 12px;
  line-height: 1.55;
  color: rgba(var(--v-theme-on-surface), 0.68);
  word-break: break-all;
}

@media (max-width: 760px) {
  .library-nav-row {
    gap: 8px;
    padding-inline: 10px;
  }

  .library-scroll {
    padding-inline: 10px;
  }

  .library-action-button {
    width: 28px !important;
    height: 28px !important;
  }

  .library-detail-hero {
    min-height: 380px;
    padding: 0 18px 28px;
  }

  .library-detail-hero-content {
    width: 100%;
  }

  .library-detail-hero h2 {
    font-size: 24px;
  }

  .library-detail-section {
    padding-inline: 18px;
  }

  .library-episode-card {
    flex-basis: 146px;
  }
}
</style>
