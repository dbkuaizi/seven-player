import { normalizeOfflineRecentTargets } from './directoryTarget'

export function normalizeFileListDensity(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'compact' || normalized === 'comfortable') {
    return normalized
  }
  return 'default'
}

export function createDefaultSettings() {
  return {
    preferredPlayer: 'mpv',
    players: [],
    configPath: '',
    showTitleBadges: true,
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
    smallFileFilterMB: Number(view?.smallFileFilterMB || 0),
    fileListDensity: normalizeFileListDensity(view?.fileListDensity),
    offlineRecentTargets: normalizeOfflineRecentTargets(view?.offlineRecentTargets),
  }
}
