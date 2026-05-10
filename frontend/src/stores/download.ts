import { ref, reactive, computed, onUnmounted } from 'vue'
import { defineStore } from 'pinia'
import { Events } from '@wailsio/runtime'
import type { DownloadRecord } from '@bindings/changeme/backed/api/apiserver/v1'
import { ListDownloads } from '@bindings/changeme/backed/internal/pkg/server/config'

export const useDownloadStore = defineStore('downloads', () => {
  // 核心数据：以 gid 为 key 的下载记录表
  const records = ref<Record<string, DownloadRecord>>({})

  // 初始化标志
  const initialized = ref(false)

  // 兜底轮询定时器
  let fallbackTimer: ReturnType<typeof setInterval> | null = null

  // 事件取消函数
  let unsubUpdate: (() => void) | null = null
  let unsubRemoved: (() => void) | null = null

  // ==================== 视图计算属性 ====================

  /** 全部下载列表（按 created_at 降序） */
  const allDownloads = computed<DownloadRecord[]>(() => {
    return Object.values(records.value).sort((a, b) =>
      (b.created_at || '').localeCompare(a.created_at || '')
    )
  })

  /** 正在下载 (active, waiting, paused) */
  const runningDownloads = computed<DownloadRecord[]>(() =>
    allDownloads.value.filter(d =>
      d.status === 'active' || d.status === 'waiting' || d.status === 'paused'
    )
  )

  /** 未完成 (error, paused) */
  const unfinishedDownloads = computed<DownloadRecord[]>(() =>
    allDownloads.value.filter(d =>
      d.status === 'error' || d.status === 'paused'
    )
  )

  /** 已完成 */
  const completedDownloads = computed<DownloadRecord[]>(() =>
    allDownloads.value.filter(d => d.status === 'complete')
  )

  /** 回收站 */
  const trashedDownloads = computed<DownloadRecord[]>(() =>
    allDownloads.value.filter(d => d.status === 'removed')
  )

  // ==================== 初始化 ====================

  async function init() {
    if (initialized.value) return

    // 从 SQLite 加载初始全量数据
    try {
      const [list] = await ListDownloads('', 0, 1000)
      const map: Record<string, DownloadRecord> = {}
      for (const r of list || []) {
        map[r.gid] = r
      }
      records.value = map
    } catch (e) {
      console.error('[DownloadStore] 初始加载失败:', e)
    }

    // 订阅实时事件
    setupEventListeners()

    // 启动兜底轮询（每 60 秒全量刷新，防止事件丢失）
    fallbackTimer = setInterval(async () => {
      try {
        const [list] = await ListDownloads('', 0, 1000)
        const map: Record<string, DownloadRecord> = {}
        for (const r of list || []) {
          map[r.gid] = r
        }
        records.value = map
      } catch (e) {
        console.error('[DownloadStore] 兜底刷新失败:', e)
      }
    }, 60000)

    initialized.value = true
  }

  // ==================== 事件处理 ====================

  function setupEventListeners() {
    // 监听单条下载更新（来自 api 事件 + 进度轮询）
    unsubUpdate = Events.On('download-update', (ev: { data: DownloadRecord }) => {
      const dr = ev.data
      if (!dr || !dr.gid) return
      records.value = { ...records.value, [dr.gid]: dr }
    })

    // 监听下载移除
    unsubRemoved = Events.On('download-removed', (ev: { data: string }) => {
      const gid = ev.data
      if (!gid) return
      const next = { ...records.value }
      delete next[gid]
      records.value = next
    })
  }

  // ==================== 销毁 ====================

  function destroy() {
    if (fallbackTimer) {
      clearInterval(fallbackTimer)
      fallbackTimer = null
    }
    if (unsubUpdate) { unsubUpdate(); unsubUpdate = null }
    if (unsubRemoved) { unsubRemoved(); unsubRemoved = null }
    initialized.value = false
    records.value = {}
  }

  return {
    records,
    initialized,
    init,
    destroy,
    // 视图
    allDownloads,
    runningDownloads,
    unfinishedDownloads,
    completedDownloads,
    trashedDownloads,
  }
})
