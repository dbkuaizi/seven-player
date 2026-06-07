<script setup>
import { Browser } from "@wailsio/runtime";
import { computed, ref, watch } from "vue";
import appIconUrl from "../../assets/seven-player.png";
import CnbIcon from "../icons/CnbIcon.vue";
import {
  capabilityTags,
  playerStatusText,
} from "../../utils/playerPresentation";
import {
  normalizeUIScalePercent,
  themeColorItems,
} from "../../utils/settings";

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
  "save-theme-color",
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
const aboutLinks = [
  {
    title: "个人博客",
    subtitle: "www.dbkuaizi.com/archives/seven-player.html",
    href: "https://www.dbkuaizi.com/archives/seven-player.html",
    icon: "mdi-web",
  },
  {
    title: "CNB 开源地址",
    subtitle: "cnb.cool/dbkuaizi/seven-player",
    href: "https://cnb.cool/dbkuaizi/seven-player",
    customIcon: "cnb",
  },
  {
    title: "GitHub 开源地址",
    subtitle: "github.com/dbkuaizi/seven-player",
    href: "https://github.com/dbkuaizi/seven-player",
    icon: "mdi-github",
  },
];
const playerRecommendationOrder = ["mpv", "potplayer", "vlc", "mpc-be", "mpc-hc"];
const playerRecommendationMeta = {
  mpv: { label: "首选推荐", level: 5, reason: "续播更准确，字幕和跳转支持完整" },
  potplayer: { label: "推荐", level: 4, reason: "Windows 常用，播放兼容性好" },
  vlc: { label: "推荐", level: 3, reason: "跨平台稳定，基础播放可靠" },
  "mpc-be": { label: "兼容", level: 2, reason: "适合本机已安装时作为备用" },
  "mpc-hc": { label: "备用", level: 1, reason: "旧版兼容播放器，优先级最低" },
};

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

const orderedPlayers = computed(() => {
  const orderMap = new Map(
    playerRecommendationOrder.map((id, index) => [id, index]),
  );
  return [...props.playerOptions].sort((left, right) => {
    const leftIndex = orderMap.has(left.id) ? orderMap.get(left.id) : 99;
    const rightIndex = orderMap.has(right.id) ? orderMap.get(right.id) : 99;
    return leftIndex - rightIndex;
  });
});

function isCurrentPlayer(player) {
  return player?.id === props.settings.preferredPlayer && !player.disabled;
}

function canSelectPlayer(player) {
  return hasCustomPlayerPath(player) && !player.disabled;
}

function playerState(player) {
  if (!player?.supported) {
    return {
      icon: "mdi-minus-circle-outline",
      color: "medium-emphasis",
      label: "不支持",
    };
  }
  if (player.disabled) {
    return {
      icon: "mdi-pause-circle-outline",
      color: "medium-emphasis",
      label: "已禁用",
    };
  }
  if (hasCustomPlayerPath(player)) {
    return {
      icon: "mdi-check-circle-outline",
      color: "success",
      label: "已设置",
    };
  }
  if (player.available) {
    return {
      icon: "mdi-check-circle-outline",
      color: "success",
      label: "已检测",
    };
  }
  return {
    icon: "mdi-alert-circle-outline",
    color: "warning",
    label: "未设置",
  };
}

function playerTone(player) {
  return isCurrentPlayer(player)
    ? "primary"
    : playerState(player).color;
}

function playerRecommendation(player) {
  return playerRecommendationMeta[player?.id] || {
    label: "备用",
    level: 1,
    reason: "作为兼容播放器备用",
  };
}

function playerPathHint(player) {
  if (player?.path) {
    return playerStatusText(player);
  }
  return "选择本地可执行文件";
}

function selectPlayer(player) {
  if (canSelectPlayer(player)) {
    emit("select-player", player);
  }
}

