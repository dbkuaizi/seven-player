import { rootBreadcrumb } from './directoryTarget'

export function normalizeBreadcrumbPath(path) {
  const normalized = Array.isArray(path)
    ? path
      .map((item) => ({
        id: String(item?.id || ''),
        name: String(item?.name || '').trim(),
      }))
      .filter((item) => item.id && item.name)
    : []

  if (!normalized.length || normalized[0].id !== '0') {
    return [rootBreadcrumb(), ...normalized.filter((item) => item.id !== '0')]
  }

  return normalized
}

export function resolveDirectoryPath(data, dirId, options = {}, knownDirectoryPaths = {}) {
  const normalizedDirId = String(data?.dirId || dirId || '0')
  const responsePath = normalizeBreadcrumbPath(data?.path)
  const fallbackPath = normalizeBreadcrumbPath(options?.fallbackPath)
  const cachedPath = normalizeBreadcrumbPath(knownDirectoryPaths?.[normalizedDirId])
  const displayName = String(data?.name || options?.fallbackName || '').trim()

  if (normalizedDirId === '0') {
    return [rootBreadcrumb()]
  }

  if (responsePath.length > 1) {
    return responsePath
  }

  if (fallbackPath.length > 1) {
    return fallbackPath
  }

  if (cachedPath.length > 1) {
    return cachedPath
  }

  const parentPath = normalizeBreadcrumbPath(knownDirectoryPaths?.[String(data?.parentId || '')])
  if (parentPath.length > 0 && displayName) {
    return [...parentPath, { id: normalizedDirId, name: displayName }]
  }

  if (displayName) {
    return [rootBreadcrumb(), { id: normalizedDirId, name: displayName }]
  }

  return [rootBreadcrumb()]
}

export function breadcrumbPathUntil(path, dirId) {
  const normalizedPath = Array.isArray(path) ? path : []
  const index = normalizedPath.findIndex((item) => item.id === dirId)
  if (index === -1) {
    return [rootBreadcrumb()]
  }
  return normalizedPath.slice(0, index + 1)
}
