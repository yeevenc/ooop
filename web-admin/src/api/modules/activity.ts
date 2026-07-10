import { get, post, put, del } from '@/utils/request'

export interface ActivityOrganizer {
  id: string
  name: string
}

// 后台活动项（取自后端 PublicActivity 的常用字段）。
export interface AdminActivity {
  id: string
  title: string
  categoryId: string
  categoryLabel: string
  imageUrl: string
  status: string
  costLabel: string
  costType: string
  timeRange: string
  activityTime: string
  dateText: string
  currentCount: number
  totalCount: number
  locationText: string
  city: string
  feeDetail: string
  genderRequirement: string
  intro: string
  notice: string
  organizer: ActivityOrganizer
  createdAt: string
}

export interface ActivityListParams {
  page: number
  page_size: number
  keyword?: string
  status?: string
  category_id?: string
}

export interface ActivityListResult {
  list: AdminActivity[]
  total: number
  page: number
  page_size: number
}

export interface PushChannelResult {
  channel: string
  triggered: boolean
  success: boolean
  message: string
  response?: string
}

export interface PushNotificationResult {
  triggered: boolean
  success: boolean
  alias: string
  message: string
  channels?: PushChannelResult[]
}

export interface AdminActivityReviewResult {
  activity: AdminActivity
  notification: PushNotificationResult
}

// 后台可改的活动文本字段（日期/坐标/图片/发起人/状态不在此处改）。
export interface UpdateActivityPayload {
  title: string
  category_id: string
  activity_time?: string
  location_text: string
  city: string
  total_count: number
  cost_type?: string
  fee_detail?: string
  gender_requirement?: string
  intro: string
  notice?: string
}

export function getActivityList(params: ActivityListParams) {
  return get<ActivityListResult>('admin/activities', { params })
}

export function getActivityDetail(id: string | number) {
  return get<AdminActivity>(`admin/activities/${id}`)
}

export function updateActivity(id: string | number, data: UpdateActivityPayload) {
  return put<AdminActivity>(`admin/activities/${id}`, data)
}

export function deleteActivity(id: string | number) {
  return del(`admin/activities/${id}`)
}

// 审核：通过 / 拒绝（仅对待审核活动生效）。
export function reviewActivity(id: string | number, action: 'approve' | 'reject') {
  return put<AdminActivityReviewResult>(`admin/activities/${id}/review`, { action })
}

// 上下架：taken_down 下架 / ongoing 恢复。
export function setActivityStatus(id: string | number, status: 'taken_down' | 'ongoing') {
  return put<AdminActivity>(`admin/activities/${id}/status`, { status })
}

// ===== 活动分类 =====

export interface AdminCategory {
  id: string
  label: string
  icon: string
  sort: number
  status: number
}

export interface CategoryPayload {
  id?: string
  label: string
  icon: string
  sort: number
  status: number
}

export function getCategoryList() {
  return get<AdminCategory[]>('admin/activity-categories')
}

export function createCategory(data: CategoryPayload) {
  return post<AdminCategory>('admin/activity-categories', data)
}

export function updateCategory(id: string, data: CategoryPayload) {
  return put<AdminCategory>(`admin/activity-categories/${id}`, data)
}

export function deleteCategory(id: string) {
  return del(`admin/activity-categories/${id}`)
}
