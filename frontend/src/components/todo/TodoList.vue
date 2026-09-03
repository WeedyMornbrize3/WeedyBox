<!-- frontend/src/components/todo/TodoList.vue -->
<template>
  <div class="todo-container select-none">
    <!-- 标题 -->
    <h2 class="text-xl font-semibold text-primary mb-4">📋 TODO 列表</h2>

    <!-- 统计 -->
    <TodoStats :stats="todoStore.stats" />

    <!-- 输入框 -->
    <TodoInput class="mt-4" />

    <!-- 加载状态 -->
    <div v-if="todoStore.loading" class="text-center py-8 text-tertiary">
      <span class="inline-block animate-spin mr-2">⏳</span>
      加载中...
    </div>

    <!-- 错误 -->
    <div v-else-if="todoStore.error" class="text-center py-8 text-danger">
      ❌ {{ todoStore.error }}
    </div>

    <!-- 空状态 -->
    <div v-else-if="todoStore.todos.length === 0" class="text-center py-8 text-tertiary">
      🎉 还没有 TODO，添加一个吧！
    </div>

    <!-- TODO 列表 -->
    <div v-else class="space-y-2 mt-4">
      <TodoItem
        v-for="todo in todoStore.todos"
        :key="todo.id"
        :todo="todo"
        @toggle="handleToggle"
        @delete="handleDelete"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useTodoStore } from '../../stores/todoStore'
import TodoInput from './TodoInput.vue'
import TodoItem from './TodoItem.vue'
import TodoStats from './TodoStats.vue'

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