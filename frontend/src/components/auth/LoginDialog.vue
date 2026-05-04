<script setup>
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  loginTab: { type: String, default: 'qr' },
  loginLoading: { type: Boolean, default: false },
  qrImage: { type: String, default: '' },
  loginStatus: { type: String, default: '' },
  cookieInput: { type: String, default: '' },
  cookieSubmitting: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:modelValue',
  'update:loginTab',
  'update:cookieInput',
  'close',
  'start-login',
  'paste-cookie',
  'submit-cookie',
])

function close() {
  emit('update:modelValue', false)
  emit('close')
}
</script>

<template>
  <v-dialog
    :model-value="props.modelValue"
    max-width="440"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card>
      <div class="login-dialog-header">
        <v-tabs
          :model-value="props.loginTab"
          color="primary"
          class="login-dialog-tabs"
          @update:model-value="emit('update:loginTab', $event)"
        >
          <v-tab value="qr">扫码登录</v-tab>
          <v-tab value="cookie">Cookie 登录</v-tab>
        </v-tabs>
        <v-btn icon="mdi-close" variant="text" size="small" @click="close" />
      </div>
      <v-divider />

      <v-window
        :model-value="props.loginTab"
        @update:model-value="emit('update:loginTab', $event)"
      >
        <v-window-item value="qr">
          <v-card-text>
            <div class="text-body-2 text-medium-emphasis">
              使用 115 App 扫描二维码即可恢复本地会话。如果你已经在浏览器里登录，也可以切到 Cookie 登录。
            </div>

            <div class="qr-shell mt-4">
              <v-progress-circular v-if="props.loginLoading && !props.qrImage" indeterminate color="primary" />
              <v-img
                v-else-if="props.qrImage"
                :src="props.qrImage"
                width="220"
                height="220"
                cover
                class="rounded-lg"
              />
              <v-icon v-else size="64" color="medium-emphasis">mdi-qrcode</v-icon>
            </div>

            <v-alert class="mt-4" variant="tonal" type="info">
              {{ props.loginStatus }}
            </v-alert>
          </v-card-text>

          <v-card-actions class="px-6 pb-4">
            <v-spacer />
            <v-btn color="primary" :loading="props.loginLoading" @click="emit('start-login')">
              刷新二维码
            </v-btn>
          </v-card-actions>
        </v-window-item>

        <v-window-item value="cookie">
          <v-card-text>
            <div class="text-body-2 text-medium-emphasis mb-4">
              粘贴浏览器中 115 的 Cookie，建议至少包含 `UID`、`CID`、`SEID` 和 `KID`。
            </div>

            <v-textarea
              :model-value="props.cookieInput"
              class="scroll-textarea scroll-textarea--cookie"
              rows="4"
              variant="outlined"
              label="Cookie"
              placeholder="UID=...; CID=...; SEID=...; KID=..."
              @update:model-value="emit('update:cookieInput', $event)"
            />
          </v-card-text>

          <v-card-actions class="px-6 pb-4">
            <v-btn variant="text" prepend-icon="mdi-clipboard-text-outline" @click="emit('paste-cookie')">
              从剪贴板粘贴
            </v-btn>
            <v-spacer />
            <v-btn
              color="primary"
              prepend-icon="mdi-cookie-check-outline"
              :loading="props.cookieSubmitting"
              @click="emit('submit-cookie')"
            >
              使用 Cookie 登录
            </v-btn>
          </v-card-actions>
        </v-window-item>
      </v-window>
    </v-card>
  </v-dialog>
</template>
