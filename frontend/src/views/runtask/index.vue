<script setup lang="ts">
import { ref, onMounted, onUnmounted, h, computed } from 'vue'
import { NButton, NTag, NProgress, NSpace, useMessage } from 'naive-ui'
import { ListDownloads, Pause, Unpause, Remove } from '@bindings/changeme/backed/api/apiserver/aria2service'
import type { DownloadRecord } from '@bindings/changeme/backed/pkg/store/models'

const message = useMessage()
const allDownloads = ref<DownloadRecord[]>([])
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const downloads = computed(() =>
  allDownloads.value.filter(d => d.status === 'active' || d.status === 'waiting' || d.status === 'paused')
)

function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let size = bytes
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

async function fetchDownloads() {
  loading.value = true
  try {
    const [records] = await ListDownloads('', 0, 1000)
    allDownloads.value = records || []
  } catch (e) {
    console.error('获取下载列表失败:', e)
  } finally { loading.value = false }
}

async function handlePause(gid: string) {
  try { await Pause(gid) } catch (e: any) { message.warning('暂停失败: ' + e) }
  await fetchDownloads()
}
async function handleUnpause(gid: string) {
  try { await Unpause(gid) } catch (e: any) { message.warning('恢复失败: ' + e) }
  await fetchDownloads()
}
async function handleRemove(gid: string) {
  try { await Remove(gid) } catch (e: any) { message.warning('删除失败: ' + e) }
  await fetchDownloads()
}

const columns = [
  { title: '文件名', key: 'filename', ellipsis: { tooltip: true },
    render(row: DownloadRecord) { return row.filename || row.url?.split('/').pop() || row.gid?.slice(0, 8) } },
  { title: '大小', key: 'total_length', width: 100,
    render(row: DownloadRecord) { return row.total_length > 0 ? formatBytes(row.total_length) : '未知' } },
  { title: '进度', key: 'progress', width: 200,
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
  { title: '速度', key: 'download_speed', width: 100,
    render(row: DownloadRecord) { return row.status === 'active' ? formatSpeed(row.download_speed) : '-' } },
  { title: '状态', key: 'status', width: 80,
    render(row: DownloadRecord) {
      const s: Record<string, any> = { active: { label: '下载中', type: 'info' }, waiting: { label: '等待中', type: 'default' }, paused: { label: '已暂停', type: 'warning' } }
      const t = s[row.status] || { label: row.status, type: 'default' as const }
      return h(NTag, { type: t.type, size: 'small' }, () => t.label)
    }
  },
  { title: '保存到', key: 'dir', ellipsis: { tooltip: true }, width: 180 },
  { title: '操作', key: 'actions', width: 140,
    render(row: DownloadRecord) {
      const btns = []
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
  fetchDownloads()
  timer = setInterval(fetchDownloads, 3000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2 class="page-title">正在下载</h2>
      <span class="count">{{ downloads.length }} 个任务</span>
    </div>
    <div class="table-area">
      <n-empty v-if="!loading && downloads.length === 0" description="暂无正在下载的任务" style="margin-top: 120px" />
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
