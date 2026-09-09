// frontend/src/stores/todoStore.ts

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { todoApi } from '../services/todoService'
import type { Todo, TodoStats } from '../../bindings/WeedyBox/internal/model'

export const useTodoStore = defineStore('todo', () => {
    // ===== State =====
    const todos = ref<Todo[]>([])
    const loading = ref(false)
    const error = ref<string | null>(null)
    const stats = ref<TodoStats>({ total: 0, completed: 0, pending: 0 })
    const curPriority = ref<number>(1)

    // ===== Getters =====
    const completedTodos = computed(() => 
        todos.value.filter(t => t.completed)
    )
    
    const pendingTodos = computed(() => 
        todos.value.filter(t => !t.completed)
    )
 
    const highPriorityTodos = computed(() =>
        todos.value.filter(t => t.priority === 2)
    )

    const completionRate = computed(() => {
        if (stats.value.total === 0) return 0
        return Math.round((stats.value.completed / stats.value.total) * 100)
    })

    // ===== Actions =====
    async function loadTodos() {
        loading.value = true
        error.value = null
        try {
            const [todosResult, statsResult] = await Promise.all([
                todoApi.getAll(),
                todoApi.getStats()
            ])
            todos.value = todosResult
            stats.value = statsResult
        } catch (e) {
            error.value = e instanceof Error ? e.message : '加载失败'
            console.error('加载 TODO 失败:', e)
        } finally {
            loading.value = false
        }
    }

    async function addTodo(title: string, description?: string) {
        try {
            const todo = await todoApi.create({
                title: title.trim(), 
                description: description?.trim() || undefined, 
                priority: curPriority.value
            })
            todos.value.unshift(todo)
            await loadStats()
            return todo
        } catch (e) {
            error.value = e instanceof Error ? e.message : '创建失败'
            console.error('创建 TODO 失败:', e)
            throw e
        }
    }

    async function toggleComplete(id: number) {
        try {
            const todo = await todoApi.toggleComplete(id)
            const index = todos.value.findIndex(t => t.id === id)
            if (index !== -1) {
                todos.value[index] = todo
            }
            await loadStats()
            return todo
        } catch (e) {
            error.value = e instanceof Error ? e.message : '切换状态失败'
            console.error('切换状态失败:', e)
            throw e
        }
    }

    async function updateTodo(id: number, updates: Partial<Todo>) {
        try {
            const todo = await todoApi.update(id, updates)
            const index = todos.value.findIndex(t => t.id === id)
            if (index !== -1) {
                todos.value[index] = todo
            }
            return todo
        } catch (e) {
            error.value = e instanceof Error ? e.message : '更新失败'
            console.error('更新 TODO 失败:', e)
            throw e
        }
    }

    async function deleteTodo(id: number) {
        try {
            await todoApi.delete(id)
            todos.value = todos.value.filter(t => t.id !== id)
            await loadStats()
        } catch (e) {
            error.value = e instanceof Error ? e.message : '删除失败'
            console.error('删除 TODO 失败:', e)
            throw e
        }
    }

    async function loadStats() {
        try {
            stats.value = await todoApi.getStats()
        } catch (e) {
            console.error('加载统计失败:', e)
        }
    }

    function init() {
        loadTodos()
    }

    return {
        // State
        todos,
        loading,
        error,
        stats,
        curPriority,
        // Getters
        completedTodos,
        pendingTodos,
        highPriorityTodos,
        completionRate,
        // Actions
        loadTodos,
        addTodo,
        toggleComplete,
        updateTodo,
        deleteTodo,
        loadStats,
        init,
    }
})