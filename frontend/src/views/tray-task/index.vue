<script setup lang="ts">
import { Window } from '@wailsio/runtime'
import { lightTheme } from 'naive-ui'
import { ref, onMounted } from 'vue'
import DownloadDialog from '@/components/download-dialog/index.vue'

const show = ref(false)
onMounted(() => { show.value = true })

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
      <n-dialog-provider>
        <DownloadDialog :show="show" @update:show="onUpdateShow" />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
