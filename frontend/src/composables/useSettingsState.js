import { computed, ref } from 'vue'
import {
  SaveCleanTitleDisplayEnabled,
  DeletePlayerPath,
  SaveFileListDensity,
  SavePlayerDisabled,
  SavePreferredPlayer,
  SaveSmallFileFilterMB,
  SaveShowTitleBadgesEnabled,
  SaveThemeColor,
  SaveThemeMode,
  SaveUIScalePercent,
  SelectPlayerPath,
} from '../../bindings/sevenplayer/app'
import {
  createDefaultSettings,
  normalizeFileListDensity,
  normalizeSettingsView,
  normalizeThemeColor,
  normalizeThemeMode,
  normalizeUIScalePercent,
} from '../utils/settings'
import { isUserCancelledError } from '../utils/error'

export function useSettingsState({ showNotice, showError, refreshFilePresentation, applyUIScale, applyThemeMode, applyThemeColor }) {
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
    applyUIScale?.(settings.value.uiScalePercent)
    applyThemeMode?.(settings.value.themeMode)
    applyThemeColor?.(settings.value.themeColor)
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

  async function saveCleanTitleDisplay(enabled) {
    try {
      applySettingsView(await SaveCleanTitleDisplayEnabled(Boolean(enabled)))
      refreshFilePresentation()
    } catch (error) {
      showError(error, '保存标题精简显示设置失败')
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

  async function saveUIScale(value) {
    const nextValue = normalizeUIScalePercent(value)

    try {
      applySettingsView(await SaveUIScalePercent(nextValue))
      showNotice('success', `界面缩放已切换为 ${nextValue}%。`)
    } catch (error) {
      showError(error, '保存界面缩放失败')
    }
  }

  async function saveThemeMode(value) {
    const nextValue = normalizeThemeMode(value)

    try {
      applySettingsView(await SaveThemeMode(nextValue))
      showNotice('success', '外观模式已保存。')
    } catch (error) {
      showError(error, '保存外观模式失败')
    }
  }

  async function saveThemeColor(value) {
    const nextValue = normalizeThemeColor(value)

    try {
      applySettingsView(await SaveThemeColor(nextValue))
      showNotice('success', '主题色已保存。')
    } catch (error) {
      showError(error, '保存主题色失败')
    }
  }

  async function choosePlayerPath(playerId = selectedPlayer.value?.id || settings.value?.preferredPlayer || 'mpv') {
    playerLoading.value = true

    try {
      applySettingsView(await SelectPlayerPath(playerId))
      showNotice('success', '播放器路径已保存。')
    } catch (error) {
      if (isUserCancelledError(error)) {
        return
      }
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
    saveCleanTitleDisplay,
    saveUIScale,
    saveThemeMode,
    saveThemeColor,
    saveSmallFileFilter,
    saveFileListDensity,
    choosePlayerPath,
    selectPlayerFromList,
    togglePlayerDisabled,
    deletePlayer,
  }
}
