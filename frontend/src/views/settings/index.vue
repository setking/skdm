<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import {
  NForm, NFormItem, NInput, NInputNumber, NSwitch, NButton,
  NCard, NSpin, NTag, useMessage
} from 'naive-ui'
import { Dialogs } from '@wailsio/runtime'
import { GetSettings, SaveSettings, GetAppVersion } from '@bindings/changeme/backed/internal/pkg/server/config'
import type { Settings } from '@bindings/changeme/backed/api/apiserver/v1'
import { useUpdateStore } from '@/stores/update'
import { FolderOpenOutline } from "@vicons/ionicons5";

const message = useMessage()
const updateStore = useUpdateStore()
const loading = ref(false)
const saving = ref(false)
const appVersion = ref('')

const form = ref<Settings>({
  default_download_dir: './download',
  max_concurrent_downloads: 5,
  max_connection_per_server: 1,
  split: 5,
  max_download_limit: 0,
  continue_download: true,
  allow_overwrite: true,
  auto_file_renaming: true,
  auto_check_update: true,
  auto_start_unfinished: true,
})
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
      form.value.default_download_dir = folder as string
    }
  } catch (e) {
    console.error('选择目录失败:', e)
  }
}
// 显示用的限速值（KB/s），0 表示不限速
const speedLimitKB = computed({
  get: () => form.value.max_download_limit > 0 ? Math.round(form.value.max_download_limit / 1024) : 0,
  set: (val: number) => { form.value.max_download_limit = val > 0 ? val * 1024 : 0 }
})

async function loadSettings() {
  loading.value = true
  try {
    const settings = await GetSettings()
    if (settings) {
      Object.assign(form.value, settings)
    }
    appVersion.value = await GetAppVersion()
  } catch (e) {
    console.error('加载设置失败:', e)
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await SaveSettings(form.value)
    message.success('设置已保存')
  } catch (e: any) {
    message.error('保存失败: ' + e)
  } finally {
    saving.value = false
  }
}

/** 手动检查更新（用户点击按钮触发） */
async function handleCheckUpdate() {
  await updateStore.manualCheck()
  showUpdateResult()
}

/** 根据缓存的更新检查结果显示提示 */
function showUpdateResult() {
  const r = updateStore.result
  if (!r) return
  if (r.error) {
    message.error('检查更新失败: ' + r.error)
  } else if (r.has_update) {
    message.info(`发现新版本 v${r.latest_version}，当前版本 v${r.current_version}`, { duration: 8000 })
  } else {
    message.success('已是最新版本')
  }
}

// 监听后端启动时自动发送的更新检查结果
watch(() => updateStore.result, (r) => {
  if (r && form.value.auto_check_update) {
    showUpdateResult()
  }
})

onMounted(async () => {
  await loadSettings()
})
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2 class="page-title">设置</h2>
    </div>
    <div class="settings-area">
      <n-spin :show="loading">
        <n-form label-placement="left" label-width="160" require-mark-placement="right-hanging"
          :style="{ maxWidth: '640px' }">

          <!-- 基本设置 -->
          <n-card title="基本设置" size="small" :bordered="false" style="margin-bottom: 16px">
            <n-form-item label="默认下载目录">
              <n-input v-model:value="form.default_download_dir" placeholder="默认下载目录" :disabled="loading">
                <template #prefix>
                  <n-icon :component="FolderOpenOutline" />
                </template>
                <template #suffix>
                  <n-button size="tiny" :disabled="loading" @click="selectFolder">浏览</n-button>
                </template>
              </n-input>
            </n-form-item>
          </n-card>

          <!-- 下载设置 -->
          <n-card title="下载设置" size="small" :bordered="false" style="margin-bottom: 16px">
            <n-form-item label="最大同时下载数">
              <n-input-number v-model:value="form.max_concurrent_downloads" :min="1" :max="20" style="width: 160px" />
              <span class="hint">1-20</span>
            </n-form-item>
            <n-form-item label="最大连接数/服务器">
              <n-input-number v-model:value="form.max_connection_per_server" :min="1" :max="32" style="width: 160px" />
              <span class="hint">1-32</span>
            </n-form-item>
            <n-form-item label="分片数">
              <n-input-number v-model:value="form.split" :min="1" :max="32" style="width: 160px" />
              <span class="hint">1-32，值越大占用内存越多</span>
            </n-form-item>
            <n-form-item label="断点续传">
              <n-switch v-model:value="form.continue_download" />
              <span class="hint">应用重启后继续未完成的下载</span>
            </n-form-item>
            <n-form-item label="启动时自动开始任务">
              <n-switch v-model:value="form.auto_start_unfinished" />
              <span class="hint">应用启动时所有任务暂停</span>
            </n-form-item>
          </n-card>

          <!-- 文件设置 -->
          <n-card title="文件设置" size="small" :bordered="false" style="margin-bottom: 16px">
            <n-form-item label="允许覆盖已有文件">
              <n-switch v-model:value="form.allow_overwrite" />
              <span class="hint">下载时如果文件已存在则覆盖</span>
            </n-form-item>
            <n-form-item label="自动重命名">
              <n-switch v-model:value="form.auto_file_renaming" />
              <span class="hint">文件存在时自动添加序号后缀</span>
            </n-form-item>
          </n-card>

          <!-- 限速设置 -->
          <n-card title="限速设置" size="small" :bordered="false" style="margin-bottom: 16px">
            <n-form-item label="全局下载限速">
              <n-input-number v-model:value="speedLimitKB" :min="0" :step="100" style="width: 160px" placeholder="0" />
              <span class="hint">KB/s，0 表示不限速</span>
            </n-form-item>
          </n-card>

          <!-- 更新设置 -->
          <n-card title="更新" size="small" :bordered="false" style="margin-bottom: 16px">
            <n-form-item label="当前版本">
              <n-tag type="info" size="small">v{{ appVersion }}</n-tag>
            </n-form-item>
            <n-form-item label="自动检查更新">
              <n-switch v-model:value="form.auto_check_update" />
              <span class="hint">启动时自动检查是否有新版本</span>
            </n-form-item>
            <n-form-item label="手动检查">
              <n-button size="small" :loading="updateStore.checking" @click="handleCheckUpdate">检查更新</n-button>
            </n-form-item>
          </n-card>

          <div style="padding-left: 160px; margin-top: 8px">
            <n-button type="primary" :loading="saving" @click="handleSave">保存设置</n-button>
          </div>
        </n-form>
      </n-spin>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: 16px 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  margin-bottom: 12px;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.settings-area {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.hint {
  margin-left: 8px;
  font-size: 12px;
  color: #999;
}
</style>
