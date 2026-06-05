<script setup>
import { ref } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  password: { type: String, default: '' },
  rememberPassword: { type: Boolean, default: false },
  rememberedPasswordAvailable: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:modelValue',
  'update:password',
  'update:rememberPassword',
  'close',
  'confirm',
])

const passwordVisible = ref(false)

function close() {
  passwordVisible.value = false
  emit('update:modelValue', false)
  emit('close')
}
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="360"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card>
      <div class="login-dialog-header">
        <div class="text-subtitle-1 font-weight-medium">开启隐私模式</div>
        <v-btn icon="mdi-close" variant="text" size="small" @click="close" />
      </div>
      <v-divider />

      <v-card-text>
        <div class="text-body-2 text-medium-emphasis mb-4">
          输入 115 隐私模式密码，开启后会展示隐私目录内容。
        </div>

        <v-text-field
          :model-value="props.password"
          autofocus
          density="comfortable"
          hide-details="auto"
          label="隐私模式密码"
          :type="passwordVisible ? 'text' : 'password'"
          variant="outlined"
          :append-inner-icon="passwordVisible ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
          @update:model-value="emit('update:password', $event)"
          @click:append-inner="passwordVisible = !passwordVisible"
          @keydown.enter.prevent="emit('confirm')"
        />

        <v-checkbox
          :model-value="props.rememberPassword"
          class="mt-2"
          color="primary"
          density="compact"
          hide-details
          :label="props.rememberedPasswordAvailable ? '已记住密码摘要，下次可直接开启' : '记住密码摘要，下次无需重新输入'"
          @update:model-value="emit('update:rememberPassword', $event)"
        />
      </v-card-text>

      <v-card-actions class="px-6 pb-4">
        <v-spacer />
        <v-btn variant="text" @click="close">取消</v-btn>
        <v-btn
          color="primary"
          prepend-icon="mdi-shield-lock-open-outline"
          :loading="props.loading"
          @click="emit('confirm')"
        >
          开启
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
