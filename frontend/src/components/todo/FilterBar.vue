<!-- frontend/src/components/todo/FilterBar.vue -->
<!-- 筛选栏：优先级 / 创建时间（「完成状态」筛选已随完成功能移除）。
     横向排列，放不下时横向滚动（scrollbar-theme 定义在 global.css，不能写进 UnoCSS shortcut）。 -->
<template>
  <div class="filter-bar">
    <span class="i-lucide-list-filter icon-sm text-tertiary flex-shrink-0" aria-hidden="true" />

    <FilterChipGroup
      v-model="filters.priority"
      label="优先级"
      :options="PRIORITY_OPTIONS"
    />

    <FilterChipGroup
      v-model="filters.time"
      label="创建时间"
      :options="TIME_OPTIONS"
    />

    <!-- 计数 + 重置：横向滚动时贴在最右侧 -->
    <div class="ml-auto flex items-center gap-2 flex-shrink-0">
      <!-- 计数变化用 aria-live 播报（视觉上就是普通文本，不加动效噪音） -->
      <span class="text-xs text-tertiary whitespace-nowrap tabular-nums" aria-live="polite">
        已筛选 {{ todoStore.filteredTodos.length }} / {{ todoStore.todos.length }} 条
      </span>
      <n-button
        v-if="todoStore.isFilterActive"
        quaternary
        size="tiny"
        title="恢复默认筛选"
        @click="todoStore.resetFilters()"
      >
        <template #icon>
          <span class="i-lucide-rotate-ccw icon-xs" aria-hidden="true" />
        </template>
        重置
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { NButton } from 'naive-ui'
import { useTodoStore, PRIORITY_OPTIONS, TIME_OPTIONS } from '../../stores/todoStore'
import FilterChipGroup from './FilterChipGroup.vue'

const todoStore = useTodoStore()
// storeToRefs 保持 filters 的响应性（直接解构会丢失）
const { filters } = storeToRefs(todoStore)
</script>
