<script setup lang="ts" xmlns:src="http://www.w3.org/1999/xhtml">
import { lightTheme } from 'naive-ui'
</script>
<template>
  <n-config-provider :theme="lightTheme">
    <n-global-style />
      <n-space vertical size="large">
        <n-layout>
          <RouterView />
        </n-layout>
      </n-space>
  </n-config-provider>
</template>
<style lang="scss" scoped>
.n-space{
  width: 100%;
  height: 100%;
}
</style>
