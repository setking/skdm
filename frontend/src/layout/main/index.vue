<script setup lang="ts">
import UpdateDialog from '@/components/update-dialog/index.vue'
</script>

<template>
<UpdateDialog />
<keep-alive>
  <router-view></router-view>
</keep-alive>
</template>
