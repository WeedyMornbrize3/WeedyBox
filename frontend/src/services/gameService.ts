// frontend/src/services/gameService.ts
// 游戏启动的前端封装：对应后端 internal/service/game（Wails 服务名 GameService）。
//
// 一件事：把「游戏库里的作品」和「本机上的游戏程序 / Steam 客户端」接起来。
// 位置就是 library_works 表上的一列 game_path（与元数据同表，重新抓取/重新入库
// 不会把它冲掉），随 Fetch/Store 的返回一起带到前端（LibraryMeta.gamePath）。
//
// 启动方式（2026-11-23）：library_works.launch_mode。Steam 源默认 steam ——
// 跑 steam://rungameid/<appid> 交给 Steam 客户端；也可切成本地程序（game_path）。
//
// 四个动作：
//   status()        读位置 + exe 是否还在 + 是否正在运行 + 生效的启动方式（进入页面时调一次，之后轮询）
//   setPath()       记录/更换位置（第一次启动没有位置、或用户主动换程序时）
//   setLaunchMode() 切换「通过 Steam / 本地程序」启动方式
//   launch()        按生效方式启动；exe 方式下位置缺失/文件被移动时返回 missing=true，
//                   前端据此弹系统文件框让用户选一次 exe，而不是当成启动失败
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { GameService } from '../../bindings/WeedyBox/internal/service/game'
import { DialogService } from '../../bindings/WeedyBox/internal/service'
import type {
  GameLaunchModeState,
  GameLaunchResult,
  GamePathState,
  GameStatus,
} from '../../bindings/WeedyBox/internal/service/game'
import { LIBRARY_SOURCE_DLSITE } from './libraryService'

export type { GameLaunchModeState, GameLaunchResult, GamePathState, GameStatus }

/** 启动方式：空串 = 还没选过（后端按来源取默认值） */
export type LibraryLaunchMode = '' | 'steam' | 'exe'

export const gameApi = {
  /**
   * 读游戏状态：位置、exe 是否还存在、是否正在运行。
   * 后端会用运行缓存加速，并在游戏退出/文件被删时把缓存清掉。
   */
  async status(workId: string, source = LIBRARY_SOURCE_DLSITE): Promise<GameStatus> {
    const result = await GameService.Status(workId, source)
    if (!result) throw new Error('读取游戏状态失败：后端未返回结果')
    return result
  },

  /**
   * 记录/更换游戏程序位置。
   * 后端会校验文件确实存在且不是文件夹，并把路径归一化成绝对路径。
   */
  async setPath(workId: string, path: string, source = LIBRARY_SOURCE_DLSITE): Promise<GamePathState> {
    const result = await GameService.SetPath({ workId, path, source })
    if (!result) throw new Error('记录游戏位置失败：后端未返回结果')
    return result
  },

  /**
   * 切换启动方式（只有 Steam 源能选 steam）。
   * 返回的 mode 是**生效值**（空串已按来源解析），前端据此更新 chip 选中态。
   */
  async setLaunchMode(
    workId: string,
    mode: LibraryLaunchMode,
    source = LIBRARY_SOURCE_DLSITE,
  ): Promise<GameLaunchModeState> {
    const result = await GameService.SetLaunchMode({ workId, mode, source })
    if (!result) throw new Error('切换启动方式失败：后端未返回结果')
    return result
  },

  /**
   * 用库里记录的启动方式启动游戏。
   * 返回 missing=true 表示「没有位置 / 文件已不在」——调用方应让用户选一次程序；
   * 真正的启动失败（进程起不来）会走 reject。
   */
  async launch(workId: string, source = LIBRARY_SOURCE_DLSITE): Promise<GameLaunchResult> {
    const result = await GameService.Launch(workId, source)
    if (!result) throw new Error('启动游戏失败：后端未返回结果')
    return result
  },

  /** 系统「选择文件」对话框，只筛 .exe；用户取消返回空串（不是错误） */
  async pickExecutable(title = '选择游戏程序'): Promise<string> {
    return (await DialogService.PickExecutable(title)) ?? ''
  },
}
