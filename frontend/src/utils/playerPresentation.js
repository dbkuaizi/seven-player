export function capabilityTags(player) {
  if (!player) {
    return []
  }

  return [
    {
      label: player.supportsStartPosition ? '支持跳转播放' : '不支持跳转播放',
      color: player.supportsStartPosition ? 'primary' : 'warning',
    },
    {
      label: player.supportsSubtitle ? '支持外挂字幕' : '不支持外挂字幕',
      color: player.supportsSubtitle ? 'primary' : 'warning',
    },
    {
      label: player.supportsManagedResume ? '支持托管续播' : '依赖播放器自身续播',
      color: player.supportsManagedResume ? 'success' : 'info',
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
