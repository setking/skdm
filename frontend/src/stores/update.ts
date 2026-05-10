import { ref, readonly } from 'vue'
import { defineStore } from 'pinia'
import { Events } from '@wailsio/runtime'
import { CheckForUpdate } from '@bindings/changeme/backed/internal/pkg/server/config'

/** 后端 UpdateCheckResult 对应的前端类型 */
export interface UpdateCheckResult {
  has_update: boolean
  current_version: string
  latest_version: string
  release_url: string
  error: string
}

export const useUpdateStore = defineStore('update', () => {
  const result = ref<UpdateCheckResult | null>(null)
  const checking = ref(false)

  /** 订阅启动时自动检查结果（后端仅 emit 一次） */
  Events.On('update-check', (ev: { data: UpdateCheckResult }) => {
    result.value = ev.data
  })

  /** 手动检查更新 */
  async function manualCheck() {
    checking.value = true
    try {
      const r = await CheckForUpdate()
      result.value = {
        has_update: r?.has_update ?? false,
        current_version: r?.current_version ?? '',
        latest_version: r?.latest_version ?? '',
        release_url: r?.release_url ?? '',
        error: r?.error ?? '',
      }
      return result.value
    } finally {
      checking.value = false
    }
  }

  return {
    result: readonly(result),
    checking: readonly(checking),
    manualCheck,
  }
})
