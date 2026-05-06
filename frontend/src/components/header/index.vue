<script setup lang="ts">
import { Window } from '@wailsio/runtime'
import { Add, Close, Remove, ExpandOutline, ContractOutline } from '@vicons/ionicons5'
import { ref } from 'vue'
import DownloadDialog from '@/components/download-dialog/index.vue'

const isMaximised = ref(false)
const showDownloadDialog = ref(false)

async function handleToggleMaximise() {
  try {
    isMaximised.value = await Window.IsMaximised()
    if (isMaximised.value) {
      Window.Unmaximise()
    } else {
      Window.Maximise()
    }
    isMaximised.value = !isMaximised.value
  } catch {
    // fallback: toggle directly
    Window.ToggleMaximise()
  }
}

function handleMinimise() {
  Window.Minimise()
}

function handleClose() {
  Window.Close()
}

function handleNewTask() {
  showDownloadDialog.value = true
}
</script>

<template>
  <div class="title-bar">
  <n-space vertical size="large">
    <n-layout has-sider>
      <n-layout-sider content-style="padding: 12px;">
        <div class="title-bar-left">
          <span class="app-title">skdm</span>
        </div>
      </n-layout-sider>
      <n-layout-content content-style="padding: 12px;">
        <n-flex justify="end">
          <div class="title-bar-actions">
            <n-button class="task-btn" size="tiny" @click="handleNewTask">
              <n-icon :component="Add" />
              <span>新建</span>
            </n-button>
            <n-button class="win-btn" size="tiny" @click="handleMinimise">
              <n-icon :component="Remove" />
            </n-button>
            <n-button class="win-btn" size="tiny" @click="handleToggleMaximise">
              <n-icon :component="isMaximised ? ContractOutline : ExpandOutline" />
            </n-button>
            <n-button class="win-btn win-btn-close" size="tiny" @click="handleClose">
              <n-icon :component="Close" />
            </n-button>
          </div>
        </n-flex>
      </n-layout-content>
    </n-layout>
  </n-space>
  </div>
  <DownloadDialog v-model:show="showDownloadDialog" />
</template>

<style lang="scss" scoped>
.title-bar {
  user-select: none;
  -webkit-app-region: drag;
  --wails-draggable: drag;
}

.title-bar-left {
  display: flex;
  align-items: center;
  gap: 8px;

  .app-icon {
    width: 22px;
    height: 22px;
  }

  .app-title {
    font-size: 14px;
    font-weight: 500;
  }
}

.title-bar-center {
  display: flex;
  align-items: center;
  -webkit-app-region: no-drag;
}

.title-bar-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  -webkit-app-region: no-drag;
}

.task-btn {
  width: 50px;
  height: 28px;
  --wails-draggable: no-drag;
}

.win-btn {
  width: 36px;
  height: 28px;
  --wails-draggable: no-drag;
}

.win-btn-close:hover {
  background: #e81123 !important;
  color: #fff !important;
}
</style>
