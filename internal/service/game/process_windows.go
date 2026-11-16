// internal/service/game/process_windows.go
//
// Windows 下的「这个游戏是否在运行」检测。
//
// 为什么不用 gopsutil：本项目只跑 Windows，go.mod 里已经有 golang.org/x/sys
// （Steam 扫描用了 windows/registry），Toolhelp 快照 + QueryFullProcessImageName
// 就够用，不必为了一个"列进程"再引一个依赖树。
//
// 判定用**完整映像路径**而不是进程名：叫 game.exe 的程序可以有好几个，
// 只看名字会把别人的进程误判成这个游戏；完整路径能精确对应到库里记的那个 exe。
package game

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// findRunningProcess 扫一遍系统进程，找「映像完整路径 == path」的进程。
//
// 先用快照里的 exe 文件名做粗筛（绝大多数进程在第一步就被跳过），
// 名字对上了再 OpenProcess + QueryFullProcessImageName 比对完整路径。
func findRunningProcess(path string) (uint32, bool) {
	targetKey := normalizePathKey(path)
	targetName := filepath.Base(path)

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err := windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if !strings.EqualFold(name, targetName) {
			continue
		}
		if processImageMatches(entry.ProcessID, targetKey) {
			return entry.ProcessID, true
		}
	}
	return 0, false
}

// processAlive 判断缓存里的 PID 是否还活着，且映像仍是那个 exe。
//
// 必须比对路径而不是只看 PID：进程退出后 PID 会被系统复用，
// 单看 PID「还在」会把别的程序误判成游戏仍在运行。
func processAlive(pid uint32, path string) bool {
	if pid == 0 {
		return false
	}
	return processImageMatches(pid, normalizePathKey(path))
}

// processImageMatches 打开进程并比对完整映像路径。
func processImageMatches(pid uint32, targetKey string) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	return processHandleMatches(handle, targetKey)
}

// processHandleMatches 从已打开的进程句柄取完整映像路径并比对。
func processHandleMatches(handle windows.Handle, targetKey string) bool {
	buffer := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		return false
	}
	full := windows.UTF16ToString(buffer[:size])
	return normalizePathKey(full) == targetKey
}
