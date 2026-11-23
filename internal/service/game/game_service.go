// internal/service/game/game_service.go
//
// 游戏启动服务：把「游戏库里的一个作品」和「本机上的游戏程序」接起来。
//
// 位置存哪儿：和游戏库元数据**同一张表** —— internal/database/library_repo.go 的
// library_works.game_path 列（一个作品对应本机一个游戏程序）。没有单开表：
// 位置的生命周期就和这条库记录绑在一起（删库记录自动消失），重新抓取/重新入库
// 也不会把它冲掉（Upsert 的 ON CONFLICT SET 刻意不含 game_path）。
//
// 为什么单独一个服务而不是塞进 LibraryService：
// LibraryService 回答「作品是什么」（抓取 / 缓存 / 入库），这里回答
// 「怎么把游戏跑起来」（定位 exe、校验文件、起进程）。两者依赖的东西不同，
// 混在一起以后加"启动参数 / 工作目录 / 启动前检查"又得改一个已经很大的服务。
//
// 放在独立子包（与 scan 一致）：同一功能域的文件集中在一处，service 根目录
// 只留跨域编排（TODO / 窗口 / 游戏库 / 对话框）。
package game

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"WeedyBox/internal/database"
)

// GameService 是暴露给前端的游戏启动服务。
type GameService struct {
	DB *database.SQLiteService

	// running 是「正在运行的游戏」缓存：key = 归一化后的 exe 绝对路径，value = 进程号。
	//
	// 用户要求的语义：游戏在运行 → 把路径载入缓存，方便前端轮询实时看状态；
	// 不再运行且路径还在缓存里 → 移除缓存。缓存只是加速与状态来源，
	// 每次查询仍会复核进程是否真的还在（PID 可能被系统复用，进程也可能自己退出）。
	mu      sync.Mutex
	running map[string]uint32
}

// GamePathState 是「设置游戏位置」成功后返回的状态。
// Path 是归一化后的绝对路径；Exists 是写入前校验时文件确实存在（SetPath 只存能打开的程序）。
type GamePathState struct {
	WorkID string `json:"workId"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// SetGamePathRequest 是「设置游戏位置」的入参。
// 可选参数在 Wails 绑定里不好表达，所以统一用结构体（与 FetchWorkRequest 同一风格）。
type SetGamePathRequest struct {
	WorkID string `json:"workId"`
	Path   string `json:"path"`
	Source string `json:"source"` // 元数据来源，空 = dlsite
}

// SetLaunchModeRequest 是「切换启动方式」的入参（Steam 源才有得选）。
//
// Mode："steam" = 跑 steam://rungameid/<appid> 交给 Steam 客户端；
// "exe" = 直接跑库里记录的 game_path；空串 = 恢复默认（Steam 源走 steam，其它走 exe）。
type SetLaunchModeRequest struct {
	WorkID string `json:"workId"`
	Source string `json:"source"`
	Mode   string `json:"mode"`
}

// GameLaunchModeState 是切换启动方式后的结果。Mode 是**生效值**（空串已按来源解析）。
type GameLaunchModeState struct {
	WorkID string `json:"workId"`
	Mode   string `json:"mode"`
}

// GameLaunchResult 是「启动游戏」的结果。
//
// Missing 单独给一个布尔字段（而不是只靠错误字符串）：前端靠它判断
// 「是位置没设置/文件不在了，需要让用户选一次 exe」还是「启动真的失败了」，
// 解析中文错误文案来分支太脆。
//
// Mode 是这次实际用的启动方式（steam / exe），前端据此显示回执。
type GameLaunchResult struct {
	WorkID  string `json:"workId"`
	Path    string `json:"path"`
	Started bool   `json:"started"`
	Running bool   `json:"running"`
	Missing bool   `json:"missing"`
	Message string `json:"message"`
	Mode    string `json:"mode"`
}

// GameStatus 是「这个作品的游戏状态」：位置、文件是否还在、是否正在运行。
//
// 前端进入作品详情 / 定时轮询时调它：
//   - Path 为空        → 还没设置位置（按钮第一次点击去选程序）；
//   - Exists = false   → 位置记了但 exe 已不在（提示"已删除或变更位置"，启动按钮禁用）；
//   - Running = true   → 游戏正在运行（按钮显示「运行中」并禁用，防止重复启动）。
//
// LaunchMode 是生效的启动方式（steam / exe）：Steam 源默认 steam，其余默认 exe。
type GameStatus struct {
	WorkID     string `json:"workId"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Running    bool   `json:"running"`
	PID        uint32 `json:"pid"`
	LaunchMode string `json:"launchMode"`
}

