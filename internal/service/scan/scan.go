// internal/service/scan/scan.go
//
// 本地磁盘扫描：把「一个装着游戏文件夹的根目录」读成游戏列表的两种粒度。
//
// 两个方法的分工（对应两种用途，**别混用**）：
//   - ScanExecutablesInSubfolders：通用。往一级子文件夹里看一眼 .exe，
//     用 exe 文件名当游戏名候选 —— 文件夹怎么命名都行，靠 exe 认游戏。
//   - ScanSubfolderNames：特定。**只读根目录本身**，把一级子文件夹名当游戏名 ——
//     给「已经按规则命名好」的库用，不打开任何子文件夹，省掉 N 次 I/O。用户原话：
//     「第二个服务函数是用来已经规范命名文件夹内的游戏(节约资源 特定)」。
//
// 共同约束（用户明确要求）：
//   - 只扫一层：根目录下的 .exe 不算（那是散装文件，不构成「一个游戏文件夹」）；
//   - 不递归：游戏目录里常有几十层资源/存档，递归会把盘读穿。
//
// 暂未在 main.go 注册为 Wails 服务（用户要求「暂时不为前端提供 api 调用」），
// 但方法一律按可绑定的形状写（导出方法 + ctx 首参 + json tag 的返回结构体）：
// 将来要暴露给前端时，只要在 main.go 里加一行
// application.NewService(&scan.ScanService{}) 并重新生成绑定即可。
package scan

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanService 本地目录扫描服务。
//
// 无状态：根目录由每次调用的参数传入（游戏库可能挂多个根），所以结构体里
// 刻意不放 Root 字段 —— 有字段就会有人改它，然后并发调用互相踩。
type ScanService struct{}

// GameExecutable 是一级子文件夹里的一个 .exe。
type GameExecutable struct {
	FileName string `json:"fileName"` // 含扩展名，原样大小写，如 game.EXE
	Name     string `json:"name"`     // 去掉扩展名后的文件名 —— 游戏名候选
	Path     string `json:"path"`     // 绝对路径（将来要拿它启动游戏，不能依赖进程工作目录）
	Size     int64  `json:"size"`     // 字节；取不到信息时为 0（不影响「这是个 exe」的判定）
}

// GameFolder 是一个一级子文件夹及其内部的 .exe。
type GameFolder struct {
	FolderName  string           `json:"folderName"`
	FolderPath  string           `json:"folderPath"`  // 绝对路径
	Executables []GameExecutable `json:"executables"` // 按文件名排序（os.ReadDir 的既定顺序），不按大小
}

// SkippedFolder 记录被跳过的子文件夹：读不动（权限/坏盘/被占用），不是扫描失败。
type SkippedFolder struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// GameScanResult 是 ScanExecutablesInSubfolders 的结果。
type GameScanResult struct {
	Root    string          `json:"root"`    // 规范化后的绝对根目录
	Folders []GameFolder    `json:"folders"` // 每个都至少有一个 .exe
	Skipped []SkippedFolder `json:"skipped"` // 读不动的子文件夹（可能为空）
}

// ===== 对外方法 =====

// ScanExecutablesInSubfolders 扫描根目录的**一级子文件夹**，返回每个文件夹里的 .exe。
//
// 语义边界（都是用户明确要求的）：
//   - 不含根目录本身的 .exe；
//   - 不递归进子文件夹的下一层；
//   - 没有 .exe 的子文件夹不进结果（工具/资料/存档目录不是游戏），
//     但它们在 ScanSubfolderNames 里会出现 —— 那个方法是纯名单，不做判断。
//
// 容错：某个子文件夹读不动（无权限、坏盘、被独占）只记进 Skipped 并继续，
// 不让一个坏文件夹毁掉整次扫描。
func (s *ScanService) ScanExecutablesInSubfolders(ctx context.Context, root string) (GameScanResult, error) {
	root, folderNames, err := readChildDirectories(ctx, root)
	if err != nil {
		return GameScanResult{}, err
	}

	result := GameScanResult{
		Root:    root,
		Folders: make([]GameFolder, 0, len(folderNames)),
		Skipped: make([]SkippedFolder, 0),
	}

	for _, folderName := range folderNames {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("扫描被取消: %w", err)
		}

		folderPath := filepath.Join(root, folderName)
		executables, err := executablesIn(ctx, folderPath)
		if err != nil {
			// 取消要中止整次扫描；其余（权限/IO）只跳过这一个文件夹。
			if ctxErr := ctx.Err(); ctxErr != nil {
				return result, fmt.Errorf("扫描被取消: %w", ctxErr)
			}
			result.Skipped = append(result.Skipped, SkippedFolder{Path: folderPath, Reason: err.Error()})
			continue
		}
		if len(executables) == 0 {
			continue
		}

		result.Folders = append(result.Folders, GameFolder{
			FolderName:  folderName,
			FolderPath:  folderPath,
			Executables: executables,
		})
	}

	return result, nil
}

