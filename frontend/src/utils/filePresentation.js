import { formatDateTime, formatDurationMs, formatSize } from './format'

const videoExtensions = new Set(['.mp4', '.mkv', '.avi', '.mov', '.wmv', '.flv', '.m4v', '.rmvb', '.ts', '.webm'])
const audioExtensions = new Set(['.mp3', '.flac', '.m4a', '.aac', '.wav', '.ogg', '.opus', '.ape', '.dts'])
const subtitleExtensions = new Set(['.srt', '.ass', '.ssa', '.vtt', '.sub'])
const archiveExtensions = new Set(['.zip', '.rar', '.7z', '.tar', '.gz', '.iso'])

const fileBadgeDefinitions = [
  {
    pattern: /\b(2160P|1080P|720P|4K)\b/gi,
    normalize: (match) => match.toUpperCase(),
    describe: (label) => `${label} 分辨率`,
  },
  {
    pattern: /\b(WEB[- .]?DL|WEB[- .]?RIP|BLURAY|REMUX|BDRIP|HDRIP|HDTV|DVDRIP)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/[ .]/g, ''),
    describe: (label) => `${label} 片源类型`,
  },
  {
    pattern: /\b(HEVC|H\.?265|X265|AV1|X264|H\.?264|AVC)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/\./g, ''),
    describe: (label) => `${label} 视频编码`,
  },
  {
    pattern: /\b(AAC|ACC|AC3|EAC3|DTS(?:-HD)?|TRUEHD|FLAC|ATMOS)\b/gi,
    normalize: (match) => normalizeAudioBadge(match),
    describe: (label) => audioBadgeDescription(label),
  },
  {
    pattern: /\b(HDR10\+?|HDR|DV|DOLBY[ .-]?VISION)\b/gi,
    normalize: (match) => match.toUpperCase().replace(/[ .]/g, ''),
    describe: (label) => `${label} 高动态范围信息`,
  },
  {
    pattern: /\b(CHS|CHT|ENG|JPN|KOR|MANDARIN|CANTONESE)\b/gi,
    normalize: (match) => match.toUpperCase(),
    describe: (label) => subtitleBadgeDescription(label),
  },
  {
    pattern: /简繁英字幕|简繁字幕|中英字幕|中日字幕|中韩字幕|国粤双语|粤国双语|双音轨|多音轨|简体|繁体|中字|双语|内封|外挂|国语|粤语|英语|日语|韩语|普通话/g,
    normalize: (match) => match,
    describe: (label) => textBadgeDescription(label),
  },
  {
    pattern: /杜比视界|杜比全景声/g,
    normalize: (match) => match,
    describe: (label) => textBadgeDescription(label),
  },
]

export function normalizeFileItem(item, options = {}) {
  const resumeMs = Number(item?.resumeMs || 0)
  const durationSec = Number(item?.durationSec || 0)
  const originalName = String(item?.originalName || item?.name || '')
  const presentation = buildFilePresentation(originalName, Boolean(item?.isDirectory), options)
  const resumeBadge = buildResumeBadge(resumeMs, durationSec)
  const durationBadge = buildDurationBadge(durationSec, resumeMs)
  const badgeList = [
    ...(resumeBadge ? [resumeBadge] : []),
    ...(durationBadge ? [durationBadge] : []),
    ...(options.showTitleBadges !== false ? presentation.badges : []),
  ]
  const visibleBadges = badgeList.slice(0, 4)
  const hiddenBadges = badgeList.slice(4)

  return {
    ...item,
    originalName,
    rowKey: item?.fileId || item?.pickCode || item?.name,
    icon: presentation.icon,
    kindLabel: presentation.kindLabel,
    mediaKind: presentation.mediaKind,
    displayName: presentation.title,
    displayNameMain: presentation.titleMain,
    displayNameExtension: presentation.titleExtension,
    badges: badgeList,
    metaBadges: presentation.badges,
    visibleBadges,
    hiddenBadgeCount: hiddenBadges.length,
    hiddenBadgeSummary: hiddenBadges.map((badge) => `${badge.label}：${badge.description}`).join(' · '),
    iconColor: colorForMediaKind(presentation.mediaKind),
    sizeText: item?.isDirectory ? '--' : formatSize(item?.size),
    updatedText: formatDateTime(item?.updatedAt),
    durationSec,
    durationText: durationSec > 0 ? formatDurationMs(durationSec * 1000) : '',
    resumeMs,
    resumeText: resumeMs > 0 ? formatResumeProgressText(resumeMs, durationSec) : '',
  }
}

