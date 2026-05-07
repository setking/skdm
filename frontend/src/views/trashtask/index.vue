<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import { ListDownloads, DeleteDownloadRecord, PurgeDownloadResults, ContinueDownload, RemoveDownloadResult } from '@bindings/changeme/backed/api/apiserver/aria2service'
import type { DownloadRecord } from '@bindings/changeme/backed/pkg/store/models'

const message = useMessage()
const downloads = ref<DownloadRecord[]>([])
const loading = ref(false)

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

async function fetchDownloads() {
  loading.value = true
  try {
    const [records] = await ListDownloads('removed', 0, 1000)
    downloads.value = records || []
  } catch (e) {
    console.error('获取回收站列表失败:', e)
  } finally { loading.value = false }
}

async function handleDelete(gid: string) {
  try {
    await RemoveDownloadResult(gid).catch(() => {})
    await DeleteDownloadRecord(gid)
    message.success('已永久删除')
  } catch (e: any) { message.error('删除失败: ' + e) }
  await fetchDownloads()
}
async function handleContinue(gid: string) {
  try {
    await ContinueDownload(gid)
    message.success('已重新开始下载')
  } catch (e: any) { message.error('重新下载失败: ' + e) }
  await fetchDownloads()
}
async function handlePurgeAll() {
  try {
    await PurgeDownloadResults()
    for (const d of downloads.value) {
      await DeleteDownloadRecord(d.gid).catch(() => {})
    }
    await fetchDownloads()
    message.success('已清空回收站')
  } catch (e: any) { message.error('清空失败: ' + e) }
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true },
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '删除时间', key: 'updated_at', width: 170, render(row: DownloadRecord) { return row.updated_at?.replace('T', ' ').slice(0, 19) || '-' } },
  { title: '原链接', key: 'url', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 80,
    render() { return h(NTag, { type: 'default', size: 'small' }, () => '已删除') }
  },
  { title: '操作', key: 'actions', width: 170,
    render(row: DownloadRecord) {
      return h(NSpace, { size: 4 }, () => [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleContinue(row.gid) }, () => '重新下载'),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.gid) }, () => '永久删除'),
      ])
    }
  },
]

onMounted(fetchDownloads)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2 class="page-title">回收站</h2>
      <n-space align="center">
        <span class="count" v-if="downloads.length > 0">{{ downloads.length }} 个任务</span>
        <n-button v-if="downloads.length > 0" size="small" type="error" quaternary @click="handlePurgeAll">清空回收站</n-button>
      </n-space>
    </div>
    <div class="table-area">
      <n-empty v-if="!loading && downloads.length === 0" description="回收站为空" style="margin-top: 120px" />
      <n-data-table v-else :columns="columns" :data="downloads" :loading="loading" :bordered="false" striped size="small"
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