// ScanSubfolderNames 只读根目录一层，返回一级子文件夹名（按名称排序）。
//
// 给「文件夹已经规范命名」的游戏库用：**不打开任何子文件夹**，一次 ReadDir 结束，
// 名称本身就是游戏名。比 ScanExecutablesInSubfolders 省掉 N 次目录读取，
// 代价是对命名没有兜底 —— 名字不规范就直接是脏数据。
//
// 与上一个方法的另一个差别：没有 .exe 的文件夹这里也照样返回（纯名单不做判断），
// 要不要过滤交给调用方。
func (s *ScanService) ScanSubfolderNames(ctx context.Context, root string) ([]string, error) {
	_, folderNames, err := readChildDirectories(ctx, root)
	if err != nil {
		return nil, err
	}
	return folderNames, nil
}

// ===== 内部实现 =====

// readChildDirectories 校验根目录并列出它的一级子文件夹（绝对路径 + 名称）。
//
// 两个对外方法共用这一份校验，保证「什么算合法根目录」只有一处定义。
func readChildDirectories(ctx context.Context, root string) (string, []string, error) {
	root, err := normalizeScanRoot(root)
	if err != nil {
		return "", nil, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return root, nil, fmt.Errorf("读取目录 %s 失败: %w", root, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		// 目录很大时别让人等一个已经撤销的操作：每个条目查一次取消。
		if ctxErr := ctx.Err(); ctxErr != nil {
			return root, nil, fmt.Errorf("扫描被取消: %w", ctxErr)
		}
		if isDirectory(root, entry) {
			names = append(names, entry.Name())
		}
	}
	return root, names, nil
}

// normalizeScanRoot 把用户给的目录变成可靠的绝对路径，并在解析前就拒掉坏输入。
//
// 为什么要 Abs：扫描结果里的 Path 将来要用来启动游戏，而应用的工作目录是
// 可执行文件所在目录（开发态是仓库根），相对路径换一个上下文就失效。
func normalizeScanRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("扫描目录为空")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("解析扫描目录 %q 失败: %w", root, err)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("扫描目录不存在: %s", abs)
		}
		return "", fmt.Errorf("无法访问扫描目录 %s: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("扫描目标不是文件夹: %s", abs)
	}
	return abs, nil
}

// executablesIn **只读一层**，返回 folderPath 里所有 .exe 文件（按文件名排序）。
func executablesIn(ctx context.Context, folderPath string) ([]GameExecutable, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	executables := make([]GameExecutable, 0, 4)
	for _, entry := range entries {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if !hasExeSuffix(entry.Name()) {
			continue
		}
		// 名字以 .exe 结尾的目录（少见，但 Windows 上真能建）不是可执行文件。
		if isDirectory(folderPath, entry) {
			continue
		}

		fileName := entry.Name()
		size := int64(0)
		if info, infoErr := entry.Info(); infoErr == nil {
			size = info.Size()
		}
		executables = append(executables, GameExecutable{
			FileName: fileName,
			Name:     strings.TrimSuffix(fileName, filepath.Ext(fileName)),
			Path:     filepath.Join(folderPath, fileName),
			Size:     size,
		})
	}
	return executables, nil
}

// hasExeSuffix 判定 .exe 扩展名，大小写不敏感（.EXE / .Exe 都算）。
//
// 用 filepath.Ext 而不是 strings.HasSuffix：后者会把 "game.exe.txt" 判成 exe。
func hasExeSuffix(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".exe")
}

// isDirectory 判断一个目录项是不是文件夹。
//
// 不直接用 entry.IsDir()：Windows 下目录联接（junction）/符号链接的 IsDir() 是 false，
// 而游戏常被 junction 到别的盘（C:\Games\某游戏 → D:\某游戏）。这类条目再 Stat 一次
// （跟随链接）确认；扫描深度只有一层，不会顺着链接转圈。
func isDirectory(parent string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&fs.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(parent, entry.Name()))
	return err == nil && info.IsDir()
}
