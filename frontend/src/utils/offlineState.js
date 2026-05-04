import { formatSize } from './format'

export function createEmptyOfflineState() {
  return {
    quota: 0,
    total: 0,
    activeCount: 0,
    failedCount: 0,
    completedCount: 0,
    tasks: [],
  }
}

export function normalizeOfflineState(data) {
  if (!data) {
    return createEmptyOfflineState()
  }

  return {
    quota: Number(data.quota || 0),
    total: Number(data.total || 0),
    activeCount: Number(data.activeCount || 0),
    failedCount: Number(data.failedCount || 0),
    completedCount: Number(data.completedCount || 0),
    tasks: Array.isArray(data.tasks) ? data.tasks.map((task) => ({
      ...task,
      percent: Number(task.percent || 0),
      percentText: task.percentText || '0%',
      speedText: task.speedText || '--',
      leftTimeText: task.leftTimeText || '--',
    })) : [],
  }
}

export function offlineTaskMetaText(task) {
  if (!task) {
    return '--'
  }

  return [formatSize(task.size), task.addTime]
    .map((item) => String(item || '').trim())
    .filter((item) => item && item !== '--')
    .join(' · ') || '--'
}
