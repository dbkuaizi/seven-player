<script setup>
import ScraperDirectoryTreeSelect from "./ScraperDirectoryTreeSelect.vue";
import {
  capabilityTags,
  playerStatusText,
} from "../../utils/playerPresentation";

const props = defineProps({
  settingsTab: { type: String, default: "player" },
  settings: { type: Object, required: true },
  playerOptions: { type: Array, default: () => [] },
  playerLoading: { type: Boolean, default: false },
  scraperStatus: { type: Object, default: null },
  scraperActionLoading: { type: Boolean, default: false },
});

const emit = defineEmits([
  "update:settingsTab",
  "select-player",
  "choose-player-path",
  "toggle-player-disabled",
  "delete-player",
  "save-scraper-directories",
  "save-scraper-settings",
  "start-scraper",
  "pause-scraper",
]);

const scraperSourceOptions = [
  {
    title: "TMDB",
    value: "tmdb",
    description: "影视元数据、海报与剧集信息优先来源。",
  },
  {
    title: "豆瓣",
    value: "douban",
    description: "中文片名、简介和本土评分补充来源。",
  },
  {
    title: "Bangumi",
    value: "bangumi",
    description: "动画、番剧类资源的补充来源。",
  },
];

const scraperLanguageOptions = [
  { title: "简体中文", value: "zh-CN" },
  { title: "繁体中文", value: "zh-TW" },
  { title: "English", value: "en-US" },
  { title: "日本語", value: "ja-JP" },
  { title: "한국어", value: "ko-KR" },
];

function updateScraperSettings(patch) {
  emit("save-scraper-settings", patch);
}

function scraperDirectorySummary() {
  const count = Array.isArray(props.settings.scraperDirectories)
    ? props.settings.scraperDirectories.length
    : 0;
  return count ? `限定 ${count} 个目录` : "全部目录";
}

function scraperSourceSummary() {
  const count = enabledScraperSources().length;
  return count ? `${count} 个数据源` : "未启用数据源";
}

function scraperSourceOrderSummary() {
  const enabled = new Set(enabledScraperSources());
  const names = orderedScraperSources()
    .filter((source) => enabled.has(source.value))
    .map((source) => source.title);
  return names.length ? names.join(" -> ") : "未启用数据源";
}

function scraperIsRunning() {
  const status = props.scraperStatus?.status;
  return status === "running" || status === "queued";
}

function scraperStatusSummary() {
  const status = props.scraperStatus;
  if (!status) {
    return "尚未开始刮削";
  }
  if (status.status === "running" || status.status === "queued") {
    return status.message || "正在刮削";
  }
  return status.message || "刮削已结束";
}

function orderedScraperSources() {
  const enabled = Array.isArray(props.settings.scraperSources)
    ? props.settings.scraperSources
    : [];
  const known = new Set();
  const ordered = [];

  for (const value of enabled) {
    const source = scraperSourceOptions.find((item) => item.value === value);
    if (!source || known.has(source.value)) {
      continue;
    }
    known.add(source.value);
    ordered.push(source);
  }

  for (const source of scraperSourceOptions) {
    if (known.has(source.value)) {
      continue;
    }
    ordered.push(source);
  }

  return ordered;
}

function enabledScraperSources() {
  const enabled = new Set(
    Array.isArray(props.settings.scraperSources)
      ? props.settings.scraperSources
      : [],
  );
  return orderedScraperSources()
    .filter((source) => enabled.has(source.value))
    .map((source) => source.value);
}

function toggleScraperSource(source, enabled) {
  const current = new Set(enabledScraperSources());
  if (enabled) {
    current.add(source.value);
  } else {
    current.delete(source.value);
  }

  const next = orderedScraperSources()
    .filter((item) => current.has(item.value))
    .map((item) => item.value);
  updateScraperSettings({ sources: next });
}

function moveScraperSource(source, direction) {
  const list = orderedScraperSources();
  const index = list.findIndex((item) => item.value === source.value);
  const targetIndex = index + direction;
  if (index < 0 || targetIndex < 0 || targetIndex >= list.length) {
    return;
  }

  const nextList = [...list];
  const [current] = nextList.splice(index, 1);
  nextList.splice(targetIndex, 0, current);

  const enabled = new Set(enabledScraperSources());
  updateScraperSettings({
    sources: nextList
      .filter((item) => enabled.has(item.value))
      .map((item) => item.value),
  });
}

function handleSourceDragStart(event, source) {
  event.dataTransfer.effectAllowed = "move";
  event.dataTransfer.setData("text/plain", source.value);
}

