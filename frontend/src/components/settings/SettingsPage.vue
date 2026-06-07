<script setup>
import { ref, watch } from "vue";
import {
  capabilityTags,
  playerStatusText,
} from "../../utils/playerPresentation";
import { normalizeUIScalePercent } from "../../utils/settings";

const props = defineProps({
  settingsTab: { type: String, default: "player" },
  settings: { type: Object, required: true },
  playerOptions: { type: Array, default: () => [] },
  playerLoading: { type: Boolean, default: false },
});

const emit = defineEmits([
  "update:settingsTab",
  "preview-ui-scale",
  "save-ui-scale",
  "save-theme-mode",
  "select-player",
  "choose-player-path",
  "toggle-player-disabled",
  "delete-player",
]);

const uiScaleDraft = ref(normalizeUIScalePercent(props.settings.uiScalePercent));
const themeModeItems = [
  { title: "跟随系统", value: "system", icon: "mdi-monitor" },
  { title: "浅色", value: "light", icon: "mdi-white-balance-sunny" },
  { title: "深色", value: "dark", icon: "mdi-weather-night" },
];

watch(
  () => props.settings.uiScalePercent,
  (value) => {
    uiScaleDraft.value = normalizeUIScalePercent(value);
  },
)

function previewUIScale(value) {
  const nextValue = normalizeUIScalePercent(value);
  uiScaleDraft.value = nextValue;
  emit("preview-ui-scale", nextValue);
}

function saveUIScale(value = uiScaleDraft.value) {
  const nextValue = normalizeUIScalePercent(value);
  uiScaleDraft.value = nextValue;
  emit("save-ui-scale", nextValue);
}

function hasCustomPlayerPath(player) {
  return player?.source === "custom";
}
</script>

