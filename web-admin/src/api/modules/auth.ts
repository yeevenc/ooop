import { post, put } from '@/utils/request'

export interface LoginParams {
  username: string
  password: string
}

export interface LoginUser {
  id: number
  username: string
  status: number
  created_at: string
}

export interface LoginResult {
  token: string
  user: LoginUser
}

export interface ChangePasswordParams {
  old_password: string
  new_password: string
}

// 后台登录只走 admin_users，不复用 APP 用户登录接口。
export function login(params: LoginParams) {
  return post<LoginResult, LoginParams>('admin/auth/login', params, { withToken: false })
}

export function changePassword(params: ChangePasswordParams) {
  return put<null, ChangePasswordParams>('admin/auth/password', params)
}