function handleSourceDrop(event, target) {
  const sourceValue = event.dataTransfer.getData("text/plain");
  if (!sourceValue || sourceValue === target.value) {
    return;
  }

  const list = orderedScraperSources();
  const fromIndex = list.findIndex((item) => item.value === sourceValue);
  const toIndex = list.findIndex((item) => item.value === target.value);
  if (fromIndex < 0 || toIndex < 0) {
    return;
  }

  const nextList = [...list];
  const [source] = nextList.splice(fromIndex, 1);
  nextList.splice(toIndex, 0, source);

  const enabled = new Set(enabledScraperSources());
  updateScraperSettings({
    sources: nextList
      .filter((item) => enabled.has(item.value))
      .map((item) => item.value),
  });
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
          :class="{ 'pill-tab--active': props.settingsTab === 'scraper' }"
          :color="props.settingsTab === 'scraper' ? 'primary' : undefined"
          prepend-icon="mdi-auto-fix"
          :variant="props.settingsTab === 'scraper' ? 'flat' : 'text'"
          rounded="pill"
          size="small"
          role="tab"
          :aria-selected="props.settingsTab === 'scraper'"
          @click="emit('update:settingsTab', 'scraper')"
        >
          刮削
        </v-btn>
        <v-btn
          class="pill-tab"
          disabled
          prepend-icon="mdi-tune-variant"
          rounded="pill"
          size="small"
          variant="text"
          role="tab"
          aria-selected="false"
        >
          更多设置
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

        <v-window-item value="scraper">
          <v-card-text class="pa-4">
            <div class="settings-panel scraper-settings-panel">
              <div class="text-subtitle-1 font-weight-medium">刮削设置</div>
              <div class="text-caption text-medium-emphasis mt-1 mb-3">
                后续用于识别影视文件并补全海报、简介、演员、评分和剧集信息。
              </div>

              <div class="scraper-start-panel mb-3">
                <div class="scraper-start-copy">
                  <div class="text-body-2 font-weight-medium">手动刮削任务</div>
                  <div class="text-caption text-medium-emphasis">
                    {{ scraperDirectorySummary() }} ·
                    {{ scraperSourceSummary() }} · 语言
                    {{ props.settings.scraperLanguage || "zh-CN" }}
                  </div>
                  <div class="text-caption text-medium-emphasis mt-1">
                    {{ scraperStatusSummary() }}
                  </div>
                </div>
                <v-btn
                  :color="scraperIsRunning() ? undefined : 'primary'"
                  :prepend-icon="
                    scraperIsRunning()
                      ? 'mdi-pause-circle-outline'
                      : 'mdi-play-circle-outline'
                  "
                  size="small"
                  :disabled="
                    !scraperIsRunning() && enabledScraperSources().length === 0
                  "
                  :loading="props.scraperActionLoading"
                  :variant="scraperIsRunning() ? 'tonal' : 'flat'"
                  @click="
                    scraperIsRunning()
                      ? emit('pause-scraper')
                      : emit('start-scraper')
                  "
                >
                  {{ scraperIsRunning() ? "暂停刮削" : "开始刮削" }}
                </v-btn>
              </div>

              <ScraperDirectoryTreeSelect
                :model-value="props.settings.scraperDirectories"
                @save="emit('save-scraper-directories', $event)"
              />

              <v-row dense>
                <v-col cols="12" sm="6">
                  <v-menu
                    :close-on-content-click="false"
                    location="bottom start"
                    max-height="360"
                    offset="6"
                  >
                    <template #activator="{ props: menuProps }">
                      <v-text-field
                        v-bind="menuProps"
                        class="scraper-source-activator"
                        density="compact"
                        hide-details="auto"
                        label="数据源优先级"
                        :model-value="scraperSourceOrderSummary()"
                        persistent-placeholder
                        prepend-inner-icon="mdi-database-search-outline"
                        readonly
                        variant="outlined"
                      />
                    </template>

                    <v-card class="scraper-source-menu" elevation="8">
                      <div class="scraper-source-header">
                        <div>
                          <div class="text-body-2 font-weight-medium">
                            数据源优先级
                          </div>
                          <div class="text-caption text-medium-emphasis">
                            勾选启用；拖拽左侧图标或用箭头调整尝试顺序。
                          </div>
                        </div>
                        <v-chip size="x-small" color="primary" variant="tonal">
                          {{ enabledScraperSources().length }} 个启用
                        </v-chip>
                      </div>

                      <div class="scraper-source-list">
                        <div
                          v-for="(source, index) in orderedScraperSources()"
                          :key="source.value"
                          class="scraper-source-item"
                          @dragover.prevent
                          @drop.prevent="handleSourceDrop($event, source)"
                        >
                          <v-icon
                            class="scraper-source-drag"
                            draggable="true"
                            size="18"
                            @dragstart="handleSourceDragStart($event, source)"
                          >
                            mdi-drag
                          </v-icon>
                          <v-btn
                            class="scraper-source-check"
                            :color="
                              enabledScraperSources().includes(source.value)
                                ? 'primary'
                                : undefined
                            "
                            :icon="
                              enabledScraperSources().includes(source.value)
                                ? 'mdi-checkbox-marked'
                                : 'mdi-checkbox-blank-outline'
                            "
                            size="x-small"
                            variant="text"
                            @click.stop="
                              toggleScraperSource(
                                source,
                                !enabledScraperSources().includes(source.value),
                              )
                            "
                          />
                          <div class="scraper-source-text">
                            <div class="scraper-source-title">
                              {{ source.title }}
                            </div>
                            <div class="scraper-source-desc">
                              {{ source.description }}
                            </div>
                          </div>
                          <div class="scraper-source-actions">
                            <v-btn
                              icon="mdi-arrow-up"
                              size="x-small"
                              variant="text"
                              :disabled="index === 0"
                              @click="moveScraperSource(source, -1)"
                            />
                            <v-btn
                              icon="mdi-arrow-down"
                              size="x-small"
                              variant="text"
                              :disabled="
                                index === scraperSourceOptions.length - 1
                              "
                              @click="moveScraperSource(source, 1)"
                            />
                          </div>
                        </div>
                      </div>
                    </v-card>
                  </v-menu>
                </v-col>

                  <v-col cols="12" sm="6">
                    <v-select
                      density="compact"
                      hide-details
                      label="刮削语言"
                    :items="scraperLanguageOptions"
                    :model-value="props.settings.scraperLanguage"
                    variant="outlined"
                      @update:model-value="
                        updateScraperSettings({ language: $event })
                      "
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-text-field
                      density="compact"
                      hide-details="auto"
                      label="TMDB Read Access Token"
                      :model-value="props.settings.tmdbReadAccessToken"
                      placeholder="粘贴 TMDB API Read Access Token (v4 auth)"
                      prepend-inner-icon="mdi-key-variant"
                      variant="outlined"
                      @update:model-value="
                        updateScraperSettings({ tmdbReadAccessToken: $event })
                      "
                    />
                  </v-col>
              </v-row>

              <v-list class="scraper-option-list mt-3" density="compact">
                <v-list-item rounded="lg">
                  <template #prepend>
                    <v-icon color="primary">mdi-folder-search-outline</v-icon>
                  </template>
                  <v-list-item-title>自动识别新增资源</v-list-item-title>
                  <v-list-item-subtitle>
                    文件入库后自动匹配影片、剧集、海报和基础信息。
                  </v-list-item-subtitle>
                  <template #append>
                    <v-switch
                      color="primary"
                      density="compact"
                      hide-details
                      :model-value="props.settings.scraperAutoScan"
                      @update:model-value="
                        updateScraperSettings({ autoScan: $event })
                      "
                    />
                  </template>
                </v-list-item>

                <v-list-item rounded="lg">
                  <template #prepend>
                    <v-icon color="primary">mdi-database-sync-outline</v-icon>
                  </template>
                  <v-list-item-title>覆盖已有刮削结果</v-list-item-title>
                  <v-list-item-subtitle>
                    重新刮削时允许用新结果替换已保存的海报和简介。
                  </v-list-item-subtitle>
                  <template #append>
                    <v-switch
                      color="primary"
                      density="compact"
                      hide-details
                      :model-value="props.settings.scraperOverwrite"
                      @update:model-value="
                        updateScraperSettings({ overwrite: $event })
                      "
                    />
                  </template>
                </v-list-item>

                <v-list-item rounded="lg">
                  <template #prepend>
                    <v-icon color="primary">mdi-image-multiple-outline</v-icon>
                  </template>
                  <v-list-item-title>下载海报与背景图</v-list-item-title>
                  <v-list-item-subtitle>
                    将影视墙需要的图片缓存到本地，减少重复加载。
                  </v-list-item-subtitle>
                  <template #append>
                    <v-switch
                      color="primary"
                      density="compact"
                      hide-details
                      :model-value="props.settings.scraperDownloadImages"
                      @update:model-value="
                        updateScraperSettings({ downloadImages: $event })
                      "
                    />
                  </template>
                </v-list-item>
              </v-list>

              <v-alert
                class="mt-3"
                density="compact"
                type="info"
                variant="tonal"
              >
                建议将 TMDB 放在首位，并配置可用的 Read Access Token。开始刮削后会递归扫描所选
                115 目录，把视频文件聚合成影视条目，再按数据源优先级补全海报、背景图、演员、评分与剧集信息。
              </v-alert>
            </div>
          </v-card-text>
        </v-window-item>
      </v-window>
    </v-card>
  </section>
</template>
