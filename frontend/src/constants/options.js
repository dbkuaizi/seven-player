export const navigationItems = [
  { value: 'files', label: '文件管理', icon: 'mdi-folder-outline' },
  { value: 'library', label: '影视库', icon: 'mdi-view-dashboard-outline' },
  { value: 'downloads', label: '云下载', icon: 'mdi-download-circle-outline' },
  { value: 'settings', label: '系统设置', icon: 'mdi-cog-outline' },
]

export const typeOptions = [
  { label: '全部', value: 'all' },
  { label: '目录', value: 'dir' },
  { label: '视频', value: 'video' },
  { label: '其他文件', value: 'file' },
]

export const sortOptions = [
  { label: '目录优先', value: 'folders' },
  { label: '名称 A-Z', value: 'name' },
  { label: '最近更新', value: 'updated' },
  { label: '体积最大', value: 'size' },
  { label: '续播优先', value: 'resume' },
]

export const smallFileFilterOptions = [
  { label: '关闭', value: 0 },
  { label: '1 MB', value: 1 },
  { label: '2 MB', value: 2 },
  { label: '3 MB', value: 3 },
  { label: '5 MB', value: 5 },
  { label: '10 MB', value: 10 },
]

export const fileListDensityOptions = [
  { label: '紧凑', value: 'compact' },
  { label: '默认', value: 'default' },
  { label: '宽松', value: 'comfortable' },
]

export const downloadFilterOptions = [
  { value: 'active', label: '正在下载' },
  { value: 'failed', label: '下载失败' },
  { value: 'completed', label: '完成记录' },
]

export const pageSizeOptions = [
  { title: '20 / 页', value: 20 },
  { title: '50 / 页', value: 50 },
  { title: '100 / 页', value: 100 },
]

export const OFFLINE_TARGET_PICKER_VALUE = '__pick_other__'
