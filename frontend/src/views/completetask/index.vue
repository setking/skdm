<script setup lang="ts">
import { h, onMounted } from 'vue'
import { NButton, NTag, NSpace, NIcon, useMessage, useDialog } from 'naive-ui'
import { Remove, RemoveDownloadResult, DeleteWithLocalFile, OpenFileLocation } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { useDownloadStore } from '@/stores/download'
import { formatBytes } from '@/utils/format'
import TablePage from '@/components/table/index.vue'

const message = useMessage()
const dialog = useDialog()
const store = useDownloadStore()

function folderIcon() {
  return h('svg', { viewBox: '0 0 24 24', width: 16, height: 16, fill: 'currentColor' }, [
    h('path', { d: 'M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z' })
  ])
}

async function handleRemove(gid: string) {
  try { await Remove(gid) } catch (e: any) { message.warning('删除失败: ' + e) }
}
async function handleDelete(gid: string) {
  dialog.warning({
    title: '确认删除',
    content: '此操作将同时删除本地下载文件，是否继续？',
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await RemoveDownloadResult(gid).catch(() => {})
        await DeleteWithLocalFile(gid)
        message.success('已永久删除')
      } catch (e: any) { message.error('删除失败: ' + e) }
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
  { title: '位置', key: 'dir', minWidth: 50, width: 50,
    render(row: DownloadRecord) {
      return h(NButton, { size: 'tiny', quaternary: true, onClick: () => handleOpenFolder(row.gid), title: row.dir || '' },
        () => h(NIcon, { size: 16 }, folderIcon))
    }
  },
  { title: '完成时间', key: 'completed_at', minWidth: 130, width: 170, render(row: DownloadRecord) { return row.completed_at?.replace('T', ' ').slice(0, 19) || row.updated_at?.replace('T', ' ').slice(0, 19) || '-' } },
  { title: '状态', key: 'status', minWidth: 70, width: 80,
    render() { return h(NTag, { type: 'success', size: 'small' }, () => '已完成') }
  },
  { title: '操作', key: 'actions', minWidth: 140, width: 180,
    render(row: DownloadRecord) {
      return h(NSpace, { size: 4 }, () => [
        h(NButton, { size: 'tiny', quaternary: true, onClick: () => handleRemove(row.gid) }, () => '移到回收站'),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.gid) }, () => '永久删除'),
      ])
    }
  },
]

onMounted(() => {
  store.init()
})
</script>

<template>
  <TablePage
    title="已完成"
    :count="store.completedDownloads.length"
    empty-description="暂无已完成的任务"
    :columns="columns"
    :data="store.completedDownloads"
  />
</template>
