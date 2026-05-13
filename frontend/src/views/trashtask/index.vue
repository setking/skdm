<script setup lang="ts">
import { h, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { DeleteDownloadRecord, PurgeDownloadResults, ContinueDownload, RemoveDownloadResult, DeleteWithLocalFile } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { useDownloadStore } from '@/stores/download'
import { formatBytes } from '@/utils/format'
import TablePage from '@/components/table/index.vue'

const message = useMessage()
const dialog = useDialog()
const store = useDownloadStore()

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
        await DeleteDownloadRecord(gid).catch(() => {})
        message.success('已永久删除')
      } catch (e: any) { message.error('删除失败: ' + e) }
    }
  })
}
async function handleContinue(gid: string) {
  try {
    await ContinueDownload(gid)
    message.success('已重新开始下载')
  } catch (e: any) { message.error('重新下载失败: ' + e) }
}
async function handlePurgeAll() {
  dialog.warning({
    title: '确认清空',
    content: '将清空回收站中所有记录（不删除本地文件），是否继续？',
    positiveText: '确认清空',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await PurgeDownloadResults()
        const items = [...store.trashedDownloads]
        for (const d of items) {
          await DeleteDownloadRecord(d.gid).catch(() => {})
        }
        message.success('已清空回收站')
      } catch (e: any) { message.error('清空失败: ' + e) }
    }
  })
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true }, minWidth: 120,
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', minWidth: 80, width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '删除时间', key: 'updated_at', minWidth: 130, width: 170, render(row: DownloadRecord) { return row.updated_at?.replace('T', ' ').slice(0, 19) || '-' } },
  { title: '原链接', key: 'url', ellipsis: { tooltip: true }, minWidth: 120 },
  { title: '状态', key: 'status', minWidth: 70, width: 80,
    render() { return h(NTag, { type: 'default', size: 'small' }, () => '已删除') }
  },
  { title: '操作', key: 'actions', minWidth: 140, width: 170,
    render(row: DownloadRecord) {
      return h(NSpace, { size: 4 }, () => [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleContinue(row.gid) }, () => '重新下载'),
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
    title="回收站"
    :count="store.trashedDownloads.length"
    empty-description="回收站为空"
    :columns="columns"
    :data="store.trashedDownloads"
  >
    <template #header-right>
      <n-space align="center">
        <span v-if="store.trashedDownloads.length > 0" class="count">{{ store.trashedDownloads.length }} 个任务</span>
        <n-button v-if="store.trashedDownloads.length > 0" size="small" type="error" quaternary @click="handlePurgeAll">清空回收站</n-button>
      </n-space>
    </template>
  </TablePage>
</template>

<style scoped>
.count { color: #999; font-size: 13px; }
</style>
