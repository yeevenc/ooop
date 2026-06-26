function padTime(value: number) {
  return String(value).padStart(2, '0')
}

export function formatDateTime(value?: string | null) {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }

  const year = date.getFullYear()
  const month = padTime(date.getMonth() + 1)
  const day = padTime(date.getDate())
  const hours = padTime(date.getHours())
  const minutes = padTime(date.getMinutes())
  const seconds = padTime(date.getSeconds())

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}
