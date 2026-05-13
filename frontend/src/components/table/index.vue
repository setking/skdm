<script setup lang="ts">
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'

const props = defineProps<{
  title: string
  count: number
  columns: DataTableColumns<Record<string, unknown>>
  data: Record<string, unknown>[]
  checkedRowKeys?: DataTableRowKey[]
  rowKey?: string
}>()

const emit = defineEmits<{
  'update:checkedRowKeys': [keys: DataTableRowKey[]]
}>()

function onCheckedKeys(keys: DataTableRowKey[]) {
  emit('update:checkedRowKeys', keys)
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2 class="page-title">{{ title }}</h2>
      <div class="header-right">
        <slot name="header-right">
          <span class="count">{{ count }} 个任务</span>
        </slot>
      </div>
    </div>
    <slot name="batch-actions" />
    <div class="table-area">
      <n-data-table
        :columns="columns"
        :data="data"
        :bordered="false"
        striped
        size="small"
        flex-height
        style="height: 100%; width: 100%"
        :row-key="(row: Record<string, unknown>) => row[props.rowKey || 'gid'] as DataTableRowKey"
        :checked-row-keys="checkedRowKeys"
        :on-update:checked-row-keys="onCheckedKeys"
      />
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px 24px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; margin-bottom: 12px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.count { color: #999; font-size: 13px; }
.table-area { flex: 1; min-height: 0; min-width: 0; position: relative; }
:deep(.n-data-table) { width: 100% !important; }
.empty-overlay { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; z-index: 1; pointer-events: none; }
</style>
