<script setup lang="ts">
import { Window } from '@wailsio/runtime'
import { lightTheme } from 'naive-ui'
import { ref } from 'vue'
import DownloadDialog from '@/components/download-dialog/index.vue'

const show = ref(true)

function onUpdateShow(val: boolean) {
  if (!val) {
    // Dialog closed (submitted or cancelled): close the popup window
    Window.Close()
  }
}
</script>

<template>
  <n-config-provider :theme="lightTheme">
    <n-global-style />
    <n-message-provider>
      <DownloadDialog :show="show" @update:show="onUpdateShow" />
    </n-message-provider>
  </n-config-provider>
</template>
