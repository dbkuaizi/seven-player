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

export const themeColorItems = [
  { value: 'blue', title: '蓝色（默认）', light: '#1867c0', dark: '#74a7ff' },
  { value: 'teal', title: '青蓝', light: '#007C89', dark: '#44D7E8' },
  { value: 'sky', title: '天蓝', light: '#0284C7', dark: '#7DD3FC' },
  { value: 'royal', title: '亮蓝', light: '#2563EB', dark: '#7AA2FF' },
  { value: 'cyan', title: '湖蓝', light: '#0891B2', dark: '#67E8F9' },
  { value: 'emerald', title: '翠绿', light: '#059669', dark: '#6EE7B7' },
  { value: 'green', title: '绿色', light: '#16A34A', dark: '#86EFAC' },
  { value: 'lime', title: '黄绿', light: '#65A30D', dark: '#BEF264' },
  { value: 'yellow', title: '黄色', light: '#CA8A04', dark: '#FDE047' },
  { value: 'amber', title: '琥珀', light: '#B7791F', dark: '#FCD34D' },
  { value: 'orange', title: '橙色', light: '#EA580C', dark: '#FDBA74' },
  { value: 'red', title: '红色', light: '#DC2626', dark: '#FCA5A5' },
  { value: 'rose', title: '玫红', light: '#E11D48', dark: '#FDA4AF' },
  { value: 'pink', title: '粉色', light: '#DB2777', dark: '#F9A8D4' },
  { value: 'fuchsia', title: '品红', light: '#C026D3', dark: '#F0ABFC' },
  { value: 'purple', title: '紫色', light: '#9333EA', dark: '#D8B4FE' },
  { value: 'violet', title: '紫罗兰', light: '#7C3AED', dark: '#C4B5FD' },
  { value: 'indigo', title: '靛蓝', light: '#4F46E5', dark: '#A5B4FC' },
  { value: 'gray', title: '灰色', light: '#4B5563', dark: '#D1D5DB' },
  { value: 'slate', title: '石墨', light: '#475569', dark: '#CBD5E1' },
]

export function normalizeThemeColor(value) {
  const normalized = String(value || '').trim().toLowerCase()
  return themeColorItems.some((item) => item.value === normalized)
    ? normalized
    : 'blue'
}

export function resolveThemeColor(value) {
  const normalized = normalizeThemeColor(value)
  return themeColorItems.find((item) => item.value === normalized) || themeColorItems[0]
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
    themeColor: 'blue',
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
    themeColor: normalizeThemeColor(view?.themeColor),
    smallFileFilterMB: Number(view?.smallFileFilterMB || 0),
    fileListDensity: normalizeFileListDensity(view?.fileListDensity),
    offlineRecentTargets: normalizeOfflineRecentTargets(view?.offlineRecentTargets),
  }
}
