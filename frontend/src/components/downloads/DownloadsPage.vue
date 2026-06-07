<script setup>
import PageSizeMenu from '../app/PageSizeMenu.vue'

const props = defineProps({
  loggedIn: { type: Boolean, default: false },
  downloadLoading: { type: Boolean, default: false },
  downloadSubmitting: { type: Boolean, default: false },
  downloadFilter: { type: String, default: 'active' },
  downloadFilterOptions: { type: Array, default: () => [] },
  downloadQuotaText: { type: String, default: '0 / 0' },
  downloadQuotaProgress: { type: Number, default: 0 },
  fileListDensityClass: { type: String, default: '' },
  fileListAvatarSize: { type: Number, default: 28 },
  fileListIconSize: { type: Number, default: 16 },
  tasks: { type: Array, default: () => [] },
  downloadEmptyText: { type: String, default: '' },
  showDownloadPagination: { type: Boolean, default: false },
  downloadPaginationSummaryText: { type: String, default: '' },
  pageSize: { type: Number, default: 20 },
  pageSizeOptions: { type: Array, default: () => [] },
  downloadPageCount: { type: Number, default: 1 },
  downloadPage: { type: Number, default: 1 },
})

const emit = defineEmits([
  'update:downloadFilter',
  'open-download-dialog',
  'refresh',
  'open-directory',
  'copy-url',
  'delete-task',
  'page-size-change',
  'page-change',
])

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
</script>

<template>
  <section class="page-section downloads-page">
    <v-card class="section-card d-flex flex-column">
      <div class="page-toolbar download-toolbar download-nav-row">
        <div class="download-pill-tabs" role="tablist" aria-label="云下载筛选">
          <v-btn
            v-for="entry in props.downloadFilterOptions"
            :key="entry.value"
            class="pill-tab"
            :class="{ 'pill-tab--active': props.downloadFilter === entry.value }"
            :color="props.downloadFilter === entry.value ? 'primary' : undefined"
            :variant="props.downloadFilter === entry.value ? 'flat' : 'text'"
            rounded="pill"
            size="small"
            role="tab"
            :aria-selected="props.downloadFilter === entry.value"
            @click="emit('update:downloadFilter', entry.value)"
          >
            {{ entry.label }}
          </v-btn>
        </div>

        <div class="download-toolbar-spacer" />

        <div class="download-quota mr-2">
          <div class="download-quota-text text-caption text-medium-emphasis">
            <span class="download-quota-label">额度</span>
            <span class="download-quota-value">{{ props.downloadQuotaText }}</span>
          </div>
          <v-progress-linear
            class="download-quota-bar"
            :model-value="props.downloadQuotaProgress"
            color="primary"
            bg-color="rgba(var(--v-theme-on-surface), 0.08)"
            height="6"
            rounded
          />
        </div>

        <v-tooltip text="添加云下载" location="bottom">
          <template #activator="{ props: tooltipProps }">
            <v-btn
              v-bind="tooltipProps"
              class="download-add-btn"
              color="primary"
              icon="mdi-link-plus"
              size="small"
              variant="text"
              :disabled="!props.loggedIn"
              @click="emit('open-download-dialog')"
            />
          </template>
        </v-tooltip>

        <v-btn
          icon="mdi-refresh"
          variant="text"
          size="small"
          class="download-refresh-btn"
          :disabled="!props.loggedIn"
          :loading="props.downloadLoading"
          @click="emit('refresh')"
        />
      </div>

      <v-progress-linear
        :active="props.downloadLoading || props.downloadSubmitting"
        :indeterminate="props.downloadLoading || props.downloadSubmitting"
        height="2"
      />

      <template v-if="!props.loggedIn">
        <div class="state-shell">
          <v-icon size="52" color="primary">mdi-download-circle-outline</v-icon>
          <div class="text-subtitle-1">请先登录 115 账号</div>
          <div class="text-body-2 text-medium-emphasis">
            登录后即可查看离线任务、选择保存目录并直接打开对应网盘目录。
          </div>
        </div>
      </template>

      <template v-else>
        <div class="table-scroll files-scroll" :class="props.fileListDensityClass">
          <table class="files-table downloads-table">
            <thead>
              <tr>
                <th class="name-column">文件名</th>
                <th class="download-progress-column">进度</th>
                <th class="download-action-column text-right text-no-wrap">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!props.tasks.length">
                <td colspan="3" class="empty-row">
                  {{ props.downloadEmptyText }}
                </td>
              </tr>

              <tr v-for="task in props.tasks" :key="task.infoHash" class="file-row">
                <td class="name-column">
                  <div class="name-cell">
                    <v-avatar
                      :size="props.fileListAvatarSize"
                      variant="tonal"
                      :color="offlineTaskColor(task)"
                    >
                      <v-icon :size="props.fileListIconSize" :color="offlineTaskColor(task)">
                        {{ offlineTaskIcon(task) }}
                      </v-icon>
                    </v-avatar>

                    <div class="name-text">
                      <div class="file-title">
                        <span class="file-title-main text-truncate">{{ task.name }}</span>
                      </div>
                      <div class="file-subtitle text-truncate">
                        {{ task.metaText }}
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
                      @click.stop="emit('open-directory', task)"
                    />
                    <v-btn
                      icon="mdi-content-copy"
                      size="small"
                      variant="text"
                      title="复制任务链接"
                      @click.stop="emit('copy-url', task)"
                    />
                    <v-btn
                      icon="mdi-delete-outline"
                      size="small"
                      variant="text"
                      color="error"
                      title="删除任务"
                      @click.stop="emit('delete-task', task)"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="props.showDownloadPagination" class="pagination-bar">
          <div class="pagination-summary text-caption text-medium-emphasis">
            {{ props.downloadPaginationSummaryText }}
          </div>

          <div class="pagination-controls">
            <PageSizeMenu
              :model-value="props.pageSize"
              :items="props.pageSizeOptions"
              @update:model-value="emit('page-size-change', $event)"
            />

            <v-pagination
              :length="props.downloadPageCount"
              :model-value="props.downloadPage"
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
