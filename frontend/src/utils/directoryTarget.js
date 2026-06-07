export function rootBreadcrumb() {
  return { id: '0', name: '我的文件' }
}

export function createDirectoryTarget(id, path) {
  const normalizedPath = Array.isArray(path) && path.length
    ? path.map((item) => ({
      id: item.id,
      name: item.name,
    }))
    : [rootBreadcrumb()]

  return {
    id: String(id || '0'),
    name: normalizedPath[normalizedPath.length - 1]?.name || '我的文件',
    path: normalizedPath,
  }
}

export function normalizeDirectoryTargetValue(id, path) {
  const normalizedId = String(id || '').trim()
  const rawPath = Array.isArray(path)
    ? path
      .map((item) => ({
        id: String(item?.id || '').trim(),
        name: String(item?.name || '').trim(),
      }))
      .filter((item) => item.id && item.name)
    : []

  if (normalizedId === '0' || (!normalizedId && !rawPath.length)) {
    return createDirectoryTarget('0', [rootBreadcrumb()])
  }

  if (!rawPath.length) {
    return null
  }

  const normalizedPath = rawPath[0]?.id === '0'
    ? rawPath
    : [rootBreadcrumb(), ...rawPath.filter((item) => item.id !== '0')]

  const finalId = normalizedId || normalizedPath[normalizedPath.length - 1]?.id || '0'
  if (finalId === '0') {
    return createDirectoryTarget('0', [rootBreadcrumb()])
  }

  const matchedIndex = normalizedPath.findIndex((item) => item.id === finalId)
  if (matchedIndex === -1) {
    return null
  }

  return createDirectoryTarget(finalId, normalizedPath.slice(0, matchedIndex + 1))
}

export function normalizeOfflineRecentTargets(targets) {
  return normalizeDirectoryTargets(targets, 3)
}

export function normalizeDirectoryTargets(targets, limit = 3) {
  const normalized = []
  const seen = new Set()
  const max = Math.max(0, Number(limit || 0))
  if (!max) {
    return normalized
  }

  for (const entry of Array.isArray(targets) ? targets : []) {
    const target = normalizeDirectoryTargetValue(entry?.id, entry?.path)
    if (!target || seen.has(target.id)) {
      continue
    }
    seen.add(target.id)
    normalized.push(target)
    if (normalized.length === max) {
      break
    }
  }

  return normalized
}

export function rememberOfflineRecentTargets(targets, target) {
  const currentTarget = normalizeDirectoryTargetValue(target?.id, target?.path)
  if (!currentTarget) {
    return normalizeOfflineRecentTargets(targets)
  }

  return normalizeOfflineRecentTargets([currentTarget, ...(Array.isArray(targets) ? targets : [])])
}

export function formatDirectoryTargetPath(path) {
  return (Array.isArray(path) ? path : []).map((item) => item.name).join(' / ') || '我的文件'
}

export function createOfflineTargetOption(target) {
  return {
    value: `target:${target.id}`,
    title: formatDirectoryTargetPath(target.path),
    icon: 'mdi-folder-outline',
    target,
  }
}
