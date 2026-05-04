export function normalizeMultilineInput(value) {
  return String(value || '')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function parseTimeInput(value) {
  const text = String(value || '').trim()
  if (!text) {
    return null
  }

  if (/^\d+$/.test(text)) {
    return Number(text) * 1000
  }

  const parts = text.split(':').map((item) => item.trim())
  if (parts.some((item) => !/^\d+$/.test(item))) {
    return null
  }

  if (parts.length === 2) {
    const minutes = Number(parts[0])
    const seconds = Number(parts[1])
    return ((minutes * 60) + seconds) * 1000
  }

  if (parts.length === 3) {
    const hours = Number(parts[0])
    const minutes = Number(parts[1])
    const seconds = Number(parts[2])
    return (((hours * 60 * 60) + (minutes * 60) + seconds) * 1000)
  }

  return null
}

export function basename(path) {
  return String(path || '').split(/[\\/]/).filter(Boolean).pop() || ''
}