// ===== 对外方法 =====

// SetPath 记录/更新一个作品的游戏位置（用户点「游戏启动」但没有位置、或主动「更换位置」时调用）。
//
// 写入前必须校验：路径非空、解析成绝对路径后文件真的存在、且不是文件夹。
// 这样库里不会存进一条「误选文件夹」或「已经不存在的程序」，启动时就不会莫名失败。
func (s *GameService) SetPath(ctx context.Context, req SetGamePathRequest) (GamePathState, error) {
	state := GamePathState{WorkID: strings.ToUpper(strings.TrimSpace(req.WorkID))}
	if state.WorkID == "" {
		return state, errors.New("作品编号为空")
	}
	if s.DB == nil {
		return state, errors.New("数据库未就绪")
	}

	path, err := normalizeGamePath(req.Path)
	if err != nil {
		return state, err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.DB.SetGamePath(ctx, sourceOrDefault(req.Source), state.WorkID, path); err != nil {
		return state, err
	}
	state.Path = path
	state.Exists = true
	return state, nil
}

// Status 读一个作品的游戏状态：位置、exe 是否还存在、是否正在运行。
//
// 前端进入作品详情时调一次，之后定时轮询（运行中 ↔ 已退出 要能实时反映）。
// 运行状态的判定顺序：
//  1. 路径为空 → 一切都是 false；
//  2. exe 不在 → 丢掉缓存里的路径，返回 Exists=false（前端提示"已删除或变更位置"）；
//  3. 先查缓存：缓存里有 PID 且该 PID 的映像仍是这个 exe → 运行中；
//     缓存里的 PID 已失效（进程退出 / PID 被复用）→ 从缓存移除；
//  4. 缓存没有命中 → 扫一遍系统进程找这个 exe，找到就把路径+PID 载入缓存。
//
// 进程查询本身很便宜（Toolhelp 快照），每次进入页面/轮询调一次足够实时。
func (s *GameService) Status(ctx context.Context, workID, source string) (GameStatus, error) {
	status := GameStatus{WorkID: strings.ToUpper(strings.TrimSpace(workID))}
	if status.WorkID == "" {
		return status, nil
	}
	if s.DB == nil {
		return status, errors.New("数据库未就绪")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	src := sourceOrDefault(source)
	path, err := s.DB.GetGamePath(ctx, src, status.WorkID)
	if err != nil {
		return status, err
	}
	status.Path = path

	mode, err := s.DB.GetLaunchMode(ctx, src, status.WorkID)
	if err != nil {
		return status, err
	}
	status.LaunchMode = resolveLaunchMode(mode, src)

	if path == "" {
		return status, nil
	}
	if !gameFileExists(path) {
		// 文件都没了：缓存里若还留着它，按用户要求移除
		s.forgetRunning(path)
		return status, nil
	}

	status.Exists = true
	status.Running, status.PID = s.isRunning(path)
	return status, nil
}

// Launch 启动游戏：按库里记录的启动方式分流。
//
//   - steam 方式（Steam 源默认）：跑 steam://rungameid/<appid>，由 Steam 客户端拉起。
//     这种方式**不需要** game_path —— 游戏装没装、装在哪由 Steam 自己管；
//   - exe 方式（其它来源默认，或用户手动切回）：用 game_path 起独立进程。
//
// 分支：
//   - exe 方式下没有记录 / 记录的文件已不存在 → 返回 Started=false、Missing=true、
//     Message 说明原因（**不报 error**：这是可预期的、需要用户选一次程序的状态，
//     前端据此弹选择框）；
//   - 有文件但进程起不来（权限/损坏/被拦截）→ 返回 error，前端按失败提示；
//   - 成功 → Started=true，exe 方式还会把 PID 载入运行缓存。
//
// 启动实现见 launch_windows.go：exe 用独立进程 + 工作目录设为 exe 所在目录
// （很多游戏按相对路径找资源，不设工作目录会找不到 pak/存档）；
// Steam URL 交给系统默认协议处理器（ShellExecute "open"）。
func (s *GameService) Launch(ctx context.Context, workID, source string) (GameLaunchResult, error) {
	result := GameLaunchResult{WorkID: strings.ToUpper(strings.TrimSpace(workID))}
	if result.WorkID == "" {
		return result, errors.New("作品编号为空")
	}
	if s.DB == nil {
		return result, errors.New("数据库未就绪")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	src := sourceOrDefault(source)

	path, err := s.DB.GetGamePath(ctx, src, result.WorkID)
	if err != nil {
		return result, err
	}
	result.Path = path

	storedMode, err := s.DB.GetLaunchMode(ctx, src, result.WorkID)
	if err != nil {
		return result, err
	}
	mode := resolveLaunchMode(storedMode, src)
	result.Mode = mode

	// Steam 方式：不需要本地 exe（没装在本地 / 用户就是要让 Steam 客户端处理都成立）
	if mode == database.LaunchModeSteam {
		appID := steamAppID(result.WorkID)
		if appID == "" {
			return result, fmt.Errorf("作品编号 %q 不是有效的 Steam appid", result.WorkID)
		}
		url := steamRunGameURL(appID)
		if err := startURL(url); err != nil {
			return result, fmt.Errorf("通过 Steam 启动失败: %w", err)
		}
		result.Started = true
		result.Message = fmt.Sprintf("已请求 Steam 启动游戏（%s）", url)
		return result, nil
	}

	if path == "" {
		result.Missing = true
		result.Message = "还没有记录这个游戏的启动位置"
		return result, nil
	}
	if !gameFileExists(path) {
		s.forgetRunning(path)
		result.Missing = true
		result.Message = fmt.Sprintf("游戏程序不存在（可能被移动或卸载）：%s", path)
		return result, nil
	}

	pid, err := startDetached(path)
	if err != nil {
		return result, fmt.Errorf("启动游戏失败: %w", err)
	}
	s.rememberRunning(path, pid)
	result.Started = true
	result.Running = true
	return result, nil
}

// SetLaunchMode 切换一个作品的启动方式（Steam 源的作品才有"通过 Steam / 本地程序"两种）。
//
// 只改「下次怎么启动」，不顺手启动游戏。mode 的取值与校验见 database.SetLaunchMode；
// 额外一条：steam 方式只对 steam 来源开放 —— dlsite 作品没有 appid，拼不出 Steam URL。
func (s *GameService) SetLaunchMode(ctx context.Context, req SetLaunchModeRequest) (GameLaunchModeState, error) {
	state := GameLaunchModeState{WorkID: strings.ToUpper(strings.TrimSpace(req.WorkID))}
	if state.WorkID == "" {
		return state, errors.New("作品编号为空")
	}
	if s.DB == nil {
		return state, errors.New("数据库未就绪")
	}

	src := sourceOrDefault(req.Source)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	switch mode {
	case "", database.LaunchModeSteam, database.LaunchModeExe:
	default:
		return state, fmt.Errorf("不支持的启动方式: %q", req.Mode)
	}
	if mode == database.LaunchModeSteam && src != database.SourceSteam {
		return state, errors.New("只有 Steam 来源的作品才能通过 Steam 启动")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.DB.SetLaunchMode(ctx, src, state.WorkID, mode); err != nil {
		return state, err
	}
	state.Mode = resolveLaunchMode(mode, src)
	return state, nil
}

// ===== 内部实现 =====

// sourceOrDefault 来源标识：空值一律当 dlsite（当前唯一来源）。
func sourceOrDefault(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return database.SourceDLsite
	}
	return source
}

// defaultLaunchMode 按来源取默认启动方式（用户 2026-11-23 要求）：
// Steam 来源默认跑 steam://rungameid/<appid>（不依赖本地 exe），其它来源只能跑本地程序。
func defaultLaunchMode(source string) string {
	if source == database.SourceSteam {
		return database.LaunchModeSteam
	}
	return database.LaunchModeExe
}

// resolveLaunchMode 把库里存的值解析成生效值：空串/非法值一律退回来源默认，
// 这样「从没选过」和「显式选了默认」对启动行为是一致的。
func resolveLaunchMode(stored, source string) string {
	switch strings.ToLower(strings.TrimSpace(stored)) {
	case database.LaunchModeSteam:
		return database.LaunchModeSteam
	case database.LaunchModeExe:
		return database.LaunchModeExe
	default:
		return defaultLaunchMode(source)
	}
}

// steamAppID 校验并返回 Steam appid（纯数字）；不是数字就返回空串。
// 不引 getMeta 包：这里只需要"是不是一串数字"，不需要商店接口那一套。
func steamAppID(workID string) string {
	id := strings.TrimSpace(workID)
	if id == "" {
		return ""
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

// steamRunGameURL 是 Steam 官方的「启动指定游戏」协议地址（Steam 客户端已安装时由系统转交）。
func steamRunGameURL(appID string) string {
	return "steam://rungameid/" + appID
}

// normalizeGamePath 把用户给的路径变成可靠的绝对路径，并校验它确实是一个存在的文件。
//
// 为什么一定要 Abs + Clean：启动进程的工作目录是可执行文件所在目录（开发态是仓库根），
// 相对路径换一个上下文就失效；库里应该只存不依赖 cwd 的绝对路径。
func normalizeGamePath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("游戏位置为空")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析游戏路径 %q 失败: %w", path, err)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("游戏程序不存在: %s", abs)
		}
		return "", fmt.Errorf("无法访问游戏程序 %s: %w", abs, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("选中的是文件夹，请选择游戏程序本身（.exe）: %s", abs)
	}
	return abs, nil
}

// gameFileExists 判断记下的位置此刻是否仍是一个存在的文件。
// 目录也算"不合法"：启动的是进程，不是文件夹。
func gameFileExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// normalizePathKey 是运行缓存的键：Windows 路径大小写不敏感，统一小写 + 干净化。
func normalizePathKey(path string) string {
	return strings.ToLower(filepath.Clean(strings.TrimSpace(path)))
}

// isRunning 判断这个 exe 现在是否在运行，并顺手维护运行缓存。
//
// 缓存命中也要复核（PID 可能被系统复用、游戏可能已经退出）：
// 复核失败就删缓存，再全量扫一遍系统进程。
func (s *GameService) isRunning(path string) (bool, uint32) {
	key := normalizePathKey(path)

	if pid, ok := s.cachedRunning(key); ok {
		if processAlive(pid, path) {
			return true, pid
		}
		s.forgetRunningKey(key)
	}

	pid, ok := findRunningProcess(path)
	if !ok {
		return false, 0
	}
	s.rememberRunning(path, pid)
	return true, pid
}

// rememberRunning 把「这个路径正在运行」载入缓存。
func (s *GameService) rememberRunning(path string, pid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running == nil {
		s.running = make(map[string]uint32)
	}
	s.running[normalizePathKey(path)] = pid
}

// cachedRunning 读缓存里的 PID。
func (s *GameService) cachedRunning(key string) (uint32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pid, ok := s.running[key]
	return pid, ok
}

// forgetRunning 按路径移除缓存（文件被删/游戏已退出时调用）。
func (s *GameService) forgetRunning(path string) {
	s.forgetRunningKey(normalizePathKey(path))
}

// forgetRunningKey 按缓存键移除（内部已经算好 key 时用，避免重复归一化）。
func (s *GameService) forgetRunningKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, key)
}