export function cleanDisplayTitle(name, options = {}) {
  return buildDisplayTitle(String(name || ''), '', true, options) || String(name || '').trim()
}

export function formatResumeProgressText(resumeMs, durationSec = 0) {
  if (!resumeMs) {
    return ''
  }

  const current = formatDurationMs(resumeMs)
  const total = Number(durationSec || 0) > 0 ? formatDurationMs(Number(durationSec) * 1000) : ''
  return total ? `${current}/${total}` : current
}

export function compareItems(left, right, mode) {
  const leftName = left.displayName || left.name
  const rightName = right.displayName || right.name

  if (mode === 'folders') {
    if (left.isDirectory !== right.isDirectory) {
      return left.isDirectory ? -1 : 1
    }
    return compareText(leftName, rightName)
  }

  if (mode === 'name') {
    return compareText(leftName, rightName)
  }

  if (mode === 'updated') {
    return compareTimestamp(right.updatedAt, left.updatedAt) || compareText(leftName, rightName)
  }

  if (mode === 'size') {
    if ((right.size || 0) !== (left.size || 0)) {
      return (right.size || 0) - (left.size || 0)
    }
    return compareText(leftName, rightName)
  }

  if (mode === 'resume') {
    if ((right.resumeMs || 0) !== (left.resumeMs || 0)) {
      return (right.resumeMs || 0) - (left.resumeMs || 0)
    }
    if (left.isDirectory !== right.isDirectory) {
      return left.isDirectory ? -1 : 1
    }
    return compareText(leftName, rightName)
  }

  return compareText(leftName, rightName)
}

export function normalizeAvatarUrl(value) {
  const normalized = String(value || '').trim()
  if (!normalized) {
    return ''
  }
  if (normalized.startsWith('data:')) {
    return normalized
  }
  if (normalized.startsWith('//')) {
    return `https:${normalized}`
  }
  if (/^https?:\/\//i.test(normalized)) {
    return normalized
  }
  if (normalized.startsWith('/')) {
    return `https://115.com${normalized}`
  }
  return `https://${normalized.replace(/^\/+/, '')}`
}

function buildResumeBadge(resumeMs, durationSec = 0) {
  const progressText = formatResumeProgressText(resumeMs, durationSec)
  if (!progressText) {
    return null
  }

  return {
    label: `上次播放：${progressText}`,
    description: `上次退出位置 ${progressText}`,
    color: 'success',
  }
}

function buildDurationBadge(durationSec = 0, resumeMs = 0) {
  const normalizedDuration = Number(durationSec || 0)
  if (normalizedDuration <= 0 || Number(resumeMs || 0) > 0) {
    return null
  }

  const durationText = formatDurationMs(normalizedDuration * 1000)
  if (!durationText) {
    return null
  }

  return {
    label: `时长：${durationText}`,
    description: `视频总时长 ${durationText}`,
    color: 'info',
  }
}

function buildFilePresentation(name, isDirectory, options = {}) {
  const resolvedName = String(name || '').trim()
  const extension = extractExtension(resolvedName)
  const badgeSource = isDirectory ? resolvedName : stripExtension(resolvedName, extension)
  const badges = extractFileBadges(badgeSource)
  const mediaKind = detectMediaKind(isDirectory, extension)
  const displayTitle = buildDisplayTitle(resolvedName, extension, isDirectory, options)
    || (resolvedName || (isDirectory ? '未命名文件夹' : '未命名文件'))
  const displayParts = splitDisplayTitle(displayTitle, extension, isDirectory)

  return {
    title: displayTitle,
    titleMain: displayParts.main,
    titleExtension: displayParts.extension,
    badges,
    mediaKind,
    icon: iconForMediaKind(mediaKind),
    kindLabel: labelForMediaKind(mediaKind),
  }
}

