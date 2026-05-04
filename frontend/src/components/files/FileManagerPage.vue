<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  loggedIn: { type: Boolean, default: false },
  directoryLoading: { type: Boolean, default: false },
  actionLoading: { type: Boolean, default: false },
  searchLoading: { type: Boolean, default: false },
  isGlobalSearchActive: { type: Boolean, default: false },
  searchSummaryText: { type: String, default: '' },
  breadcrumbDisplayItems: { type: Array, default: () => [] },
  searchInputVisible: { type: Boolean, default: false },
  searchQuery: { type: String, default: '' },
  typeFilter: { type: String, default: 'all' },
  sortMode: { type: String, default: 'folders' },
  filterMenuOpen: { type: Boolean, default: false },
  smallFileFilterMB: { type: Number, default: 0 },
  settings: { type: Object, required: true },
  typeOptions: { type: Array, default: () => [] },
  sortOptions: { type: Array, default: () => [] },
  smallFileFilterOptions: { type: Array, default: () => [] },
  fileListDensityOptions: { type: Array, default: () => [] },
  items: { type: Array, default: () => [] },
  fileEmptyText: { type: String, default: '' },
  selectedItem: { type: Object, default: null },
  fileListDensityClass: { type: String, default: '' },
  fileListAvatarSize: { type: Number, default: 28 },
  fileListIconSize: { type: Number, default: 16 },
  showPagination: { type: Boolean, default: false },
  paginationSummaryText: { type: String, default: '' },
  pageSize: { type: Number, default: 20 },
  pageCount: { type: Number, default: 1 },
  activePage: { type: Number, default: 1 },
  pageSizeOptions: { type: Array, default: () => [] },
})

const emit = defineEmits([
  'update:searchQuery',
  'update:typeFilter',
  'update:sortMode',
  'update:filterMenuOpen',
  'open-login',
  'open-breadcrumb',
  'reload',
  'toggle-search',
  'search-blur',
  'search-clear',
  'trigger-search',
  'save-small-file-filter',
  'save-file-list-density',
  'save-show-title-badges',
  'open-details',
  'primary-action',
  'page-size-change',
  'page-change',
])

const searchField = ref(null)
const badgeVisibleCounts = ref({})
const badgeRowElements = new Map()
let badgeMeasureCanvas = null
let badgeRefreshFrame = 0

const loading = computed(() => props.directoryLoading || props.actionLoading || props.searchLoading)

watch(() => props.searchInputVisible, (visible) => {
  if (visible) {
    nextTick(() => {
      searchField.value?.focus?.()
    })
  }
})

watch(
  () => [props.items, props.fileListDensityClass],
  () => {
    nextTick(() => {
      queueBadgeVisibilityRefresh()
    })
  },
  { deep: true },
)

