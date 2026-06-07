import { computed } from 'vue'
import { formatDateOnly, formatStorageSize } from '../utils/format'
import { normalizeAvatarUrl } from '../utils/filePresentation'

export function useAccountSummary({ loggedIn, user, proxyBase }) {
  const accountDisplayName = computed(() => {
    if (!loggedIn.value || !user.value) {
      return '未登录'
    }
    return user.value.userName || '115 用户'
  })

  const accountAvatarUrl = computed(() => buildAccountAvatarUrl(user.value?.faceUrl, proxyBase.value))

  const accountVipLevelText = computed(() => {
    if (!loggedIn.value || !user.value) {
      return '--'
    }
    if (user.value.vipLabel) {
      return user.value.vipLabel
    }
    return user.value.isVip ? 'VIP' : '普通用户'
  })

  const accountVipExpireText = computed(() => {
    if (!loggedIn.value || !user.value) {
      return '--'
    }
    if (!user.value.isVip) {
      return '非 VIP'
    }
    if (user.value.vipForever) {
      return '永久'
    }
    if (user.value.vipExpireAt) {
      return formatDateOnly(user.value.vipExpireAt)
    }
    return '--'
  })

  const accountSpaceUsageText = computed(() => {
    const total = Number(user.value?.spaceTotal || 0)
    const used = Number(user.value?.spaceUsed || 0)
    if (total > 0 && used >= 0) {
      return `${used > 0 ? formatStorageSize(used) : '0 B'} / ${formatStorageSize(total)}`
    }
    if (used > 0) {
      return formatStorageSize(used)
    }
    return '--'
  })

  const accountSpacePercent = computed(() => {
    const total = Number(user.value?.spaceTotal || 0)
    const used = Math.max(0, Number(user.value?.spaceUsed || 0))
    if (!(total > 0)) {
      return 0
    }
    return Math.min(100, Math.max(0, (used / total) * 100))
  })

  const accountSpacePercentText = computed(() => {
    if (!(Number(user.value?.spaceTotal || 0) > 0)) {
      return '--'
    }
    return `${accountSpacePercent.value.toFixed(accountSpacePercent.value >= 10 ? 1 : 2)}%`
  })

  return {
    accountDisplayName,
    accountAvatarUrl,
    accountVipLevelText,
    accountVipExpireText,
    accountSpaceUsageText,
    accountSpacePercent,
    accountSpacePercentText,
  }
}

function buildAccountAvatarUrl(value, proxyBase) {
  const normalized = normalizeAvatarUrl(value)
  if (!normalized) {
    return ''
  }
  if (!proxyBase || normalized.startsWith('data:')) {
    return normalized
  }
  return `${proxyBase}/avatar?url=${encodeURIComponent(normalized)}`
}
