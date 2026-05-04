import { computed, ref } from 'vue'
import {
  DeletePlayerPath,
  SaveFileListDensity,
  SavePlayerDisabled,
  SavePreferredPlayer,
  SaveScraperDirectories,
  SaveScraperSettings,
  SaveSmallFileFilterMB,
  SaveShowTitleBadgesEnabled,
  SelectPlayerPath,
} from '../../bindings/panplayer/app'
import {
  createDefaultSettings,
  normalizeFileListDensity,
  normalizeScraperLanguage,
  normalizeScraperSources,
  normalizeSettingsView,
} from '../utils/settings'
import { normalizeDirectoryTargets } from '../utils/directoryTarget'

export function useSettingsState({ showNotice, showError, refreshFilePresentation }) {
  const settingsTab = ref('player')
  const settings = ref(createDefaultSettings())
  const playerLoading = ref(false)
  const smallFileFilterMB = ref(0)

  const playerOptions = computed(() => settings.value?.players ?? [])

  const selectedPlayer = computed(() => {
    const preferred = settings.value?.preferredPlayer ?? 'mpv'
    return playerOptions.value.find((item) => item.id === preferred) ?? playerOptions.value[0] ?? null
  })

  function applySettingsView(view) {
    settings.value = normalizeSettingsView(view)
    smallFileFilterMB.value = Number(settings.value.smallFileFilterMB || 0)
  }

  async function changePreferredPlayer(playerId) {
    if (!playerId) {
      return
    }

    const target = playerOptions.value.find((item) => item.id === playerId)
    if (target?.disabled) {
      showNotice('warning', '已禁用的播放器不能设为默认。')
      return
    }

    playerLoading.value = true

    try {
      applySettingsView(await SavePreferredPlayer(playerId))
      showNotice('success', `默认播放器已切换为 ${selectedPlayer.value?.name || playerId}。`)
    } catch (error) {
      showError(error, '切换播放器失败')
    } finally {
      playerLoading.value = false
    }
  }

  async function saveShowTitleBadges(enabled) {
    try {
      applySettingsView(await SaveShowTitleBadgesEnabled(Boolean(enabled)))
      refreshFilePresentation()
    } catch (error) {
      showError(error, '保存徽章显示设置失败')
    }
  }

  async function saveSmallFileFilter(value) {
    const nextValue = Number(value || 0)
    smallFileFilterMB.value = nextValue

    try {
      applySettingsView(await SaveSmallFileFilterMB(nextValue))
    } catch (error) {
      smallFileFilterMB.value = Number(settings.value?.smallFileFilterMB || 0)
      showError(error, '保存小文件屏蔽规则失败')
    }
  }

  async function saveFileListDensity(value) {
    const nextValue = normalizeFileListDensity(value)

    try {
      applySettingsView(await SaveFileListDensity(nextValue))
    } catch (error) {
      showError(error, '保存文件列表密度失败')
    }
  }

  async function saveScraperDirectories(targets) {
    const nextTargets = normalizeDirectoryTargets(targets, 50)

    try {
      applySettingsView(await SaveScraperDirectories(nextTargets))
      showNotice('success', nextTargets.length ? '刮削目录已保存。' : '已恢复为不限制刮削目录。')
    } catch (error) {
      showError(error, '保存刮削目录失败')
    }
  }

  async function saveScraperSettings(patch = {}) {
    const current = settings.value || createDefaultSettings()
    const next = {
      sources: normalizeScraperSources(patch.sources ?? current.scraperSources),
      language: normalizeScraperLanguage(patch.language ?? current.scraperLanguage),
      autoScan: Boolean(patch.autoScan ?? current.scraperAutoScan),
      overwrite: Boolean(patch.overwrite ?? current.scraperOverwrite),
      downloadImages: patch.downloadImages ?? current.scraperDownloadImages,
      tmdbReadAccessToken: String(
        patch.tmdbReadAccessToken ?? current.tmdbReadAccessToken ?? '',
      ).trim(),
    }

    try {
      applySettingsView(await SaveScraperSettings(
        next.sources,
        next.language,
        next.autoScan,
        next.overwrite,
        next.downloadImages !== false,
        next.tmdbReadAccessToken,
      ))
      showNotice('success', '刮削设置已保存。')
    } catch (error) {
      showError(error, '保存刮削设置失败')
    }
  }

  async function choosePlayerPath(playerId = selectedPlayer.value?.id || settings.value?.preferredPlayer || 'mpv') {
    playerLoading.value = true

    try {
      applySettingsView(await SelectPlayerPath(playerId))
      showNotice('success', '播放器路径已保存。')
    } catch (error) {
      showError(error, '设置播放器路径失败')
    } finally {
      playerLoading.value = false
    }
  }

  async function selectPlayerFromList(player) {
    if (!player || player.disabled) {
      return
    }
    await changePreferredPlayer(player.id)
  }

  async function togglePlayerDisabled(player) {
    if (!player?.id) {
      return
    }

    playerLoading.value = true

    try {
      const nextDisabled = !player.disabled
      applySettingsView(await SavePlayerDisabled(player.id, nextDisabled))
      showNotice('success', nextDisabled ? `${player.name} 已禁用。` : `${player.name} 已启用。`)
    } catch (error) {
      showError(error, player?.disabled ? '启用播放器失败' : '禁用播放器失败')
    } finally {
      playerLoading.value = false
    }
  }

  async function deletePlayer(player) {
    if (!player?.id) {
      return
    }

    playerLoading.value = true

    try {
      applySettingsView(await DeletePlayerPath(player.id))
      showNotice('success', `${player.name} 的已保存路径已删除。`)
    } catch (error) {
      showError(error, '删除播放器路径失败')
    } finally {
      playerLoading.value = false
    }
  }

  return {
    settingsTab,
    settings,
    playerLoading,
    smallFileFilterMB,
    playerOptions,
    selectedPlayer,
    applySettingsView,
    saveShowTitleBadges,
    saveSmallFileFilter,
    saveFileListDensity,
    saveScraperDirectories,
    saveScraperSettings,
    choosePlayerPath,
    selectPlayerFromList,
    togglePlayerDisabled,
    deletePlayer,
  }
}
