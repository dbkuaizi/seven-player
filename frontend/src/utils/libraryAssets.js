export function buildLibraryImageStyle(url) {
  const normalized = normalizeLibraryAssetUrl(url)
  if (!normalized) {
    return undefined
  }
  return {
    backgroundImage: `url("${escapeLibraryAssetUrl(normalized)}")`,
  }
}

export function escapeLibraryAssetUrl(url) {
  return String(url || '').replace(/"/g, '%22')
}

export function normalizeLibraryAssetUrl(url) {
  const normalized = String(url || '').trim()
  if (!normalized) {
    return ''
  }
  return normalized
}
