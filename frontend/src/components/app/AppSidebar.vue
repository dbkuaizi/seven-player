<script setup>
const props = defineProps({
  activeSection: { type: String, required: true },
  navigationItems: { type: Array, default: () => [] },
  loggedIn: { type: Boolean, default: false },
  user: { type: Object, default: null },
  accountAvatarUrl: { type: String, default: '' },
  accountDisplayName: { type: String, default: '未登录' },
  accountVipLevelText: { type: String, default: '--' },
  accountVipExpireText: { type: String, default: '--' },
  accountSpaceUsageText: { type: String, default: '--' },
  accountSpacePercentText: { type: String, default: '--' },
  accountSpacePercent: { type: Number, default: 0 },
  hiddenModeEnabled: { type: Boolean, default: false },
  hiddenModeLoading: { type: Boolean, default: false },
  actionLoading: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:activeSection',
  'login',
  'logout',
])
</script>

<template>
  <v-navigation-drawer permanent width="220" class="sidebar">
    <div class="sidebar-layout">
      <v-list nav density="compact" class="sidebar-nav pa-2 pt-3">
        <v-list-item
          v-for="item in props.navigationItems"
          :key="item.value"
          rounded="lg"
          :active="props.activeSection === item.value"
          :title="item.label"
          :prepend-icon="item.icon"
          @click="emit('update:activeSection', item.value)"
        />
      </v-list>

      <div class="sidebar-footer pa-2 pt-0">
        <v-divider class="mb-2" />

        <v-menu v-if="props.loggedIn" location="top start">
          <template #activator="{ props: menuProps }">
            <v-card
              class="account-panel"
              rounded="lg"
              variant="flat"
              role="button"
              tabindex="0"
              :aria-controls="menuProps['aria-controls']"
              :aria-expanded="menuProps['aria-expanded']"
              :aria-haspopup="menuProps['aria-haspopup']"
              @click="menuProps.onClick"
            >
              <v-card-item class="account-panel-head pt-2 pb-1">
                <template #prepend>
                  <div v-if="props.accountAvatarUrl" class="account-avatar-image-shell account-avatar-image-shell--large" aria-hidden="true">
                    <img class="account-avatar-image" :src="props.accountAvatarUrl" alt="" />
                  </div>
                  <v-avatar
                    v-else
                    size="42"
                    class="account-avatar"
                    color="primary"
                    variant="tonal"
                  >
                    <v-icon>mdi-account-circle-outline</v-icon>
                  </v-avatar>
                </template>

                <v-card-title class="account-panel-title">
                  <div class="account-panel-title-stack">
                    <div class="account-panel-name text-truncate">{{ props.accountDisplayName }}</div>
                    <div class="account-panel-meta" v-if="props.loggedIn && props.accountVipLevelText !== '--'">
                      <span
                        v-if="props.user?.isVip"
                        class="account-vip-inline-pill"
                      >
                        VIP
                      </span>
                      <span
                        v-else
                        class="account-vip-inline-text"
                      >
                        {{ props.accountVipLevelText }}
                      </span>
                      <span
                        v-if="props.user?.isVip && props.accountVipExpireText !== '--' && props.accountVipExpireText !== '非 VIP'"
                        class="account-vip-inline-expire-text"
                      >
                        {{ props.accountVipExpireText }}
                      </span>
                    </div>
                  </div>
                </v-card-title>
              </v-card-item>

              <v-card-text class="account-panel-body">
                <div class="account-summary-row">
                  <div class="account-summary-main">
                    <div class="account-summary-label">空间占用</div>
                    <div class="account-summary-value">{{ props.accountSpaceUsageText }}</div>
                  </div>
                  <div class="account-summary-side">{{ props.accountSpacePercentText }}</div>
                </div>

                <v-progress-linear
                  class="account-space-bar"
                  :model-value="props.accountSpacePercent"
                  color="primary"
                  rounded
                  height="8"
                />
              </v-card-text>
            </v-card>
          </template>

          <v-list density="compact" min-width="180">
            <v-list-item
              prepend-icon="mdi-qrcode"
              title="重新扫码登录"
              @click="emit('login', 'qr')"
            />
            <v-list-item
              prepend-icon="mdi-cookie-outline"
              title="Cookie 登录"
              @click="emit('login', 'cookie')"
            />
            <v-divider class="my-1" />
            <v-list-item
              prepend-icon="mdi-logout"
              title="退出登录"
              :disabled="props.actionLoading"
              @click="emit('logout')"
            />
          </v-list>
        </v-menu>

        <v-card
          v-else
          class="account-panel account-panel--guest"
          rounded="lg"
          variant="flat"
          @click="emit('login', 'qr')"
        >
          <v-card-item class="account-panel-head account-panel-head--guest">
            <template #prepend>
              <v-avatar size="42" color="primary" variant="tonal">
                <v-icon>mdi-account-outline</v-icon>
              </v-avatar>
            </template>

            <v-card-title class="account-panel-title">
              未登录
            </v-card-title>
            <v-card-subtitle class="account-panel-subtitle">
              点击登录 115
            </v-card-subtitle>

            <template #append>
              <v-icon size="18">mdi-login</v-icon>
            </template>
          </v-card-item>
        </v-card>
      </div>
    </div>
  </v-navigation-drawer>
</template>
