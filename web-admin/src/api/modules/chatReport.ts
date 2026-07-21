import { get, put } from '@/utils/request'

export type ChatReportReason =
  | 'spam'
  | 'harassment'
  | 'pornography'
  | 'fraud'
  | 'illegal'
  | 'other'

export type ChatReportStatus = 'pending' | 'resolved' | 'dismissed'

export interface ChatReportUser {
  id: string
  nickname: string
  phone: string
  avatar: string
}

export interface ChatReportEvidence {
  id: string
  senderId: string
  type: 'text' | 'image'
  content: string
  createdAt: string
}

export interface ChatReportItem {
  id: string
  conversationId: string
  reporter: ChatReportUser
  reportedUser: ChatReportUser
  reason: ChatReportReason
  description: string
  evidenceCount: number
  evidence?: ChatReportEvidence[]
  status: ChatReportStatus
  handleResult: string
  handlerAdminId?: string
  handledAt?: string | null
  restrictionUntil?: string | null
  createdAt: string
  updatedAt: string
}

export interface ChatReportListParams {
  page: number
  page_size: number
  status?: ChatReportStatus | ''
  keyword?: string
}

export interface ChatReportListResult {
  list: ChatReportItem[]
  total: number
  page: number
  page_size: number
}

export function getChatReportList(params: ChatReportListParams) {
  return get<ChatReportListResult>('admin/chat-reports', { params })
}

export function getChatReportDetail(id: string) {
  return get<ChatReportItem>(`admin/chat-reports/${id}`)
}

export function resolveChatReport(
  id: string,
  data: {
    status: Exclude<ChatReportStatus, 'pending'>
    result: string
    restriction_until?: string
  },
) {
  return put<ChatReportItem, typeof data>(
    `admin/chat-reports/${id}/resolve`,
    data,
  )
}