async function openAboutLink(url) {
  try {
    await Browser.OpenURL(url);
  } catch {
    window.open(url, "_blank", "noopener,noreferrer");
  }
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
        <v-btn
          class="pill-tab"
          :class="{ 'pill-tab--active': props.settingsTab === 'about' }"
          :color="props.settingsTab === 'about' ? 'primary' : undefined"
          prepend-icon="mdi-information-outline"
          :variant="props.settingsTab === 'about' ? 'flat' : 'text'"
          rounded="pill"
          size="small"
          role="tab"
          :aria-selected="props.settingsTab === 'about'"
          @click="emit('update:settingsTab', 'about')"
        >
          关于
        </v-btn>
      </div>

      <v-window
        :model-value="props.settingsTab"
        :transition="false"
        :reverse-transition="false"
      >
        <v-window-item value="player">
          <v-card-text class="pa-4">
            <div class="settings-panel player-settings-panel">
              <div class="player-settings-head">
                <div>
                  <div class="text-subtitle-1 font-weight-medium">
                    播放器设置
                  </div>
                  <div class="text-caption text-medium-emphasis mt-1">
                    以下为支持的外部播放器，在本地电脑安装后并设置路径后使用。
                  </div>
                </div>
              </div>

              <div class="player-ranked-list">
                <div
                  v-for="(player, index) in orderedPlayers"
                  :key="player.id"
                  class="player-ranked-row"
                  :class="{
                    'player-ranked-row--active': isCurrentPlayer(player),
                    'player-ranked-row--clickable': canSelectPlayer(player),
                    'player-ranked-row--disabled': player.disabled,
                  }"
                  :role="canSelectPlayer(player) ? 'button' : undefined"
                  :tabindex="canSelectPlayer(player) ? 0 : -1"
                  @click="selectPlayer(player)"
                  @keydown.enter.prevent="selectPlayer(player)"
                  @keydown.space.prevent="selectPlayer(player)"
                >
                  <div class="player-rank-number">
                    {{ index + 1 }}
                  </div>

                  <div
                    class="player-status-mark"
                    :class="{ 'player-status-mark--active': isCurrentPlayer(player) }"
                  >
                    <v-icon
                      size="20"
                      :color="playerTone(player)"
                      :icon="playerState(player).icon"
                    />
                  </div>

                  <div class="player-ranked-main">
                    <div class="player-ranked-title">
                      <span>{{ player.name }}</span>
                      <v-chip
                        size="x-small"
                        :color="playerRecommendation(player).level >= 4 ? 'primary' : 'default'"
                        variant="tonal"
                      >
                        {{ playerRecommendation(player).label }}
                      </v-chip>
                      <v-chip
                        v-if="isCurrentPlayer(player)"
                        size="x-small"
                        color="primary"
                        variant="flat"
                      >
                        当前默认
                      </v-chip>
                      <v-chip
                        size="x-small"
                        :color="playerState(player).color"
                        variant="tonal"
                      >
                        {{ playerState(player).label }}
                      </v-chip>
                    </div>

                    <div class="player-ranked-meta">
                      <span>{{ playerRecommendation(player).reason }}</span>
                      <span class="player-ranked-dot">·</span>
                      <span>{{ playerPathHint(player) }}</span>
                    </div>

                    <div class="player-ranked-tags">
                      <v-chip
                        v-for="feature in capabilityTags(player)"
                        :key="`${player.id}-${feature.label}`"
                        size="x-small"
                        :color="feature.color"
                        variant="tonal"
                      >
                        {{ feature.label }}
                      </v-chip>
                      <v-chip size="x-small" variant="tonal">
                        推荐 {{ playerRecommendation(player).level }}/5
                      </v-chip>
                    </div>
                  </div>

                  <div class="player-ranked-actions">
                    <v-btn
                      icon="mdi-folder-open-outline"
                      size="small"
                      variant="text"
                      title="选择路径"
                      aria-label="选择路径"
                      :loading="props.playerLoading"
                      @click.stop="emit('choose-player-path', player.id)"
                    />
                    <template v-if="hasCustomPlayerPath(player)">
                      <v-btn
                        v-if="canSelectPlayer(player) && !isCurrentPlayer(player)"
                        icon="mdi-check-circle-outline"
                        size="small"
                        variant="text"
                        color="primary"
                        title="设为默认播放器"
                        aria-label="设为默认播放器"
                        :loading="props.playerLoading"
                        @click.stop="emit('select-player', player)"
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
                        :aria-label="player.disabled ? '启用播放器' : '禁用播放器'"
                        :loading="props.playerLoading"
                        @click.stop="emit('toggle-player-disabled', player)"
                      />
                      <v-btn
                        icon="mdi-delete-outline"
                        size="small"
                        variant="text"
                        color="error"
                        title="删除已保存路径"
                        aria-label="删除已保存路径"
                        :loading="props.playerLoading"
                        @click.stop="emit('delete-player', player)"
                      />
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </v-card-text>
        </v-window-item>

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

              <div class="settings-row mt-5">
                <div class="settings-row-copy">
                  <div class="text-body-2 font-weight-medium">主题色</div>
                  <div class="text-caption text-medium-emphasis">
                    影响按钮、选中状态、进度条和强调图标。
                  </div>
                </div>

                <v-select
                  class="settings-theme-select"
                  density="compact"
                  hide-details
                  item-title="title"
                  item-value="value"
                  label="主题色"
                  :model-value="props.settings.themeColor"
                  :items="themeColorItems"
                  variant="outlined"
                  @update:model-value="emit('save-theme-color', $event)"
                >
                  <template #selection="{ item }">
                    <div
                      class="theme-color-selection"
                      :style="{
                        '--theme-color-light': item.raw.light,
                        '--theme-color-dark': item.raw.dark,
                      }"
                    >
                      <span>{{ item.raw.title }}</span>
                    </div>
                  </template>
                  <template #item="{ props: itemProps, item }">
                    <v-list-item
                      v-bind="itemProps"
                      :title="item.raw.title"
                    >
                      <template #title>
                        <span
                          class="theme-color-item-title"
                          :style="{
                            '--theme-color-light': item.raw.light,
                            '--theme-color-dark': item.raw.dark,
                          }"
                        >
                          {{ item.raw.title }}
                        </span>
                      </template>
                      <template #append>
                        <v-icon
                          v-if="props.settings.themeColor === item.raw.value"
                          icon="mdi-check"
                          size="18"
                          class="theme-color-check"
                          :style="{
                            '--theme-color-light': item.raw.light,
                            '--theme-color-dark': item.raw.dark,
                          }"
                        />
                      </template>
                    </v-list-item>
                  </template>
                </v-select>
              </div>
            </div>
          </v-card-text>
        </v-window-item>

        <v-window-item value="about">
          <v-card-text class="pa-4">
            <div class="settings-panel about-settings-panel">
              <div class="about-head">
                <div class="about-app-mark">
                  <img
                    class="about-app-icon"
                    :src="appIconUrl"
                    alt=""
                    aria-hidden="true"
                  />
                </div>
                <div class="about-title-block">
                  <div class="text-subtitle-1 font-weight-medium">
                    Seven Player 1.0.0
                  </div>
                  <div class="text-caption text-medium-emphasis mt-1">
                    面向 115 用户的 Windows 外部播放器体验增强工具。
                  </div>
                </div>
              </div>

              <div class="about-info-grid">
                <div class="about-info-item">
                  <div class="about-info-label">作者</div>
                  <div class="about-info-value">两双筷子</div>
                </div>
                <div class="about-info-item">
                  <div class="about-info-label">开源协议</div>
                  <div class="about-info-value">Apache License 2.0</div>
                </div>
                <div class="about-info-item">
                  <div class="about-info-label">使用方式</div>
                  <div class="about-info-value">免费使用，无需购买</div>
                </div>
              </div>

              <div class="about-link-list">
                <button
                  v-for="link in aboutLinks"
                  :key="link.href"
                  type="button"
                  class="about-link-row"
                  @click="openAboutLink(link.href)"
                >
                  <span class="about-link-mark">
                    <CnbIcon
                      v-if="link.customIcon === 'cnb'"
                      class="about-link-cnb-icon"
                      aria-hidden="true"
                    />
                    <v-icon v-else :icon="link.icon" size="20" />
                  </span>
                  <span class="about-link-copy">
                    <span class="about-link-title">{{ link.title }}</span>
                    <span class="about-link-subtitle">{{ link.subtitle }}</span>
                  </span>
                  <v-icon
                    class="about-link-open"
                    icon="mdi-open-in-new"
                    size="18"
                  />
                </button>
              </div>

              <v-divider class="my-5" />

              <div class="about-notice">
                <div class="text-subtitle-2 font-weight-medium">
                  版权与免责声明
                </div>
                <p>
                  Seven Player 以 Apache License 2.0 协议开源，免费提供给用户使用，项目本身不收取费用，也不包含付费解锁或商业售卖行为。
                </p>
                <p>
                  本项目用于改善 115 用户在 Windows 本地电脑上的文件浏览与外部播放器调用体验，不提供内容资源，不存储或分发第三方内容，也不代表 115 官方立场。
                </p>
                <p>
                  使用者应自行确认账号、文件和播放行为符合相关服务条款及适用法律法规；因用户自身使用方式产生的责任，由用户自行承担。
                </p>
              </div>
            </div>
          </v-card-text>
        </v-window-item>
      </v-window>
    </v-card>
  </section>
</template>
