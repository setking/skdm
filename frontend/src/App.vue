<script setup lang="ts" xmlns:src="http://www.w3.org/1999/xhtml">
import { lightTheme } from 'naive-ui'
</script>
<template>
  <n-config-provider :theme="lightTheme">
    <n-global-style />
    <n-message-provider>
      <RouterView />
    </n-message-provider>
  </n-config-provider>
</template>
