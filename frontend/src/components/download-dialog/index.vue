<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { DownloadOutline, FolderOpenOutline, LinkOutline } from '@vicons/ionicons5'
import { Dialogs } from '@wailsio/runtime'
import { AddURI, GetDefaultDownloadDir, FindDownloadByURL, DeleteDownloadRecord } from '@bindings/changeme/backed/api/apiserver/aria2service'
import type { DownloadRecord } from '@bindings/changeme/backed/pkg/store/models'
import type { CancellablePromiseLike } from '@wailsio/runtime'
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
const message = useMessage()

// 持有当前 AddURI 请求的引用，用于取消
let pendingPromise: CancellablePromiseLike<unknown> | null = null

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
      const name = match[1]
      return name ? decodeURIComponent(name) : 'ed2k_download'
    }
    return 'ed2k_download'
  }

  // magnet 链接
  if (link.startsWith('magnet:')) {
    const match = link.match(/dn=([^&]+)/)
    if (match) {
      const name = match[1]
      return name ? decodeURIComponent(name) : 'magnet_download'
    }
    return 'magnet_download'
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

function parseDownloadError(err: unknown): string {
  const msg: string = (err as { message?: string; toString?: () => string })?.message || (err as { toString?: () => string })?.toString?.() || ''
  if (!msg) return '未知错误，请检查网络连接'

  // aria2 RPC 返回的错误消息
  if (msg.includes('No URI') || msg.includes('no uri') || msg.includes('no URI')) {
    return '无法连接到下载链接，请检查链接是否有效或远程服务器是否可用'
  }
  if (msg.includes('unrecognized URI') || msg.includes('not valid') || msg.includes('unsupported protocol')) {
    return '下载链接格式不正确或协议不受支持'
  }
  if (msg.includes('timeout') || msg.includes('Timeout') || msg.includes('connection refused')) {
    return '连接远程服务器超时，请检查链接或稍后重试'
  }
  if (msg.includes('403') || msg.includes('Forbidden')) {
    return '远程服务器拒绝访问（403），链接可能需要认证或已失效'
  }
  if (msg.includes('404') || msg.includes('Not Found')) {
    return '远程服务器找不到文件（404），请检查下载链接是否正确'
  }
  if (msg.includes('500') || msg.includes('Internal Server Error')) {
    return '远程服务器内部错误（500），请稍后重试'
  }
  if (msg.includes('duplicate') || msg.includes('already')) {
    return '该下载任务已存在'
  }
  if (msg.includes('DNS') || msg.includes('resolve') || msg.includes('no such host')) {
    return '无法解析域名，请检查网络连接'
  }

  // 兜底：展示原始信息的关键部分
  const short = msg.substring(0, 200)
  return `添加下载失败: ${short}`
}

/** 重复检测结果 */
type DuplicateResult =
  | { action: 'block'; reason: string }    // 禁止添加
  | { action: 'confirm'; record: DownloadRecord }  // 需用户确认
  | { action: 'retry'; oldGid: string }   // 自动覆盖并重试
  | { action: 'fresh' }                    // 无重复，正常添加

async function checkDuplicate(rawUrl: string): Promise<DuplicateResult> {
  try {
    const existing = await FindDownloadByURL(rawUrl)
    if (!existing) return { action: 'fresh' }

    switch (existing.status) {
      case 'active':
      case 'waiting':
      case 'paused':
        return { action: 'block', reason: '该任务已在下载队列中，不能重复添加' }
      case 'complete':
        return { action: 'confirm', record: existing }
      case 'error':
      case 'removed':
        return { action: 'retry', oldGid: existing.gid }
      default:
        return { action: 'fresh' }
    }
  } catch {
    // 查询失败时放行
    return { action: 'fresh' }
  }
}

async function handleOk() {
  const rawUrl = url.value.trim()
  if (!rawUrl) return

  loading.value = true
  try {
    // 1. 检查重复
    const dup = await checkDuplicate(rawUrl)

    // 2. 已在队列中 → 阻止
    if (dup.action === 'block') {
      message.warning(dup.reason)
      return
    }

    // 3. 已完成 → 询问用户是否重新下载
    if (dup.action === 'confirm') {
      const confirmed = await new Promise<boolean>((resolve) => {
        const prevName = dup.record.filename || dup.record.url
        Dialogs.Question({
          Title: '确认重新下载',
          Message: `该任务（${prevName}）之前已下载完成。\n是否重新下载？重新下载将覆盖之前的任务记录。`,
          Buttons: [{ Label: '重新下载', IsDefault: true }, { Label: '取消', IsCancel: true }],
        }).then((resp: string) => {
          resolve(resp === '重新下载')
        }).catch(() => resolve(false))
      })
      if (!confirmed) return
      // 删除旧记录
      await DeleteDownloadRecord(dup.record.gid).catch(() => {})
    }

    // 4. 出错/已删除 → 自动覆盖重试
    if (dup.action === 'retry') {
      await DeleteDownloadRecord(dup.oldGid).catch(() => {})
    }

    // 5. 提交新下载
    const options = new Options({
      dir: savePath.value || undefined,
      out: fileName.value || undefined,
    })
    pendingPromise = AddURI([rawUrl], options)
    await pendingPromise
    message.success('下载任务已创建')
    emit('update:show', false)
    resetForm()
  } catch (e: unknown) {
    if (isCancelError(e)) return
    const errMsg = parseDownloadError(e)
    message.error(errMsg, { duration: 5000 })
  } finally {
    loading.value = false
    pendingPromise = null
  }
}

function isCancelError(err: unknown): boolean {
  const name = (err as { name?: string })?.name || ''
  return name === 'CancelError'
}

function handleCancel() {
  // 正在提交中，取消请求
  if (loading.value && pendingPromise) {
    pendingPromise.cancel('用户取消')
  }
  loading.value = false
  pendingPromise = null
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
  <n-modal :show="show" :on-update:show="(val: boolean) => { if (!val) handleCancel() }"
    title="新建下载任务" preset="card" style="width: 520px">
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
        <n-button @click="handleCancel">{{ loading ? '取消提交' : '取消' }}</n-button>
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
