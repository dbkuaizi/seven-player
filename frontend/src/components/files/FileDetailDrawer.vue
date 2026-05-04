<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  selectedItem: { type: Object, default: null },
  selectedResumeText: { type: String, default: '' },
  selectedSubtitleName: { type: String, default: '' },
  selectedLastPlayedText: { type: String, default: '' },
})

const emit = defineEmits([
  'update:modelValue',
  'primary-action',
  'play',
  'builtin-play',
  'play-from-start',
  'jump',
  'choose-subtitle',
  'clear-subtitle',
  'clear-progress',
])
</script>

<template>
  <v-navigation-drawer
    :model-value="props.modelValue"
    class="detail-drawer"
    location="right"
    temporary
    :scrim="false"
    width="320"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template v-if="props.selectedItem">
      <div class="detail-header">
        <div class="d-flex align-center ga-3 min-w-0">
          <v-avatar
            size="36"
            variant="tonal"
            :color="props.selectedItem.iconColor"
          >
            <v-icon :color="props.selectedItem.iconColor">{{ props.selectedItem.icon }}</v-icon>
          </v-avatar>

          <div class="min-w-0 flex-1-1">
            <div class="text-subtitle-2 font-weight-medium text-truncate">
              {{ props.selectedItem.displayName || props.selectedItem.name }}
            </div>
            <div class="text-caption text-medium-emphasis">
              {{ props.selectedItem.kindLabel }}
            </div>
          </div>
        </div>

        <v-btn icon="mdi-close" variant="text" size="small" @click="emit('update:modelValue', false)" />
      </div>

      <v-divider />

      <div class="detail-body">
        <div class="d-flex flex-column ga-2">
          <v-btn
            v-if="props.selectedItem.isDirectory"
            color="primary"
            block
            prepend-icon="mdi-folder-open-outline"
            @click="emit('primary-action', props.selectedItem)"
          >
            打开目录
          </v-btn>

          <template v-else-if="props.selectedItem.isVideo">
            <v-btn
              color="primary"
              block
              prepend-icon="mdi-play-circle-outline"
              @click="emit('play', props.selectedItem)"
            >
              立即播放
            </v-btn>
            <v-btn
              variant="tonal"
              block
              prepend-icon="mdi-monitor-play"
              @click="emit('builtin-play', props.selectedItem)"
            >
              内置播放
            </v-btn>
            <v-btn
              variant="tonal"
              block
              prepend-icon="mdi-replay"
              @click="emit('play-from-start', props.selectedItem)"
            >
              从头播放
            </v-btn>
            <v-btn
              variant="text"
              block
              prepend-icon="mdi-timeline-clock-outline"
              @click="emit('jump', props.selectedItem)"
            >
              跳转播放
            </v-btn>
          </template>
        </div>

        <v-list class="mt-4" density="compact" lines="two">
          <v-list-item title="类型" :subtitle="props.selectedItem.kindLabel" />
          <v-list-item title="大小" :subtitle="props.selectedItem.sizeText" />
          <v-list-item title="更新时间" :subtitle="props.selectedItem.updatedText" />
          <v-list-item v-if="props.selectedItem.pickCode" title="PickCode" :subtitle="props.selectedItem.pickCode" />

          <template v-if="props.selectedItem.isVideo">
            <v-list-item title="时长" :subtitle="props.selectedItem.durationText || '--'" />
            <v-list-item title="续播位置" :subtitle="props.selectedResumeText" />
            <v-list-item title="外挂字幕" :subtitle="props.selectedSubtitleName" />
            <v-list-item title="上次播放" :subtitle="props.selectedLastPlayedText" />
          </template>
        </v-list>

        <template v-if="props.selectedItem.isVideo">
          <v-divider class="my-2" />
          <div class="d-flex flex-column ga-2">
            <v-btn
              variant="text"
              prepend-icon="mdi-subtitles-outline"
              @click="emit('choose-subtitle', props.selectedItem)"
            >
              绑定外挂字幕
            </v-btn>
            <v-btn
              variant="text"
              prepend-icon="mdi-subtitles-off-outline"
              :disabled="!props.selectedItem.subtitlePath"
              @click="emit('clear-subtitle', props.selectedItem)"
            >
              清除字幕绑定
            </v-btn>
            <v-btn
              variant="text"
              prepend-icon="mdi-history"
              :disabled="!props.selectedItem.resumeMs"
              @click="emit('clear-progress', props.selectedItem)"
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
</template>