onMounted(() => {
  window.addEventListener('resize', queueBadgeVisibilityRefresh)
  nextTick(() => {
    queueBadgeVisibilityRefresh()
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', queueBadgeVisibilityRefresh)
  if (badgeRefreshFrame) {
    window.cancelAnimationFrame(badgeRefreshFrame)
    badgeRefreshFrame = 0
  }
})

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

  for (const item of props.items) {
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
</script>

<template>
  <section class="page-section">
    <v-card class="section-card d-flex flex-column">
      <v-toolbar density="compact" flat class="page-toolbar px-2">
        <div class="breadcrumb-strip">
          <div class="path-bar">
            <div v-if="props.isGlobalSearchActive" class="search-path-indicator">
              <v-icon size="16" color="medium-emphasis">mdi-magnify</v-icon>
              <span class="text-truncate">{{ props.searchSummaryText }}</span>
            </div>

            <v-breadcrumbs
              v-else
              class="file-breadcrumbs pa-0"
              :items="props.breadcrumbDisplayItems"
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
                  @click="emit('open-breadcrumb', item.id)"
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
              :disabled="!props.loggedIn"
              :loading="props.isGlobalSearchActive ? props.searchLoading : props.directoryLoading"
              @click="emit('reload')"
            />
          </div>
        </div>

        <div class="search-slot">
          <div
            class="search-shell"
            :class="{ 'search-shell--expanded': props.searchInputVisible }"
          >
            <v-btn
              :icon="props.searchInputVisible ? 'mdi-close' : 'mdi-magnify'"
              size="small"
              variant="text"
              :disabled="!props.loggedIn"
              class="search-trigger"
              @click="emit('toggle-search')"
            />

            <div class="search-field-wrap">
              <v-text-field
                ref="searchField"
                :model-value="props.searchQuery"
                class="compact-search"
                clearable
                :disabled="!props.loggedIn"
                density="compact"
                hide-details
                variant="solo-filled"
                flat
                placeholder="全盘搜索"
                prepend-inner-icon="mdi-magnify"
                @update:model-value="emit('update:searchQuery', $event)"
                @blur="emit('search-blur')"
                @click:clear="emit('search-clear')"
                @keydown.enter.prevent="emit('trigger-search')"
              />
            </div>
          </div>
        </div>

        <v-menu
          :model-value="props.filterMenuOpen"
          location="bottom end"
          :close-on-content-click="false"
          @update:model-value="emit('update:filterMenuOpen', $event)"
        >
          <template #activator="{ props: menuProps }">
            <v-btn
              class="filter-trigger"
              :class="{ 'filter-trigger--active': props.filterMenuOpen }"
              :aria-controls="menuProps['aria-controls']"
              :aria-expanded="menuProps['aria-expanded']"
              :aria-haspopup="menuProps['aria-haspopup']"
              icon="mdi-tune-variant"
              size="small"
              variant="text"
              @click="menuProps.onClick"
            />
          </template>

          <v-card class="filter-menu" min-width="268">
            <v-card-text class="filter-menu-body">
              <v-select
                :model-value="props.typeFilter"
                class="filter-select"
                density="compact"
                hide-details
                item-title="label"
                item-value="value"
                :items="props.typeOptions"
                label="类型"
                variant="outlined"
                @update:model-value="emit('update:typeFilter', $event)"
              />

              <v-select
                :model-value="props.sortMode"
                class="filter-select"
                density="compact"
                hide-details
                item-title="label"
                item-value="value"
                :items="props.sortOptions"
                label="排序"
                variant="outlined"
                @update:model-value="emit('update:sortMode', $event)"
              />

              <v-select
                :model-value="props.smallFileFilterMB"
                class="filter-select"
                density="compact"
                hide-details
                item-title="label"
                item-value="value"
                :items="props.smallFileFilterOptions"
                label="小文件屏蔽规则"
                variant="outlined"
                @update:model-value="emit('save-small-file-filter', $event)"
              />

              <v-select
                :model-value="props.settings.fileListDensity"
                class="filter-select"
                density="compact"
                hide-details
                item-title="label"
                item-value="value"
                :items="props.fileListDensityOptions"
                label="文件列表密度"
                variant="outlined"
                @update:model-value="emit('save-file-list-density', $event)"
              />

              <v-divider class="filter-divider" />

              <v-checkbox
                :model-value="props.settings.showTitleBadges"
                class="filter-checkbox"
                color="primary"
                density="compact"
                hide-details
                label="显示徽章信息"
                @update:model-value="emit('save-show-title-badges', $event)"
              />
            </v-card-text>
          </v-card>
        </v-menu>
      </v-toolbar>

      <v-progress-linear
        :active="loading"
        :indeterminate="loading"
        height="2"
      />

      <template v-if="!props.loggedIn">
        <div class="state-shell">
          <v-icon size="52" color="primary">mdi-folder-search-outline</v-icon>
          <div class="text-subtitle-1">登录后即可浏览 115 文件</div>
          <div class="text-body-2 text-medium-emphasis">
            支持扫码登录和 Cookie 登录，登录状态会写入应用目录下的 SQLite 文件。
          </div>
          <div class="d-flex ga-2 flex-wrap">
            <v-btn color="primary" prepend-icon="mdi-qrcode" @click="emit('open-login', 'qr')">
              扫码登录
            </v-btn>
            <v-btn variant="tonal" prepend-icon="mdi-cookie-outline" @click="emit('open-login', 'cookie')">
              Cookie 登录
            </v-btn>
          </div>
        </div>
      </template>

      <template v-else>
        <div class="table-scroll files-scroll" :class="props.fileListDensityClass">
          <table class="files-table">
            <thead>
              <tr>
                <th class="name-column">名称</th>
                <th class="size-column text-no-wrap">大小</th>
                <th class="time-column text-no-wrap">更新时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!props.items.length">
                <td colspan="3" class="empty-row">
                  {{ props.fileEmptyText }}
                </td>
              </tr>

              <tr
                v-for="item in props.items"
                :key="item.rowKey"
                class="file-row"
                :class="{ 'selected-row': props.selectedItem?.rowKey === item.rowKey }"
                @click="emit('open-details', item)"
                @dblclick="emit('primary-action', item)"
              >
                <td class="name-column">
                  <div class="name-cell">
                    <v-avatar
                      :size="props.fileListAvatarSize"
                      class="name-avatar"
                      variant="tonal"
                      :color="item.iconColor"
                    >
                      <v-icon :size="props.fileListIconSize" :color="item.iconColor">{{ item.icon }}</v-icon>
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
                            <template #activator="{ props: tooltipProps }">
                              <v-chip
                                v-bind="tooltipProps"
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
                            <template #activator="{ props: tooltipProps }">
                              <v-chip
                                v-bind="tooltipProps"
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

        <div v-if="props.showPagination" class="pagination-bar">
          <div class="pagination-summary text-caption text-medium-emphasis">
            {{ props.paginationSummaryText }}
          </div>

          <div class="pagination-controls">
            <v-select
              :model-value="props.pageSize"
              class="page-size-select"
              density="compact"
              hide-details
              item-title="title"
              item-value="value"
              menu-icon="mdi-chevron-down"
              :items="props.pageSizeOptions"
              variant="plain"
              @update:model-value="emit('page-size-change', $event)"
            />

            <v-pagination
              :length="props.pageCount"
              :model-value="props.activePage"
              active-color="primary"
              density="comfortable"
              rounded="circle"
              size="small"
              total-visible="5"
              @update:model-value="emit('page-change', $event)"
            />
          </div>
        </div>
      </template>
    </v-card>
  </section>
</template>
