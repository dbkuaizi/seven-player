export function normalizeMultilineInput(value) {
  return String(value || '')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function basename(path) {
  return String(path || '').split(/[\\/]/).filter(Boolean).pop() || ''
}