function buildDisplayTitle(name, extension, isDirectory, options = {}) {
  const resolved = String(name || '').trim()
  if (!resolved) {
    return ''
  }

  if (options.cleanTitleDisplay === false) {
    return resolved
  }

  if (isDirectory) {
    return cleanupDisplayText(resolved)
  }

  const base = stripExtension(resolved, extension)
  const cleanedBase = cleanupDisplayText(base)
  return `${cleanedBase || base || resolved}${extension || ''}`
}

function splitDisplayTitle(title, extension, isDirectory) {
  const resolved = String(title || '').trim()
  if (!resolved) {
    return { main: '', extension: '' }
  }

  if (isDirectory) {
    return { main: resolved, extension: '' }
  }

  const actualExtension = extractExtension(resolved)
  if (!actualExtension) {
    return { main: resolved, extension: '' }
  }

  return {
    main: stripExtension(resolved, actualExtension),
    extension: actualExtension,
  }
}

function cleanupDisplayText(value) {
  let result = String(value || '').trim()
  if (!result) {
    return ''
  }

  result = stripLeadingBracketSegments(result)
  result = stripLeadingSiteNoise(result)
  result = stripDecorativeBracketSegments(result)
  result = stripBracketedAds(result)
  result = stripInlineSiteNoise(result)
  result = stripTechnicalSegments(result)
  result = preferPrimaryLocalizedTitle(result)
  result = collapseTitleDelimiters(result)

  return result || String(value || '').trim()
}

function stripLeadingBracketSegments(value) {
  let result = String(value || '').trim()
  const noisySegment = /^\s*[\[\(【（][^\]\)】）]*(发布|压制|影视|剧集|联盟|论坛|资源|高清|BT|BTHD|BDHD|BTBTT|HDTV|WEB|MP4|MKV|1080|2160|720|www\.|com|cn)[^\]\)】）]*[\]\)】）]\s*/i

  while (noisySegment.test(result)) {
    result = result.replace(noisySegment, '').trim()
  }

  return result
}

function stripLeadingSiteNoise(value) {
  return String(value || '')
    .replace(/^\s*(?:www\.[^\s]+|[A-Za-z0-9-]+\.(?:com|cn|net|org|tv|cc))\s*/i, '')
    .trim()
}

function stripBracketedAds(value) {
  return String(value || '')
    .replace(/[\[\(【（][^\]\)】）]*(?:www\.|https?:\/\/|magnet:|@|\.com|\.cn|\.net|论坛|发布|压制|资源组|影视|剧集)[^\]\)】）]*[\]\)】）]/gi, ' ')
    .trim()
}

function stripDecorativeBracketSegments(value) {
  return String(value || '')
    .replace(/[\[\(【（][^\]\)】）]*(?:全\s*\d+\s*[集季部期]|字幕|配音|双语|国语|粤语|英语|日语|韩语|普通话|内封|外挂|HDR|WEB|2160|1080|720|4K)[^\]\)】）]*[\]\)】）]/gi, ' ')
    .trim()
}

function stripInlineSiteNoise(value) {
  return String(value || '')
    .replace(/\b(?:www\.)?[A-Za-z0-9-]+\.(?:com|cn|net|org|tv|cc)\b/gi, ' ')
    .replace(/\b(?:发布组|压制组|资源组|影视联盟|高清剧集网|高清剧集|高清影视|热播资源|首发)\b/gi, ' ')
    .trim()
}

function stripTechnicalSegments(value) {
  let result = String(value || '')

  for (const definition of fileBadgeDefinitions) {
    result = result.replace(definition.pattern, ' ')
  }

  result = result
    .replace(/\b(?:NF|AMZN|ATVP|DSNP|HQ|UHD|HD|SD|DDP(?:\d(?:\.\d)?)?|DD(?:\d(?:\.\d)?)?|XIAOMI|HOTWEB|ZEROTV|OURTV|MOMOWEB|CMCTV|DREAMHD|HDSKY|HDAREA|CHD|MNHD|FLYTV)\b/gi, ' ')
    .replace(/(?:^|[\s._-])S\d{1,2}(?:E\d{1,3})?(?=$|[\s._-])/gi, (match) => match)
    .replace(/(?:^|[\s._-])(?:\d{4})(?=$|[\s._-])/g, (match) => match)
    .replace(/[-._ ]+[A-Za-z][A-Za-z0-9]{2,}(?=$)/g, ' ')

  return result.trim()
}

