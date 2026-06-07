<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '选择保存目录' },
  breadcrumbItems: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  folders: { type: Array, default: () => [] },
})

const emit = defineEmits([
  'update:modelValue',
  'close',
  'choose-current',
  'open-breadcrumb',
  'open-directory',
])

function close() {
  emit('update:modelValue', false)
  emit('close')
}
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="700"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card>
      <v-card-title>{{ props.title }}</v-card-title>
      <v-divider />

      <v-toolbar density="compact" flat class="px-2 folder-picker-toolbar">
        <div class="breadcrumb-strip">
          <div class="path-bar folder-picker-path-bar">
            <v-breadcrumbs
              class="file-breadcrumbs pa-0"
              :items="props.breadcrumbItems"
              divider="›"
            >
              <template #prepend>
                <v-icon size="16" color="medium-emphasis">mdi-folder-outline</v-icon>
              </template>

              <template #title="{ item }">
                <button
                  type="button"
                  class="file-breadcrumb-link"
                  :class="{
                    'file-breadcrumb-link--disabled': item.disabled,
                    'file-breadcrumb-link--last': item.isLast,
                  }"
                  :disabled="item.disabled"
                  :title="item.rawTitle || item.title"
                  @click="emit('open-breadcrumb', item.id)"
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
          @click="emit('choose-current')"
        >
          选择
        </v-btn>
      </v-toolbar>

      <v-progress-linear :active="props.loading" :indeterminate="props.loading" height="2" />

      <v-card-text class="pa-0">
        <v-list density="compact">
          <v-list-item
            v-for="folder in props.folders"
            :key="folder.rowKey"
            prepend-icon="mdi-folder-outline"
            :title="folder.name"
            :subtitle="folder.updatedText"
            @click="emit('open-directory', folder)"
          />

          <v-list-item v-if="!props.loading && !props.folders.length" title="当前目录没有子文件夹" />
        </v-list>
      </v-card-text>

      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="close">取消</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
