<script setup lang="ts">
import { ref, watch } from 'vue'
import { DownloadOutline, FolderOpenOutline, LinkOutline } from '@vicons/ionicons5'
import { Dialogs } from '@wailsio/runtime'
import { AddURI, GetDefaultDownloadDir } from '@bindings/changeme/backed/api/apiserver/aria2service'
import { Options } from '@bindings/github.com/siku2/arigo/models'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const url = ref('')
const savePath = ref('./download')
const fileName = ref('')
const loading = ref(false)

// 每次弹窗打开时重新读取保存的下载目录和剪贴板
watch(() => props.show, async (visible) => {
  if (!visible) return

  // 从 SQLite 读取上次使用的下载目录
  try {
    const savedDir = await GetDefaultDownloadDir()
    if (savedDir) {
      savePath.value = savedDir
    }
  } catch {
    // 使用默认值
  }

  // 自动从剪贴板粘贴下载链接
  try {
    const text = await navigator.clipboard.readText()
    if (text && isDownloadLink(text)) {
      url.value = text
      fileName.value = extractFileName(text)
    }
  } catch {
    // clipboard read may fail
  }
})

// 支持的下载协议
const DOWNLOAD_PROTOCOLS = ['http://', 'https://', 'magnet:', 'ftp://', 'ed2k://', 'thunder://']

function isDownloadLink(link: string): boolean {
  return DOWNLOAD_PROTOCOLS.some(p => link.startsWith(p))
}

function extractFileName(link: string): string {
  // ed2k 链接
  if (link.startsWith('ed2k://')) {
    const match = link.match(/\|([^|]+)\|(\d+)\|/)
    if (match) {
      return decodeURIComponent(match[1])
    }
    return 'ed2k_download'
  }

  // magnet 链接
  if (link.startsWith('magnet:')) {
    const match = link.match(/dn=([^&]+)/)
    return match ? decodeURIComponent(match[1]) : 'magnet_download'
  }

  // http/https/ftp
  try {
    const urlObj = new URL(link)
    const pathname = urlObj.pathname
    const name = pathname.substring(pathname.lastIndexOf('/') + 1)
    return decodeURIComponent(name) || 'download'
  } catch {
    return 'download'
  }
}

function onUrlChange() {
  fileName.value = extractFileName(url.value)
}

// 选择下载目录
async function selectFolder() {
  try {
    const folder = await Dialogs.OpenFile({
      CanChooseDirectories: true,
      CanChooseFiles: false,
      CanCreateDirectories: true,
      Title: '选择下载目录',
    })
    if (folder) {
      savePath.value = folder as string
    }
  } catch (e) {
    console.error('选择目录失败:', e)
  }
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
  savePath.value = './download'
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
            <n-button size="tiny" :disabled="loading" @click="selectFolder">浏览</n-button>
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
        <n-button @click="handleCancel">取消</n-button>
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