<template>
  <section class="page-section">
    <v-card class="section-card">
      <div class="settings-pill-tabs" role="tablist" aria-label="系统设置分类">
        <v-btn
          class="pill-tab"
          :class="{ 'pill-tab--active': props.settingsTab === 'player' }"
          :color="props.settingsTab === 'player' ? 'primary' : undefined"
          prepend-icon="mdi-play-box-outline"
          :variant="props.settingsTab === 'player' ? 'flat' : 'text'"
          rounded="pill"
          size="small"
          role="tab"
          :aria-selected="props.settingsTab === 'player'"
          @click="emit('update:settingsTab', 'player')"
        >
          播放器
        </v-btn>
        <v-btn
          class="pill-tab"
          :class="{ 'pill-tab--active': props.settingsTab === 'display' }"
          :color="props.settingsTab === 'display' ? 'primary' : undefined"
          prepend-icon="mdi-format-size"
          :variant="props.settingsTab === 'display' ? 'flat' : 'text'"
          rounded="pill"
          size="small"
          role="tab"
          :aria-selected="props.settingsTab === 'display'"
          @click="emit('update:settingsTab', 'display')"
        >
          显示
        </v-btn>
      </div>

      <v-window :model-value="props.settingsTab">
        <v-window-item value="display">
          <v-card-text class="pa-4">
            <div class="settings-panel">
              <div class="text-subtitle-1 font-weight-medium">显示设置</div>
              <div class="settings-row mt-4">
                <div class="settings-row-copy">
                  <div class="text-body-2 font-weight-medium">界面缩放</div>
                  <div class="text-caption text-medium-emphasis">
                    放大整体文字和控件尺寸。
                  </div>
                </div>

                <div class="settings-scale-control">
                  <v-slider
                    class="settings-scale-slider"
                    color="primary"
                    density="compact"
                    hide-details
                    :model-value="uiScaleDraft"
                    :min="100"
                    :max="150"
                    :step="5"
                    show-ticks="always"
                    tick-size="3"
                    @update:model-value="previewUIScale"
                    @end="saveUIScale"
                  />
                  <div class="settings-scale-value">
                    {{ uiScaleDraft }}%
                  </div>
                </div>
              </div>

              <div class="settings-row mt-5">
                <div class="settings-row-copy">
                  <div class="text-body-2 font-weight-medium">外观模式</div>
                  <div class="text-caption text-medium-emphasis">
                    切换浅色、深色或跟随系统。
                  </div>
                </div>

                <v-select
                  class="settings-theme-select"
                  density="compact"
                  hide-details
                  item-title="title"
                  item-value="value"
                  label="外观模式"
                  :model-value="props.settings.themeMode"
                  :items="themeModeItems"
                  variant="outlined"
                  @update:model-value="emit('save-theme-mode', $event)"
                >
                  <template #selection="{ item }">
                    <div class="settings-theme-selection">
                      <v-icon size="18" :icon="item.raw.icon" />
                      <span>{{ item.raw.title }}</span>
                    </div>
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
            </div>
          </v-card-text>
        </v-window-item>

        <v-window-item value="player">
          <v-card-text class="pa-4">
            <div class="settings-panel">
              <div class="text-subtitle-1 font-weight-medium">播放器设置</div>
              <div class="text-caption text-medium-emphasis mt-1 mb-2">
                以下为支持的外部播放器，在本地电脑安装后并设置路径后使用。点击已设置路径的播放器可切换默认播放器。
              </div>

              <v-list class="player-list" density="compact" lines="two">
                <v-list-item
                  v-for="player in props.playerOptions"
                  :key="player.id"
                  rounded="lg"
                  class="player-list-item"
                  :class="{ 'player-list-item--disabled': player.disabled }"
                  :active="
                    player.id === props.settings.preferredPlayer &&
                    !player.disabled
                  "
                  @click="emit('select-player', player)"
                >
                  <template #prepend>
                    <v-icon
                      :color="
                        player.disabled
                          ? 'medium-emphasis'
                          : player.available
                            ? 'success'
                            : 'warning'
                      "
                    >
                      {{
                        player.disabled
                          ? "mdi-pause-circle-outline"
                          : player.available
                            ? "mdi-check-circle-outline"
                            : "mdi-alert-circle-outline"
                      }}
                    </v-icon>
                  </template>

                  <template #title>
                    <div class="player-row-title">
                      <span class="player-title">{{ player.name }}</span>
                      <div class="player-inline-badges">
                        <v-chip
                          v-if="
                            player.id === props.settings.preferredPlayer &&
                            !player.disabled
                          "
                          size="x-small"
                          color="primary"
                          variant="tonal"
                        >
                          当前默认
                        </v-chip>
                        <v-chip
                          size="x-small"
                          :color="
                            player.disabled
                              ? 'default'
                              : player.available
                                ? 'success'
                                : 'warning'
                          "
                          variant="tonal"
                        >
                          {{
                            player.disabled
                              ? "已禁用"
                              : player.available
                                ? "可用"
                                : "未就绪"
                          }}
                        </v-chip>
                        <v-chip
                          v-for="feature in capabilityTags(player)"
                          :key="`${player.id}-${feature.label}`"
                          size="x-small"
                          :color="feature.color"
                          variant="tonal"
                        >
                          {{ feature.label }}
                        </v-chip>
                      </div>
                    </div>
                  </template>

                  <template #subtitle>
                    <div class="player-subtitle text-truncate">
                      {{ playerStatusText(player) }}
                    </div>
                  </template>

                  <template #append>
                    <div class="player-actions">
                      <v-btn
                        icon="mdi-folder-open-outline"
                        size="small"
                        variant="text"
                        title="选择路径"
                        :loading="props.playerLoading"
                        @click.stop="emit('choose-player-path', player.id)"
                      />
                      <v-btn
                        v-if="hasCustomPlayerPath(player)"
                        :icon="
                          player.disabled
                            ? 'mdi-play-circle-outline'
                            : 'mdi-pause-circle-outline'
                        "
                        size="small"
                        variant="text"
                        :title="player.disabled ? '启用播放器' : '禁用播放器'"
                        :loading="props.playerLoading"
                        @click.stop="emit('toggle-player-disabled', player)"
                      />
                      <v-btn
                        v-if="hasCustomPlayerPath(player)"
                        icon="mdi-delete-outline"
                        size="small"
                        variant="text"
                        color="error"
                        title="删除已保存路径"
                        :loading="props.playerLoading"
                        @click.stop="emit('delete-player', player)"
                      />
                    </div>
                  </template>
                </v-list-item>
              </v-list>
            </div>
          </v-card-text>
        </v-window-item>
      </v-window>
    </v-card>
  </section>
</template>
