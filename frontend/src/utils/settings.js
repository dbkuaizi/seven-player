import { normalizeOfflineRecentTargets } from './directoryTarget'

export function normalizeFileListDensity(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'compact' || normalized === 'comfortable') {
    return normalized
  }
  return 'default'
}

export function normalizeUIScalePercent(value) {
  const normalized = Number(value || 0)
  if (normalized < 100) {
    return 100
  }
  if (normalized > 150) {
    return 150
  }
  return Math.round(normalized / 5) * 5
}

export function normalizeThemeMode(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'light' || normalized === 'dark') {
    return normalized
  }
  return 'system'
}

export function createDefaultSettings() {
  return {
    preferredPlayer: 'mpv',
    players: [],
    configPath: '',
    showTitleBadges: true,
    cleanTitleDisplay: true,
    uiScalePercent: 100,
    themeMode: 'system',
    smallFileFilterMB: 0,
    fileListDensity: 'default',
    offlineRecentTargets: [],
  }
}

export function normalizeSettingsView(view) {
  return {
    preferredPlayer: view?.preferredPlayer || 'mpv',
    players: Array.isArray(view?.players) ? view.players : [],
    configPath: view?.configPath || '',
    showTitleBadges: view?.showTitleBadges !== false,
    cleanTitleDisplay: view?.cleanTitleDisplay !== false,
    uiScalePercent: normalizeUIScalePercent(view?.uiScalePercent),
    themeMode: normalizeThemeMode(view?.themeMode),
    smallFileFilterMB: Number(view?.smallFileFilterMB || 0),
    fileListDensity: normalizeFileListDensity(view?.fileListDensity),
    offlineRecentTargets: normalizeOfflineRecentTargets(view?.offlineRecentTargets),
  }
}
