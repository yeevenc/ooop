type UploadResponseData = string | {
  url?: string
  path?: string
}

function joinUrl(baseURL: string, path: string) {
  return `${baseURL.replace(/\/+$/, '')}/${path.replace(/^\/+/, '')}`
}

export function getUploadImageAction() {
  const configuredURL = import.meta.env.VITE_AP_BASE_FILE_URL

  if (configuredURL && configuredURL.includes('/upload/image')) {
    return configuredURL
  }

  return joinUrl(import.meta.env.VITE_API_BASE_URL, '/upload/image')
}

export function resolveUploadURL(data?: UploadResponseData | null) {
  if (!data) {
    return ''
  }

  if (typeof data === 'string') {
    return data
  }

  return data.url || data.path || ''
}
