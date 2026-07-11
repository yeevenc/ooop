import { get, put } from '@/utils/request'

export interface UserListParams {
  page: number
  page_size: number
  keyword?: string
  status?: number
}

export interface UserItem {
  id: number
  phone: string
  username: string
  nickname: string
  avatar: string
  gender: string
  region: string
  bio: string
  platform: string
  device_no: string
  push_platform: string
  registration_id: string
  /** 是否已实名认证 */
  is_real_name_verified: boolean
  /** 1 正常 / 0 封禁（APP 用户） */
  status: number
  /** 限时解封时间；永久封禁为 null */
  banned_until: string | null
  /** 封禁原因备注 */
  ban_reason: string
  register_source: string
  last_login_at: string | null
  created_at: string
}

export interface UserListResult {
  list: UserItem[]
  total: number
  page: number
  page_size: number
}

// 后台用户列表走 admin 权限接口，查询的是 APP 用户表。
export function getUserList(params: UserListParams) {
  return get<UserListResult>('admin/users', {
    params,
  })
}

// 获取指定 APP 用户详情，编辑前调用。
export function getUserDetail(id: number) {
  return get<UserItem>(`admin/users/${id}`)
}

// 后台可修改的用户资料字段，与 APP 端「修改资料」保持一致。
export interface UpdateUserPayload {
  nickname?: string
  gender?: string
  avatar?: string
  bio?: string
}

// 修改指定 APP 用户的资料，返回更新后的完整用户信息。
export function updateUser(id: number, data: UpdateUserPayload) {
  return put<UserItem>(`admin/users/${id}`, data)
}

/** 封禁类型：永久 / 限时（仅作用于 APP 用户，非后台管理员） */
export type BanType = 'permanent' | 'temporary'

export interface BanUserPayload {
  type: BanType
  /**
   * 限时封禁小时数（temporary 必填）。
   * 备注：列表页用时间区间选择器，前端换算为小时后再提交。
   */
  duration_hours?: number
  /** 封禁原因备注，可选 */
  reason?: string
}

/** 封禁 APP 用户 PUT /admin/users/:id/ban */
export function banUser(id: number, data: BanUserPayload) {
  return put<UserItem>(`admin/users/${id}/ban`, data)
}

/** 解封 APP 用户 PUT /admin/users/:id/unban */
export function unbanUser(id: number) {
  return put<UserItem>(`admin/users/${id}/unban`)
}
