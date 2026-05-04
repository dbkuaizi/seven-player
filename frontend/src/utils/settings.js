import { normalizeDirectoryTargets, normalizeOfflineRecentTargets } from './directoryTarget'

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
    scraperDirectories: [],
    scraperSources: defaultScraperSources(),
    scraperLanguage: 'zh-CN',
    scraperAutoScan: false,
    scraperOverwrite: false,
    scraperDownloadImages: true,
    tmdbReadAccessToken: '',
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
    scraperDirectories: normalizeDirectoryTargets(view?.scraperDirectories, 50),
    scraperSources: normalizeScraperSources(view?.scraperSources),
    scraperLanguage: normalizeScraperLanguage(view?.scraperLanguage),
    scraperAutoScan: Boolean(view?.scraperAutoScan),
    scraperOverwrite: Boolean(view?.scraperOverwrite),
    scraperDownloadImages: view?.scraperDownloadImages !== false,
    tmdbReadAccessToken: String(view?.tmdbReadAccessToken || '').trim(),
  }
}

export function defaultScraperSources() {
  return ['tmdb', 'douban', 'bangumi']
}

export function normalizeScraperSources(sources) {
  const allowed = new Set(defaultScraperSources())
  const normalized = []
  const seen = new Set()

  for (const source of Array.isArray(sources) ? sources : []) {
    const value = String(source || '').trim().toLowerCase()
    if (!allowed.has(value) || seen.has(value)) {
      continue
    }
    seen.add(value)
    normalized.push(value)
  }

  return normalized.length ? normalized : defaultScraperSources()
}

export function normalizeScraperLanguage(value) {
  const normalized = String(value || '').trim()
  if (['zh-CN', 'zh-TW', 'en-US', 'ja-JP', 'ko-KR'].includes(normalized)) {
    return normalized
  }
  return 'zh-CN'
}
