import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'

export function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

export function formatSpeed(bytes: number): string {
  if (bytes <= 0) return '0 B/s'
  return formatBytes(bytes) + '/s'
}

export function progressPercent(row: DownloadRecord): number {
  if (row.total_length <= 0) return 0
  return Math.round((row.completed_length / row.total_length) * 100)
}