function preferPrimaryLocalizedTitle(value) {
  const result = String(value || '').trim()
  if (!/[\u3400-\u9fff]/.test(result)) {
    return result
  }

  const englishChunk = result.match(/[A-Za-z][A-Za-z0-9]+(?:\.[A-Za-z0-9]+){1,}.*/)
  if (!englishChunk?.index) {
    return result
  }

  const localized = result.slice(0, englishChunk.index).trim()
  const episodeToken = extractEpisodeToken(englishChunk[0])
  return appendEpisodeToken(localized, episodeToken) || result
}

function collapseTitleDelimiters(value) {
  return String(value || '')
    .replace(/^[\s._-]+/, '')
    .replace(/[\s._-]+$/, '')
    .replace(/[._]+/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .replace(/[·•]+/g, ' ')
    .replace(/\s*-\s*/g, '-')
    .trim()
}

function extractEpisodeToken(value) {
  const text = String(value || '')
  if (!text) {
    return ''
  }

  const separatedSeasonEpisode = text.match(/(?:^|[\s._-])(S\d{1,2})[\s._-]+(E\d{1,3})(?=$|[\s._-])/i)
  if (separatedSeasonEpisode) {
    return `${separatedSeasonEpisode[1]}${separatedSeasonEpisode[2]}`.toUpperCase()
  }

  const seasonEpisode = text.match(/(?:^|[\s._-])(S\d{1,2}E\d{1,3})(?=$|[\s._-])/i)
  if (seasonEpisode) {
    return seasonEpisode[1].toUpperCase()
  }

  const episode = text.match(/(?:^|[\s._-])((?:EP|E)\d{1,3})(?=$|[\s._-])/i)
  if (episode) {
    return episode[1].toUpperCase()
  }

  const localizedEpisode = text.match(/第\s*\d{1,4}\s*[集话話期]/)
  if (localizedEpisode) {
    return localizedEpisode[0].replace(/\s+/g, '')
  }

  const tokens = text.split(/[\s._-]+/).filter(Boolean)
  for (let index = tokens.length - 1; index >= 0; index -= 1) {
    const token = tokens[index]
    if (!/^\d{1,3}$/.test(token)) {
      continue
    }

    const number = Number(token)
    if (!Number.isInteger(number) || number <= 0 || number > 300) {
      continue
    }
    return token
  }

  return ''
}

function appendEpisodeToken(title, token) {
  const resolvedTitle = String(title || '').trim()
  const resolvedToken = String(token || '').trim()
  if (!resolvedTitle || !resolvedToken) {
    return resolvedTitle
  }

  const tokenPattern = new RegExp(`(?:^|[\\s._-])${escapeRegExp(resolvedToken)}(?=$|[\\s._-])`, 'i')
  if (tokenPattern.test(resolvedTitle)) {
    return resolvedTitle
  }

  return `${resolvedTitle} ${resolvedToken}`
}

function escapeRegExp(value) {
  return String(value || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function extractFileBadges(name) {
  const badges = []
  const seen = new Set()

  for (const definition of fileBadgeDefinitions) {
    for (const match of String(name || '').matchAll(definition.pattern)) {
      const rawLabel = match?.[0]
      const label = String(definition.normalize ? definition.normalize(rawLabel) : rawLabel || '').trim()
      if (!label || seen.has(label)) {
        continue
      }

      seen.add(label)
      badges.push({
        label,
        description: definition.describe ? definition.describe(label) : label,
      })
    }
  }

  return badges
}

function subtitleBadgeDescription(label) {
  const normalized = String(label || '').toUpperCase()
  if (normalized === 'CHS' || label === '简体') return '简体中文字幕'
  if (normalized === 'CHT' || label === '繁体') return '繁体中文字幕'
  if (normalized === 'ENG') return '英文字幕或英文音轨'
  if (normalized === 'JPN') return '日文字幕或日文音轨'
  if (normalized === 'KOR') return '韩文字幕或韩文音轨'
  if (normalized === 'MANDARIN' || label === '国语' || label === '普通话') return '普通话音轨'
  if (normalized === 'CANTONESE' || label === '粤语') return '粤语音轨'
  if (label === '英语') return '英语音轨'
  if (label === '日语') return '日语音轨'
  if (label === '韩语') return '韩语音轨'
  if (label === '中字') return '带中文字幕'
  if (label === '中英字幕') return '中英双语字幕'
  if (label === '中日字幕') return '中日双语字幕'
  if (label === '中韩字幕') return '中韩双语字幕'
  if (label === '简繁字幕') return '简繁中文字幕'
  if (label === '简繁英字幕') return '简繁英多语字幕'
  if (label === '双语') return '双语字幕或双语音轨'
  if (label === '内封') return '字幕内封在媒体文件中'
  if (label === '外挂') return '需要外挂字幕文件'
  return `${label} 字幕或语言信息`
}

function normalizeAudioBadge(label) {
  const normalized = String(label || '').toUpperCase().replace(/\s+/g, '')
  if (normalized === 'ACC') return 'AAC'
  return normalized
}

function audioBadgeDescription(label) {
  if (label === 'ATMOS') return '杜比全景声音频'
  if (label === 'TRUEHD') return 'TrueHD 无损音频'
  if (label === 'DTS-HD') return 'DTS-HD 音频'
  if (label === 'FLAC') return 'FLAC 无损音频'
  return `${label} 音频格式`
}

function textBadgeDescription(label) {
  if (label === '双音轨') return '双音轨版本'
  if (label === '多音轨') return '多音轨版本'
  if (label === '国粤双语' || label === '粤国双语') return '普通话与粤语双音轨'
  if (label === '杜比视界') return '杜比视界版本'
  if (label === '杜比全景声') return '杜比全景声音频'
  return subtitleBadgeDescription(label)
}

function detectMediaKind(isDirectory, extension) {
  if (isDirectory) return 'folder'
  if (videoExtensions.has(extension)) return 'video'
  if (audioExtensions.has(extension)) return 'audio'
  if (subtitleExtensions.has(extension)) return 'subtitle'
  if (archiveExtensions.has(extension)) return 'archive'
  return 'file'
}

function iconForMediaKind(kind) {
  if (kind === 'folder') return 'mdi-folder-outline'
  if (kind === 'video') return 'mdi-movie-open-outline'
  if (kind === 'audio') return 'mdi-music-circle-outline'
  if (kind === 'subtitle') return 'mdi-subtitles-outline'
  if (kind === 'archive') return 'mdi-package-variant-closed'
  return 'mdi-file-outline'
}

function labelForMediaKind(kind) {
  if (kind === 'folder') return '文件夹'
  if (kind === 'video') return '视频'
  if (kind === 'audio') return '音频'
  if (kind === 'subtitle') return '字幕'
  if (kind === 'archive') return '压缩包'
  return '文件'
}

function colorForMediaKind(kind) {
  if (kind === 'folder') return 'warning'
  if (kind === 'video') return 'primary'
  if (kind === 'audio') return 'deep-purple'
  if (kind === 'subtitle') return 'teal'
  if (kind === 'archive') return 'brown'
  return 'grey'
}

function extractExtension(name) {
  const value = String(name || '')
  const index = value.lastIndexOf('.')
  if (index <= 0) {
    return ''
  }
  return value.slice(index).toLowerCase()
}

function stripExtension(name, extension) {
  if (!extension) {
    return String(name || '')
  }
  return String(name || '').slice(0, -extension.length)
}

function compareText(left, right) {
  return String(left || '').localeCompare(String(right || ''), 'zh-Hans-CN', { sensitivity: 'base' })
}

function compareTimestamp(left, right) {
  return Date.parse(left || '') - Date.parse(right || '')
}
