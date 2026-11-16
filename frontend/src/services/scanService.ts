// frontend/src/services/scanService.ts
// 本地扫描（游戏库「扫描添加」）的前端封装。
// 对应后端 internal/service/scan（Wails 服务名 ScanService）与
// internal/service/dialog_service.go（DialogService，系统文件夹选择框）。
//
// 扫描结果除了名字，还要带上「是怎么扫到的」：
//   - origin='exe'    → 按 exe 名扫到，path 就是这个 exe 的绝对路径（入库直接用它）；
//   - origin='folder' → 按文件夹名扫到（Steam 库 / 本地文件夹），folderPath 是文件夹绝对路径，
//                       入库时由后端进文件夹挑主程序（排除 UnityCrashHandler64、优先 Game.exe）。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { ScanService } from '../../bindings/WeedyBox/internal/service/scan'
import type { GameScanResult, Subfolder } from '../../bindings/WeedyBox/internal/service/scan'
import { DialogService } from '../../bindings/WeedyBox/internal/service'

export type { GameScanResult, Subfolder }

/** 扫描出来的一条候选：name 喂给搜索，detail 说明它从哪来，origin/path 供入库记录 exe */
export interface ScannedGame {
  name: string
  detail: string
  /** 来源：exe = 按 exe 文件名扫到；folder = 按文件夹名扫到 */
  origin: 'exe' | 'folder'
  /** exe 绝对路径（origin='exe' 时有） */
  path?: string
  /** 游戏文件夹绝对路径（origin='folder' 时有） */
  folderPath?: string
}

export const scanApi = {
  /** Steam：本机所有库 steamapps\common 的一级子文件夹（名字 + 绝对路径，已去重、已排序） */
  async steamFolders(): Promise<Subfolder[]> {
    return (await ScanService.ScanSteamGameFolders()) ?? []
  },

  /** 本地目录：每个一级子文件夹里的 .exe（不递归、不含根目录自己的 exe） */
  async executables(root: string): Promise<GameScanResult> {
    const result = await ScanService.ScanExecutablesInSubfolders(root)
    if (!result) throw new Error('扫描没有返回结果')
    return result as GameScanResult
  },

  /** 本地目录：只读一层，返回一级子文件夹（名字 + 绝对路径），适合已规范命名的库 */
  async subfolders(root: string): Promise<Subfolder[]> {
    return (await ScanService.ScanSubfolders(root)) ?? []
  },

  /** 系统「选择文件夹」对话框；用户取消返回空串（不是错误） */
  async pickDirectory(title = '选择游戏文件夹'): Promise<string> {
    return (await DialogService.PickDirectory(title)) ?? ''
  },
}
