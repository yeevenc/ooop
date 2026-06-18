import type { VersionUpgradeChannel } from '@/views/setting/version/types'

export interface VersionChannelOption {
  label: string
  value: VersionUpgradeChannel
}

export interface VersionChannelFilterOption {
  label: string
  value: VersionUpgradeChannel | ''
}

// 统一维护版本渠道配置，避免多个页面和弹窗重复定义
export const VERSION_CHANNEL_OPTIONS: VersionChannelOption[] = [
  { label: 'iOS', value: 'ios' },
  { label: '华为', value: 'huawei' },
  { label: '安卓其他', value: 'other' },
  { label: '全渠道', value: 'all' },
]

export const VERSION_CHANNEL_FILTER_OPTIONS: VersionChannelFilterOption[] = [
  { label: '全部渠道', value: '' },
  ...VERSION_CHANNEL_OPTIONS,
]
