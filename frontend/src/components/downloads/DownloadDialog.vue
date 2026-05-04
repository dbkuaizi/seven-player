<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  downloadInput: { type: String, default: '' },
  downloadSubmitting: { type: Boolean, default: false },
  torrentSelecting: { type: Boolean, default: false },
  offlineTargetSelectOptions: { type: Array, default: () => [] },
  offlineTargetSelectValue: { type: String, default: '' },
  offlineTargetPathText: { type: String, default: '' },
})

const emit = defineEmits([
  'update:modelValue',
  'update:downloadInput',
  'close',
  'select-torrent',
  'submit',
  'select-target',
])

function close() {
  emit('update:modelValue', false)
  emit('close')
}
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="680"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card>
      <v-card-title class="download-dialog-title">
        <span>添加云下载</span>
        <v-btn
          icon="mdi-close"
          size="small"
          variant="text"
          @click="close"
        />
      </v-card-title>
      <v-divider />
      <v-card-text>
        <v-textarea
          :model-value="props.downloadInput"
          class="scroll-textarea scroll-textarea--download"
          hide-details
          persistent-placeholder
          rows="7"
          variant="outlined"
          placeholder="输入下载链接，多个链接换行分隔（支持http/https/磁力链接/ed2k）"
          @update:model-value="emit('update:downloadInput', $event)"
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
            :items="props.offlineTargetSelectOptions"
            :model-value="props.offlineTargetSelectValue"
            variant="outlined"
            @update:model-value="emit('select-target', $event)"
          >
            <template #selection="{ item }">
              <span class="text-body-2 text-truncate">{{ item?.raw?.title || props.offlineTargetPathText }}</span>
            </template>

            <template #item="{ props: itemProps, item }">
              <v-list-item
                v-bind="itemProps"
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
          :loading="props.torrentSelecting"
          @click="emit('select-torrent')"
        >
          新建BT任务
        </v-btn>
        <v-btn
          color="primary"
          prepend-icon="mdi-download-outline"
          :loading="props.downloadSubmitting"
          @click="emit('submit')"
        >
          确认下载
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
