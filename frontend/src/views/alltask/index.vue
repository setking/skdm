<script setup lang="ts">
import { h, onMounted } from 'vue'
import { NButton, NTag, NProgress, NSpace, useMessage } from 'naive-ui'
import { Remove, DeleteDownloadRecord, ContinueDownload, RemoveDownloadResult } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { useDownloadStore } from '@/stores/download'

const message = useMessage()
const store = useDownloadStore()

const statusMap: Record<string, { label: string; type: 'warning' | 'error' }> = {
  paused: { label: '已暂停', type: 'warning' },
  error: { label: '下载失败', type: 'error' },
}

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

function formatSpeed(bytes: number): string {
  if (bytes <= 0) return '0 B/s'
  return formatBytes(bytes) + '/s'
}

function progressPercent(row: DownloadRecord): number {
  if (row.total_length <= 0) return 0
  return Math.round((row.completed_length / row.total_length) * 100)
}

async function handleRemove(gid: string) {
  try { await Remove(gid) } catch (e: any) { message.warning('删除失败: ' + e) }
}
async function handleDelete(gid: string) {
  try {
    await RemoveDownloadResult(gid).catch(() => {})
    await DeleteDownloadRecord(gid)
    message.success('已永久删除')
  } catch (e: any) { message.error('删除失败: ' + e) }
}
async function handleContinue(gid: string) {
  try {
    await ContinueDownload(gid)
    message.success('已重新开始下载')
  } catch (e: any) { message.error('继续下载失败: ' + e) }
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true }, width: 200,
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '进度', key: 'progress', width: 160,
    render(row: DownloadRecord) {
      const hasTotal = row.total_length > 0
      const pct = hasTotal ? progressPercent(row) : 0
      const indeterminate = !hasTotal && row.status === 'active'
      const status = row.status === 'error' ? 'error' as const : row.status === 'complete' ? 'success' as const : undefined
      return h(NProgress, {
        percentage: pct, processing: indeterminate, status, style: { width: '100%' },
        default: () => {
          if (row.status === 'active' || row.status === 'paused') {
            return hasTotal ? `${formatBytes(row.completed_length)} / ${formatBytes(row.total_length)}` : `计算中... ${formatBytes(row.completed_length)}`
          }
          return undefined
        }
      })
    }
  },
  { title: '速度', key: 'download_speed', width: 100,
    render(row: DownloadRecord) { return row.status === 'active' ? formatSpeed(row.download_speed) : '-' } },
  { title: '状态', key: 'status', width: 80,
    render(row: DownloadRecord) {
      const s = statusMap[row.status] || { label: row.status, type: 'default' as const }
      return h(NTag, { type: s.type, size: 'small' }, () => s.label)
    }
  },
  { title: '创建时间', key: 'created_at', width: 160, render(row: DownloadRecord) { return row.created_at?.replace('T', ' ').slice(0, 19) || '-' } },
  { title: '操作', key: 'actions', width: 180,
    render(row: DownloadRecord) {
      const btns: any[] = []
      if (row.status === 'paused') {
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleContinue(row.gid) }, () => '恢复下载'))
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleRemove(row.gid) }, () => '移到回收站'))
      } else if (row.status === 'error') {
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleContinue(row.gid) }, () => '重试'))
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.gid) }, () => '删除'))
      }
      return h(NSpace, { size: 4 }, () => btns)
    }
  },
]

onMounted(() => {
  store.init()
})
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2 class="page-title">未完成</h2>
      <span class="count">{{ store.unfinishedDownloads.length }} 个任务</span>
    </div>
    <div class="table-area">
      <n-empty v-if="store.unfinishedDownloads.length === 0" description="暂无未完成的任务" style="margin-top: 120px" />
      <n-data-table v-else :columns="columns" :data="store.unfinishedDownloads" :bordered="false" striped size="small"
        flex-height style="height: 100%" />
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px 24px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; margin-bottom: 12px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.count { color: #999; font-size: 13px; }
.table-area { flex: 1; min-height: 0; }
</style>
