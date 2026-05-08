<template>
  <n-menu v-model:value="allTask" :options="menuOptions" @update:value="handleUpdateValue" />
</template>

<script setup lang="ts">
import type { MenuOption } from 'naive-ui'
import type { Component } from 'vue'
import { ref } from 'vue'
import {
  AlertCircleSharp as AlertCircleIcon,
  SettingsSharp as SettingsIcon,
  TrashSharp as TrashIcon,
  CaretForwardSharp as CaretForwardIcon,
  CheckmarkCircleSharp as CheckmarkCircleIcon,
} from '@vicons/ionicons5'
import { NIcon } from 'naive-ui'
import { h } from 'vue'
import { RouterLink } from 'vue-router'
const allTask = ref('run-task')
function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}


function handleUpdateValue(key: string, item: MenuOption) {
  console.log(item, key)
}

const menuOptions: MenuOption[] = [
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: 'run',
          }
        },
        { default: () => '下载中' }
      ),
    key: 'run-task',
    icon: renderIcon(CaretForwardIcon)
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: 'unfinished',
          }
        },
        { default: () => '未完成' }
      ),
    key: 'unfinished-task',
    icon: renderIcon(AlertCircleIcon)
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: 'complete',
          }
        },
        { default: () => '已完成' }
      ),
    key: 'complete-task',
    icon: renderIcon(CheckmarkCircleIcon)
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: 'trash',
          }
        },
        { default: () => '回收站' }
      ),
    key: 'trash-task',
    icon: renderIcon(TrashIcon)
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: 'settings',
          }
        },
        { default: () => '设置' }
      ),
    key: 'settings',
    icon: renderIcon(SettingsIcon)
  },
]


</script>
