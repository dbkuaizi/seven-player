<script setup>
import { ref } from 'vue'
import { formatDurationMs } from '../../utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  player: { type: Object, required: true },
  loading: { type: Boolean, default: false },
  textTracks: { type: Array, default: () => [] },
  selectedItem: { type: Object, default: null },
})

const emit = defineEmits([
  'update:modelValue',
  'close',
  'choose-subtitle',
  'external-play',
  'can-play',
  'time-update',
  'ended',
  'error',
])

const playerElement = ref(null)

function close() {
  emit('update:modelValue', false)
  emit('close', playerElement.value)
}

function handleCanPlay() {
  emit('can-play', playerElement.value)
}
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="980"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="builtin-player-card">
      <div class="builtin-player-header">
        <div class="min-w-0">
          <div class="text-subtitle-1 font-weight-medium text-truncate">
            {{ props.player.title || 'Seven Player' }}
          </div>
          <div class="builtin-player-meta">
            <v-chip
              v-if="props.player.resumeUsed && props.player.startMs > 0"
              size="x-small"
              color="primary"
              variant="tonal"
            >
              {{ formatDurationMs(props.player.startMs) }}
            </v-chip>
            <v-chip
              v-if="props.player.subtitleName"
              size="x-small"
              :color="props.player.subtitleUsable ? 'teal' : 'warning'"
              variant="tonal"
            >
              {{ props.player.subtitleName }}
            </v-chip>
          </div>
        </div>
        <v-btn icon="mdi-close" variant="text" size="small" @click="close" />
      </div>

      <v-divider />

      <v-card-text class="builtin-player-body">
        <div v-if="props.loading" class="builtin-player-loading">
          <v-progress-circular indeterminate color="primary" />
        </div>
        <media-player
          v-else-if="props.player.url"
          ref="playerElement"
          class="builtin-media-player"
          :src="props.player.url"
          :title="props.player.title"
          :text-tracks.prop="props.textTracks"
          view-type="video"
          stream-type="on-demand"
          preload="metadata"
          crossorigin
          playsinline
          @can-play="handleCanPlay"
          @time-update="emit('time-update', $event)"
          @ended="emit('ended')"
          @error="emit('error')"
        >
          <media-outlet />
          <media-community-skin />
        </media-player>
      </v-card-text>

      <v-card-actions>
        <v-btn
          variant="text"
          prepend-icon="mdi-subtitles-outline"
          @click="emit('choose-subtitle', props.selectedItem)"
        >
          选择字幕
        </v-btn>
        <v-spacer />
        <v-btn
          variant="text"
          prepend-icon="mdi-open-in-new"
          @click="emit('external-play')"
        >
          外部播放器
        </v-btn>
        <v-btn color="primary" variant="flat" @click="close">
          关闭
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
