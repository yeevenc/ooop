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
  status: number
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
