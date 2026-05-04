<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  jumpInput: { type: String, default: '' },
})

const emit = defineEmits([
  'update:modelValue',
  'update:jumpInput',
  'confirm',
])
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="420"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card>
      <v-card-title>跳转播放</v-card-title>
      <v-divider />
      <v-card-text>
        <v-text-field
          :model-value="props.jumpInput"
          label="时间点"
          placeholder="例如 90、01:30、01:02:03"
          @update:model-value="emit('update:jumpInput', $event)"
        />
        <div class="text-caption text-medium-emphasis">
          输入秒数或时分秒，播放器会从该时间点开始拉起。
        </div>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="emit('update:modelValue', false)">取消</v-btn>
        <v-btn color="primary" @click="emit('confirm')">开始播放</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
