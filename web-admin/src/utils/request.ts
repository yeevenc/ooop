import axios, {
  AxiosHeaders,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
  AxiosError,
} from 'axios'
import { ElMessage } from 'element-plus'
import { getToken, removeToken } from '@/utils/auth'
import router from '@/router'

let isRedirectingToLogin = false

export interface ApiResponseData<T = unknown> {
  code: number
  data: T
  message: string
}

export interface RequestConfig<T = unknown> extends AxiosRequestConfig<T> {
  showError?: boolean
  withToken?: boolean
}

export interface BlobRequestConfig<T = unknown> extends RequestConfig<T> {
  responseType: 'blob'
}

const DEFAULT_ERROR_MESSAGE = '服务异常，请稍后重试'
const SUCCESS_CODE_LIST = [0, 200]
const UNAUTHORIZED_CODE = 401

function showErrorMessage(message?: string) {
  ElMessage.error(message || DEFAULT_ERROR_MESSAGE)
}

function resolveAuthorization(token: string) {
  return `Bearer ${token}`
}

function shouldShowError(config?: RequestConfig) {
  return config?.showError !== false
}

function createRequestService(): AxiosInstance {
  const service = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL,
    timeout: 10000,
  })

  service.interceptors.request.use(
    (config) => {
      const requestConfig = config as InternalAxiosRequestConfig & RequestConfig

      if (requestConfig.withToken === false) {
        return config
      }

      const token = getToken()

      if (!token) {
        return config
      }

      const headers = new AxiosHeaders(config.headers)
      headers.set('Authorization', resolveAuthorization(token))
      config.headers = headers

      return config
    },
    (error: AxiosError) => Promise.reject(error),
  )

  service.interceptors.response.use(
    (response: AxiosResponse<ApiResponseData>) => {
      const requestConfig = response.config as RequestConfig

      if (response.config.responseType === 'blob') {
        return response
      }

      const responseData = response.data

      if (SUCCESS_CODE_LIST.includes(responseData.code)) {
        return response
      }

      if (responseData.code === UNAUTHORIZED_CODE) {
        removeToken()
        if (!isRedirectingToLogin && router.currentRoute.value.path !== '/login') {
          isRedirectingToLogin = true
          router.push('/login').finally(() => {
            isRedirectingToLogin = false
          })
        }
        return Promise.reject(responseData)
      }

      if (shouldShowError(requestConfig)) {
        showErrorMessage(responseData.message)
      }

      return Promise.reject(responseData)
    },
    (error: AxiosError<ApiResponseData>) => {
    const requestConfig = error.config as RequestConfig | undefined
    const status = error.response?.status
    const message = error.response?.data?.message || error.message || DEFAULT_ERROR_MESSAGE

    if (status === 401) {
      removeToken()
      if (!isRedirectingToLogin && router.currentRoute.value.path !== '/login') {
        isRedirectingToLogin = true
        router.push('/login').finally(() => {
          isRedirectingToLogin = false
        })
        return Promise.reject(error)
      }
      // On login page (or redirect already in progress): fall through so showErrorMessage runs below
    }

    if (status === 500) {
      router.push('/500')
      return Promise.reject(error)
    }

    if (shouldShowError(requestConfig)) {
      showErrorMessage(message)
    }

    return Promise.reject(error)
  },
  )

  return service
}

const service = createRequestService()

export default service

export function request<T = unknown>(config: BlobRequestConfig): Promise<Blob>
export function request<T = unknown>(config: RequestConfig): Promise<ApiResponseData<T>>
export function request<T = unknown>(config: RequestConfig): Promise<ApiResponseData<T> | Blob> {
  if (config.responseType === 'blob') {
    return service
      .request<Blob, AxiosResponse<Blob>>(config)
      .then((response) => response.data)
  }

  return service
    .request<ApiResponseData<T>, AxiosResponse<ApiResponseData<T>>>(config)
    .then((response) => response.data)
}

export function get<T = unknown>(url: string, config?: RequestConfig) {
  return request<T>({
    ...config,
    url,
    method: 'get',
  })
}

export function post<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig<D>) {
  return request<T>({
    ...config,
    url,
    data,
    method: 'post',
  })
}

export function put<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig<D>) {
  return request<T>({
    ...config,
    url,
    data,
    method: 'put',
  })
}

export function del<T = unknown>(url: string, config?: RequestConfig) {
  return request<T>({
    ...config,
    url,
    method: 'delete',
  })
}
