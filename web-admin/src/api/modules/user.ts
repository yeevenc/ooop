import { get } from '@/utils/request'

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
