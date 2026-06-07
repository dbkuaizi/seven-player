export function formatDurationMs(ms) {
  const totalSeconds = Math.max(0, Math.floor(Number(ms || 0) / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${padTwo(hours)}:${padTwo(minutes)}:${padTwo(seconds)}`
  }
  return `${padTwo(minutes)}:${padTwo(seconds)}`
}

export function formatDateTime(value) {
  if (!value) {
    return '--'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '--'
  }

  const year = date.getFullYear()
  const month = padTwo(date.getMonth() + 1)
  const day = padTwo(date.getDate())
  const hours = padTwo(date.getHours())
  const minutes = padTwo(date.getMinutes())
  return `${year}-${month}-${day} ${hours}:${minutes}`
}

export function formatDateOnly(value) {
  if (!value) {
    return '--'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '--'
  }

  const year = date.getFullYear()
  const month = padTwo(date.getMonth() + 1)
  const day = padTwo(date.getDate())
  return `${year}-${month}-${day}`
}

export function formatSize(value) {
  const size = Number(value || 0)
  if (size <= 0) {
    return '--'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let next = size
  while (next >= 1024 && index < units.length - 1) {
    next /= 1024
    index += 1
  }

  return `${next >= 10 || index === 0 ? next.toFixed(0) : next.toFixed(1)} ${units[index]}`
}

export function formatStorageSize(value) {
  const size = Number(value || 0)
  if (size <= 0) {
    return '--'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let index = 0
  let next = size
  while (next >= 1024 && index < units.length - 1) {
    next /= 1024
    index += 1
  }

  const fractionDigits = index >= 3 ? 2 : index >= 1 ? 1 : 0
  return `${next.toFixed(fractionDigits)} ${units[index]}`
}

export function padTwo(value) {
  return String(value).padStart(2, '0')
}
