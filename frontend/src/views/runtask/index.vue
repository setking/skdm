<script setup lang="ts">
import { h, onMounted } from 'vue'
import { NButton, NTag, NProgress, NSpace, NIcon, useMessage, useDialog } from 'naive-ui'
import { Pause, Unpause, Remove, DeleteWithLocalFile, OpenFileLocation } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { useDownloadStore } from '@/stores/download'
import { formatBytes, formatSpeed, progressPercent } from '@/utils/format'
import TablePage from '@/components/table/index.vue'

const message = useMessage()
const dialog = useDialog()
const store = useDownloadStore()

function folderIcon() {
  return h('svg', { viewBox: '0 0 24 24', width: 16, height: 16, fill: 'currentColor' }, [
    h('path', { d: 'M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z' })
  ])
}

async function handlePause(gid: string) {
  try { await Pause(gid) } catch (e: any) { message.warning('暂停失败: ' + e) }
}
async function handleUnpause(gid: string) {
  try { await Unpause(gid) } catch (e: any) { message.warning('恢复失败: ' + e) }
}
async function handleRemove(gid: string) {
  dialog.warning({
    title: '确认删除',
    content: '将此任务移出下载列表并删除本地已下载文件，是否继续？',
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await Remove(gid)
        await DeleteWithLocalFile(gid)
      } catch (e: any) { message.warning('删除失败: ' + e) }
    }
  })
}
async function handleOpenFolder(gid: string) {
  try { await OpenFileLocation(gid) } catch (e: any) { message.warning('打开失败: ' + e) }
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true }, minWidth: 120,
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', minWidth: 80, width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '进度', key: 'progress', minWidth: 140, width: 220,
    render(row: DownloadRecord) {
      const hasTotal = row.total_length > 0
      const pct = hasTotal ? progressPercent(row) : 0
      const indeterminate = !hasTotal && row.status === 'active'
      return h(NProgress, {
        percentage: pct, processing: indeterminate, style: { width: '100%' },
        default: () => {
          if (row.status === 'waiting') return undefined
          if (hasTotal) return `${formatBytes(row.completed_length)} / ${formatBytes(row.total_length)}`
          return `计算中... ${formatBytes(row.completed_length)}`
        }
      })
    }
  },
  { title: '速度', key: 'download_speed', minWidth: 80, width: 100,
    render(row: DownloadRecord) { return row.status === 'active' ? formatSpeed(row.download_speed) : '-' } },
  { title: '状态', key: 'status', minWidth: 70, width: 80,
    render(row: DownloadRecord) {
      const s: Record<string, any> = { active: { label: '下载中', type: 'info' }, waiting: { label: '等待中', type: 'default' }, paused: { label: '已暂停', type: 'warning' } }
      const t = s[row.status] || { label: row.status, type: 'default' as const }
      return h(NTag, { type: t.type, size: 'small' }, () => t.label)
    }
  },
  { title: '位置', key: 'dir', minWidth: 50, width: 50,
    render(row: DownloadRecord) {
      return h(NButton, { size: 'tiny', quaternary: true, onClick: () => handleOpenFolder(row.gid), title: row.dir || '' },
        () => h(NIcon, { size: 16 }, folderIcon))
    }
  },
  { title: '操作', key: 'actions', minWidth: 120, width: 140,
    render(row: DownloadRecord) {
      const btns: any[] = []
      if (row.status === 'active' || row.status === 'waiting') {
        btns.push(h(NButton, { size: 'tiny', quaternary: true, onClick: () => handlePause(row.gid) }, () => '暂停'))
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleRemove(row.gid) }, () => '删除'))
      } else if (row.status === 'paused') {
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleUnpause(row.gid) }, () => '继续'))
        btns.push(h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleRemove(row.gid) }, () => '删除'))
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
    title="正在下载"
    :count="store.runningDownloads.length"
    empty-description="暂无正在下载的任务"
    :columns="columns"
    :data="store.runningDownloads"
  />
</template>
