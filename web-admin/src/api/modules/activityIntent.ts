import { get } from '@/utils/request'

export interface ActivityIntent {
  id: string
  userId: string
  city: string
  latitude: number
  longitude: number
  locationText: string
  categoryIds: number[]
  timeSlots: string[]
  note: string
  notifyEnabled: boolean
  updatedAt: string
}

export const getActivityIntents = (params: { city?: string; page: number }) =>
  get<ActivityIntent[]>('admin/activity-intents', { params })
