<template>
  <div class="flex gap-1 bg-secondary p-[3px] rounded-[10px] border border-theme">
    <button
            v-for="option in priorityOptions"
            :key="option.value"
            class="flex items-center gap-1 px-3 py-1.25 border-none rounded-lg cursor-pointer bg-transparent text-secondary text-xs transition-all duration-200 whitespace-nowrap hover:text-primary hover:bg-hover"
            :class="{ 
                '!bg-card !text-primary shadow-[0_1px_4px_var(--color-shadow)]': todoStore.curPriority === option.value
            }"
            @click="todoStore.curPriority = option.value"
            :title="option.label"
    >
      <span class="font-medium">{{ option.label }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()

const priorityOptions = [
  { value: 0 as const, label: '1' },
  { value: 1 as const, label: '2' },
  { value: 2 as const, label: '3' },
]

</script>