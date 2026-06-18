export const OOOP_USER_STORE_STORAGE_KEY = 'ooop-admin:oooPUser'

const TOKEN_KEY = 'token'

interface PersistedUserState {
  token?: string
}

function getPersistedUserState(): PersistedUserState {
  const rawValue = localStorage.getItem(OOOP_USER_STORE_STORAGE_KEY)

  if (!rawValue) {
    return {}
  }

  try {
    return JSON.parse(rawValue) as PersistedUserState
  } catch {
    return {}
  }
}

function setPersistedUserState(state: PersistedUserState) {
  localStorage.setItem(OOOP_USER_STORE_STORAGE_KEY, JSON.stringify(state))
}

// 统一封装 token 读写，避免在路由和请求层重复处理本地缓存
export function getToken() {
  return getPersistedUserState()[TOKEN_KEY] ?? ''
}

export function setToken(token: string) {
  const state = getPersistedUserState()
  state[TOKEN_KEY] = token
  setPersistedUserState(state)
}

export function removeToken() {
  const state = getPersistedUserState()
  state[TOKEN_KEY] = ''
  setPersistedUserState(state)
}
