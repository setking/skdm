<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { DownloadOutline, FolderOpenOutline, LinkOutline } from '@vicons/ionicons5'
import { AddURI } from '@bindings/changeme/backed/api/aria2server/aria2service'
import { Options } from '@bindings/github.com/siku2/arigo/models'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const url = ref('')
const savePath = ref('')
const fileName = ref('')
const loading = ref(false)

onMounted(async () => {
  // Auto-paste URL from clipboard
  try {
    const text = await navigator.clipboard.readText()
    if (text && (text.startsWith('http') || text.startsWith('magnet:') || text.startsWith('ftp://') || text.startsWith('ed2k://') || text.startsWith('thunder://'))) {
      url.value = text
    }
  } catch {
    // clipboard read may fail
  }
})

function extractFileName(link: string): string {
  try {
    const urlObj = new URL(link)
    const pathname = urlObj.pathname
    const name = pathname.substring(pathname.lastIndexOf('/') + 1)
    return decodeURIComponent(name) || 'download'
  } catch {
    // magnet links or other protocols
    if (link.startsWith('magnet:')) {
      const match = link.match(/dn=([^&]+)/)
      return match ? decodeURIComponent(match[1]) : 'magnet_download'
    }
    return 'download'
  }
}

function onUrlChange() {
  fileName.value = extractFileName(url.value)
}

async function handleOk() {
  if (!url.value.trim()) return

  loading.value = true
  try {
    const options = new Options({
      dir: savePath.value || undefined,
      out: fileName.value || undefined,
    })
    await AddURI([url.value.trim()], options)
    emit('update:show', false)
    resetForm()
  } catch (e) {
    console.error('添加下载任务失败:', e)
  } finally {
    loading.value = false
  }
}

function handleCancel() {
  emit('update:show', false)
  resetForm()
}

function resetForm() {
  url.value = ''
  savePath.value = ''
  fileName.value = ''
}
</script>

<template>
  <n-modal :show="show" :on-update:show="(val: boolean) => emit('update:show', val)" title="新建下载任务" preset="card"
    style="width: 520px" :closable="!loading" :mask-closable="!loading">
    <div class="dialog-body">
      <!-- Download link -->
      <div class="field">
        <label class="label">下载链接</label>
        <n-input v-model:value="url" placeholder="支持 http, magnet, ftp, ed2k 等链接" @change="onUrlChange"
          :disabled="loading">
          <template #prefix>
            <n-icon :component="LinkOutline" />
          </template>
        </n-input>
      </div>

      <!-- Save path -->
      <div class="field">
        <label class="label">保存到</label>
        <n-input v-model:value="savePath" placeholder="默认下载目录" :disabled="loading">
          <template #prefix>
            <n-icon :component="FolderOpenOutline" />
          </template>
          <template #suffix>
            <n-button size="tiny" :disabled="loading">浏览</n-button>
          </template>
        </n-input>
      </div>

      <!-- File name -->
      <div class="field">
        <label class="label">文件名</label>
        <n-input v-model:value="fileName" placeholder="自动识别文件名" :disabled="loading">
          <template #prefix>
            <n-icon :component="DownloadOutline" />
          </template>
        </n-input>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <n-button @click="handleCancel" :disabled="loading">取消</n-button>
        <n-button type="primary" @click="handleOk" :loading="loading" :disabled="!url.trim()">立即下载</n-button>
      </div>
    </template>
  </n-modal>
</template>

<style lang="scss" scoped>
.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  font-size: 13px;
  color: #666;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
