<!-- frontend/src/components/todo/TodoList.vue -->
<template>
  <div class="todo-container select-none h-full min-h-0 flex flex-col">
    <!-- 标题 -->
    <h2 class="text-xl font-semibold text-primary mb-4 flex-shrink-0">📋 TODO 列表</h2>

    <!-- 统计 -->
    <div class="flex flex-shrink-0">
      <TodoStats :stats="todoStore.stats" />
      <TodoPrioritySelector class="ml-auto">
      </TodoPrioritySelector>
    </div>

    <!-- 输入框（描述框展开时高度自增，下方卡片自动让位） -->
    <TodoInput class="mt-4 flex-shrink-0" />

    <!-- 筛选栏  不添加margin-top -->
    <FilterBar class="flex-shrink-0" />  

    <!-- 列表卡片：flex-1 吃掉剩余高度并内部滚动 -->
    <!-- 不要用 max-h-[calc(100vh-Npx)] 固定扣减：那个 N 算不准（实测溢出 27px），
         且描述框一展开卡片又被下推，导致最后一条滚不到底。交给 flex 自适应即无需任何魔法数字。 -->
    <div class="card mt-3 mb-6 p-2 flex-1 min-h-[120px] overflow-y-auto scrollbar-theme">
      <!-- 加载状态 -->
      <div v-if="todoStore.loading" class="text-center py-8 text-tertiary">
        <span class="inline-block animate-spin mr-2">⏳</span>
        加载中...
      </div>

      <!-- 错误 -->
      <div v-else-if="todoStore.error" class="text-center py-8 text-danger">
        ❌ {{ todoStore.error }}
      </div>

      <!-- 空状态：本来就没有数据 -->
      <div v-else-if="todoStore.todos.length === 0" class="text-center py-8 text-tertiary">
        🎉 还没有 TODO，添加一个吧！
      </div>

      <!-- 空状态：被筛选条件挡住 -->
      <div v-else-if="todoStore.isFilteredEmpty" class="text-center py-8 text-tertiary">
        <p class="mb-3">🔍 当前筛选条件下没有匹配的 TODO</p>
        <button type="button" class="chip" @click="todoStore.resetFilters()">
          ↺ 恢复默认筛选
        </button>
      </div>

      <!-- TODO 列表 -->
      <div v-else class="space-y-2 p-2">
        <TodoItem
          v-for="todo in todoStore.filteredTodos"
          :key="todo.id"
          :todo="todo"
          @toggle="handleToggle"
          @delete="handleDelete"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useTodoStore } from '../../stores/todoStore'
import TodoInput from './TodoInput.vue'
import TodoItem from './TodoItem.vue'
import TodoStats from './TodoStats.vue'
import TodoPrioritySelector from './TodoPrioritySelector.vue'
import FilterBar from './FilterBar.vue'

const todoStore = useTodoStore()

const handleToggle = (id: number) => {
  todoStore.toggleComplete(id)
}

const handleDelete = (id: number) => {
  if (confirm('确定要删除这个 TODO 吗？')) {
    todoStore.deleteTodo(id)
  }
}

onMounted(() => {
  todoStore.init()
})
</script>