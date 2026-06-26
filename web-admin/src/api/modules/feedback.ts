import { get } from '@/utils/request'

export type FeedbackType = 'product' | 'account' | 'activity'

export interface FeedbackItem {
  id: string
  userId: string
  userPhone: string
  userNickname: string
  type: FeedbackType
  content: string
  imageUrls: string[]
  devicePlatform: string
  deviceVersion: string
  appVersion: string
  createdAt: string
}

export interface FeedbackListParams {
  page: number
  page_size: number
  type?: FeedbackType | ''
  keyword?: string
}

export interface FeedbackListResult {
  list: FeedbackItem[]
  total: number
  page: number
  page_size: number
}

export function getFeedbackList(params: FeedbackListParams) {
  return get<FeedbackListResult>('admin/feedbacks', { params })
}
