<script setup lang="ts">
import { h, onMounted } from 'vue'
import { NButton, NTag, NProgress, NSpace, useMessage, useDialog } from 'naive-ui'
import { Remove, ContinueDownload, DeleteWithLocalFile } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { useDownloadStore } from '@/stores/download'
import { formatBytes, formatSpeed, progressPercent } from '@/utils/format'
import TablePage from '@/components/table/index.vue'

const message = useMessage()
const dialog = useDialog()
const store = useDownloadStore()

const statusMap: Record<string, { label: string; type: 'warning' | 'error' }> = {
  paused: { label: '已暂停', type: 'warning' },
  error: { label: '下载失败', type: 'error' },
}

async function handleDelete(gid: string) {
  dialog.warning({
    title: '确认删除',
    content: '将移出下载列表并删除本地文件，此操作不可恢复',
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Remove(gid)
        await DeleteWithLocalFile(gid)
      } catch (e: any) { message.error('删除失败: ' + e) }
    }
  })
}
async function handleContinue(gid: string) {
  try {
    await ContinueDownload(gid)
    message.success('已重新开始下载')
  } catch (e: any) { message.error('继续下载失败: ' + e) }
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true }, minWidth: 120,
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', minWidth: 80, width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '进度', key: 'progress', minWidth: 130, width: 170,
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
  { title: '速度', key: 'download_speed', minWidth: 80, width: 100,
    render(row: DownloadRecord) { return row.status === 'active' ? formatSpeed(row.download_speed) : '-' } },
  { title: '状态', key: 'status', minWidth: 70, width: 80,
    render(row: DownloadRecord) {
      const s = statusMap[row.status] || { label: row.status, type: 'default' as const }
      return h(NTag, { type: s.type, size: 'small' }, () => s.label)
    }
  },
  { title: '创建时间', key: 'created_at', minWidth: 130, width: 170, render(row: DownloadRecord) { return row.created_at?.replace('T', ' ').slice(0, 19) || '-' } },
  { title: '操作', key: 'actions', minWidth: 140, width: 180,
    render(row: DownloadRecord) {
      const btns: any[] = []
      if (row.status === 'paused') {
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleContinue(row.gid) }, () => '恢复下载'))
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.gid) }, () => '删除'))
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
  <TablePage
    title="未完成"
    :count="store.unfinishedDownloads.length"
    empty-description="暂无未完成的任务"
    :columns="columns"
    :data="store.unfinishedDownloads"
  />
</template>
