import { get, post } from '@/utils/request'

export type SystemBroadcastStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface SystemBroadcastItem {
  id: string
  title: string
  content: string
  status: SystemBroadcastStatus
  targetCount: number
  messageCount: number
  pushSuccess: number
  pushSkipped: number
  pushFailed: number
  errorMessage?: string
  startedAt?: string | null
  finishedAt?: string | null
  createdAt: string
}

export interface SystemBroadcastListParams {
  page: number
  page_size: number
  status?: SystemBroadcastStatus | ''
}

export interface SystemBroadcastListResult {
  list: SystemBroadcastItem[]
  total: number
  page: number
  page_size: number
}

export interface CreateSystemBroadcastPayload {
  title: string
  content: string
}

export function getSystemBroadcastList(params: SystemBroadcastListParams) {
  return get<SystemBroadcastListResult>('admin/system-broadcasts', { params })
}

export function createSystemBroadcast(data: CreateSystemBroadcastPayload) {
  return post<SystemBroadcastItem>('admin/system-broadcasts', data)
}
