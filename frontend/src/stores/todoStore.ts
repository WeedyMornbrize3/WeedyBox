// frontend/src/stores/todoStore.ts

import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { todoApi } from '../services/todoService'
import type { Todo, TodoStats, UpdateTodoDTO } from '../../bindings/WeedyBox/internal/model'

// ===== 筛选类型 =====
// 'pending' = 只看未完成（默认），'completed' = 只看已完成，'all' = 全部
export type CompletedFilter = 'pending' | 'completed' | 'all'
// null = 全部优先级，0 低 / 1 中 / 2 高
export type PriorityFilter = number | null
// 按创建时间过滤：'all' 不限 / 'today' 今天 / 'week' 近7天 / 'month' 近30天
export type TimeFilter = 'all' | 'today' | 'week' | 'month'

export const COMPLETED_OPTIONS: { value: CompletedFilter; label: string }[] = [
    { value: 'pending', label: '未完成' },
    { value: 'completed', label: '已完成' },
    { value: 'all', label: '全部' },
]

export const PRIORITY_OPTIONS: { value: PriorityFilter; label: string }[] = [
    { value: null, label: '全部' },
    { value: 2, label: '高' },
    { value: 1, label: '中' },
    { value: 0, label: '低' },
]

export const TIME_OPTIONS: { value: TimeFilter; label: string }[] = [
    { value: 'all', label: '全部' },
    { value: 'today', label: '今天' },
    { value: 'week', label: '近7天' },
    { value: 'month', label: '近30天' },
]

export const TIME_RANGE_DAYS: Record<Exclude<TimeFilter, 'all'>, number> = {
    today: 0,
    week: 7,
    month: 30,
}

export const useTodoStore = defineStore('todo', () => {
    // ===== State =====
    const todos = ref<Todo[]>([])
    const loading = ref(false)
    const error = ref<string | null>(null)
    const stats = ref<TodoStats>({ total: 0, completed: 0, pending: 0 })
    const curPriority = ref<number>(1)

    // 筛选条件：默认只看未完成
    const filters = reactive<{
        completed: CompletedFilter
        priority: PriorityFilter
        time: TimeFilter
    }>({
        completed: 'pending',
        priority: null,
        time: 'all',
    })

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

    // 时间是否落在筛选范围内
    function matchTime(createdAt: string, time: TimeFilter): boolean {
        if (time === 'all') return true
        const diff = Date.now() - new Date(createdAt).getTime()
        if (Number.isNaN(diff)) return true
        const days = TIME_RANGE_DAYS[time]
        // diff 为负（时间在未来）时按 0 处理；'today' 为 0 表示仅今天
        const elapsedDays = Math.max(0, diff) / (24 * 60 * 60 * 1000)
        if (days === 0) return elapsedDays < 1
        return elapsedDays <= days
    }

    // 应用筛选后的列表（模板渲染用这个，不再直接用 todos）
    const filteredTodos = computed(() =>
        todos.value.filter(t => {
            if (filters.completed === 'pending' && t.completed) return false
            if (filters.completed === 'completed' && !t.completed) return false
            if (filters.priority !== null && t.priority !== filters.priority) return false
            if (!matchTime(t.createdAt, filters.time)) return false
            return true
        })
    )

    // 是否处于非默认筛选状态（用于空状态文案）
    const isFilterActive = computed(() =>
        filters.completed !== 'pending' || filters.priority !== null || filters.time !== 'all'
    )

    // 筛选后为空、但原始数据不为空 —— 说明是筛选条件把内容挡住了
    const isFilteredEmpty = computed(() =>
        !loading.value && todos.value.length > 0 && filteredTodos.value.length === 0
    )

    function resetFilters() {
        filters.completed = 'pending'
        filters.priority = null
        filters.time = 'all'
    }

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

    /**
     * 批量删除（供「删除指定范围」使用）。
     * 后端一次 SQL 删除（DeleteByIDs），这里对列表做乐观更新：
     * 先从本地移除，失败时按原索引回滚，避免用户看到明显延迟。
     */
    async function deleteMany(ids: number[]) {
        const unique = [...new Set(ids)]
        if (unique.length === 0) return 0

        // 记录原始位置，失败时原样放回
        const removed: { index: number; todo: Todo }[] = []
        const target = new Set(unique)
        todos.value.forEach((todo, index) => {
            if (target.has(todo.id)) removed.push({ index, todo })
        })

        const snapshot = todos.value
        todos.value = todos.value.filter(t => !target.has(t.id))

        try {
            await todoApi.deleteMany(unique)
            await loadStats()
            return removed.length
        } catch (e) {
            // 回滚：按原索引升序插回，恢复原有顺序
            const restored = [...snapshot]
            for (const { index, todo } of [...removed].sort((a, b) => a.index - b.index)) {
                restored.splice(Math.min(index, restored.length), 0, todo)
            }
            todos.value = restored
            error.value = e instanceof Error ? e.message : '批量删除失败'
            console.error('批量删除 TODO 失败:', e)
            throw e
        }
    }

    /**
     * 批量标记完成（供「完成全部」使用）。
     * 复用已有的 Update 绑定（它返回更新后的 Todo），逐个把返回结果就地替换，
     * 这样后端若对 updated_at 等字段做了处理也能如实反映。
     * 未选中的条目不受影响。
     */
    async function completeMany(ids: number[]) {
        const unique = [...new Set(ids)]
        if (unique.length === 0) return 0

        // 只处理确实存在且尚未完成的条目，避免无谓请求
        const pending = todos.value.filter(t => unique.includes(t.id) && !t.completed)
        if (pending.length === 0) return 0

        try {
            for (const todo of pending) {
                const updated = await todoApi.update(todo.id, { completed: true } as UpdateTodoDTO)
                const index = todos.value.findIndex(t => t.id === todo.id)
                if (index !== -1) todos.value[index] = updated
            }
            await loadStats()
            return pending.length
        } catch (e) {
            error.value = e instanceof Error ? e.message : '批量完成失败'
            console.error('批量完成 TODO 失败:', e)
            // 已成功的部分保留（后端已是事实），仅重新拉取以对齐真实状态
            await loadTodos()
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
        filters,
        // Getters
        completedTodos,
        pendingTodos,
        highPriorityTodos,
        completionRate,
        filteredTodos,
        isFilterActive,
        isFilteredEmpty,
        // Actions
        loadTodos,
        addTodo,
        toggleComplete,
        updateTodo,
        deleteTodo,
        deleteMany,
        completeMany,
        loadStats,
        init,
        resetFilters,
    }
})