// frontend/src/services/todoService.ts
// TODO 的前端封装。2026-11-23 起「完成」功能已移除：
// 不再有 toggleComplete / getStats（后端接口仍在，只是前端不用了）。

import { TodoService } from '../../bindings/WeedyBox/internal/service'
import type { 
    Todo, 
    CreateTodoDTO, 
    UpdateTodoDTO 
} from '../../bindings/WeedyBox/internal/model'

// 辅助函数：确保返回值不为 null
function ensure<T>(value: T | null, message: string): T {
    if (value === null || value === undefined) {
        throw new Error(message)
    }
    return value
}

export const todoApi = {
    async getAll(): Promise<Todo[]> {
        return (await TodoService.GetAll()) ?? []
    },

    async getByID(id: number): Promise<Todo> {
        return ensure(await TodoService.GetByID(id), `TODO ${id} 不存在`)
    },

    async create(dto: CreateTodoDTO): Promise<Todo> {
        return ensure(await TodoService.Create(dto), '创建 TODO 失败')
    },

    async update(id: number, dto: UpdateTodoDTO): Promise<Todo> {
        return ensure(await TodoService.Update(id, dto), `更新 TODO ${id} 失败`)
    },

    async delete(id: number): Promise<void> {
        await TodoService.Delete(id)
    },

    /** 批量删除（一次 SQL）。后端对空数组直接返回，这里也做一次短路。 */
    async deleteMany(ids: number[]): Promise<void> {
        if (ids.length === 0) return
        await TodoService.DeleteByIDs(ids)
    },
}