<script setup>
import {
  capabilityTags,
  playerStatusText,
} from "../../utils/playerPresentation";

const props = defineProps({
  settingsTab: { type: String, default: "player" },
  settings: { type: Object, required: true },
  playerOptions: { type: Array, default: () => [] },
  playerLoading: { type: Boolean, default: false },
});

const emit = defineEmits([
  "update:settingsTab",
  "select-player",
  "choose-player-path",
  "toggle-player-disabled",
  "delete-player",
]);
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
      </div>

      <v-window :model-value="props.settingsTab">
        <v-window-item value="player">
          <v-card-text class="pa-4">
            <div class="settings-panel">
              <div class="text-subtitle-1 font-weight-medium">播放器设置</div>
              <div class="text-caption text-medium-emphasis mt-1 mb-2">
                点击列表项即可切换默认播放器。
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
