// frontend/src/services/scanService.ts
// 本地扫描（游戏库「扫描添加」）的前端封装。
// 对应后端 internal/service/scan（Wails 服务名 ScanService）与
// internal/service/dialog_service.go（DialogService，系统文件夹选择框）。
//
// 这里只回答"本机有哪些游戏名"：扫 Steam 库 / 扫本地游戏文件夹，把结果整理成
// `{ name, detail }` 列表交给 store。搜索、入库仍走 metaStore / libraryApi ——
// 扫描器不联网，也不写库。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { ScanService } from '../../bindings/WeedyBox/internal/service/scan'
import type { GameScanResult } from '../../bindings/WeedyBox/internal/service/scan'
import { DialogService } from '../../bindings/WeedyBox/internal/service'

export type { GameScanResult }

/** 扫描出来的一条候选：name 喂给搜索，detail 说明它从哪来 */
export interface ScannedGame {
  name: string
  detail: string
}

export const scanApi = {
  /** Steam：本机所有库的 steamapps\common 一级子文件夹名（后端已去重、已排序） */
  async steamGames(): Promise<string[]> {
    return (await ScanService.ScanSteamGameFolders()) ?? []
  },

  /** 本地目录：每个一级子文件夹里的 .exe（不递归、不含根目录自己的 exe） */
  async executables(root: string): Promise<GameScanResult> {
    const result = await ScanService.ScanExecutablesInSubfolders(root)
    if (!result) throw new Error('扫描没有返回结果')
    return result as GameScanResult
  },

  /** 本地目录：只读一层，返回一级子文件夹名（适合已规范命名的库） */
  async subfolderNames(root: string): Promise<string[]> {
    return (await ScanService.ScanSubfolderNames(root)) ?? []
  },

  /** 系统「选择文件夹」对话框；用户取消返回空串（不是错误） */
  async pickDirectory(title = '选择游戏文件夹'): Promise<string> {
    return (await DialogService.PickDirectory(title)) ?? ''
  },
}
