<script setup lang="ts">
import { useUpdateStore } from '@/stores/update'
import { Browser } from '@wailsio/runtime'

const store = useUpdateStore()

function handleDownload() {
  const url = store.result?.download_url || store.result?.release_url
  if (url) {
    Browser.OpenURL(url)
    store.dismissUpdate()
  }
}

function handleDismiss() {
  store.dismissUpdate()
}
</script>

<template>
  <n-modal
    :show="store.showDialog"
    :on-update:show="(val: boolean) => { if (!val) handleDismiss() }"
    preset="card"
    title="发现新版本"
    style="width: 480px"
    :closable="true"
    :mask-closable="false"
  >
    <div class="update-body">
      <div class="version-row">
        <span class="label">当前版本</span>
        <n-tag type="default" size="small">v{{ store.result?.current_version }}</n-tag>
      </div>
      <div class="version-row">
        <span class="label">最新版本</span>
        <n-tag type="success" size="small">v{{ store.result?.latest_version }}</n-tag>
      </div>

      <div v-if="store.result?.release_notes" class="release-notes">
        <div class="label">更新内容</div>
        <div class="notes-content">{{ store.result.release_notes }}</div>
      </div>

      <div v-if="!store.result?.release_notes" class="release-notes">
        <div class="label">更新内容</div>
        <div class="notes-content">
          <a :href="store.result?.release_url" target="_blank">{{ store.result?.release_url }}</a>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="update-footer">
        <n-button @click="handleDismiss">以后再说</n-button>
        <n-button type="primary" @click="handleDownload">
          {{ store.result?.download_url ? '立即下载更新' : '查看详情' }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.update-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.version-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.label {
  font-size: 13px;
  color: #666;
  min-width: 64px;
}
.release-notes {
  margin-top: 4px;
}
.notes-content {
  margin-top: 6px;
  padding: 10px 12px;
  background: #f7f7f7;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
}
.update-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
