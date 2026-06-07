export function capabilityTags(player) {
  if (!player) {
    return []
  }

  return [
    {
      label: player.supportsStartPosition ? '跳转' : '不支持跳转',
      color: player.supportsStartPosition ? 'primary' : 'warning',
    },
    {
      label: player.supportsSubtitle ? '字幕' : '不支持字幕',
      color: player.supportsSubtitle ? 'primary' : 'warning',
    },
  ]
}

export function playerStatusText(player) {
  if (!player) {
    return '--'
  }
  if (player.disabled && player.path) {
    return player.path
  }
  if (player.available) {
    return player.path || '已就绪'
  }
  return '未检测到可执行文件'
}
