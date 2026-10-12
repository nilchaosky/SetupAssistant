// Package windows 负责提权、列出固定磁盘，以及用当前权限启动安装包。
package windows

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const seeMaskNoCloseProcess = 0x00000040

// EnsureAdmin 保证当前进程以管理员身份运行。
// 尚未提权时用 runas 重新启动自己并等待退出。用户拒绝 UAC 时直接结束。
func EnsureAdmin() {
	if elevated() {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法提权: %v\n", err)
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法提权: %v\n", err)
		os.Exit(1)
	}
	code, err := runElevated(exe, cwd, os.Args[1:])
	if err != nil {
		if errors.Is(err, windows.ERROR_CANCELLED) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "无法提权: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func elevated() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         windows.Handle
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     windows.Handle
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    windows.Handle
	dwHotKey     uint32
	hIcon        windows.Handle
	hProcess     windows.Handle
}

var procShellExecuteExW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

func runElevated(exe, cwd string, args []string) (int, error) {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return 1, err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 1, err
	}
	params, err := windows.UTF16PtrFromString(joinArgs(args))
	if err != nil {
		return 1, err
	}
	dir, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return 1, err
	}
	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: params,
		lpDirectory:  dir,
		nShow:        windows.SW_SHOWNORMAL,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	ret, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return 1, callErr
		}
		return 1, fmt.Errorf("无法启动管理员进程")
	}
	if info.hProcess == 0 {
		return 0, nil
	}
	defer windows.CloseHandle(info.hProcess)
	fmt.Println("请在管理员窗口中继续")
	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return 1, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err != nil {
		return 1, err
	}
	return int(code), nil
}

func joinArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = syscall.EscapeArg(arg)
	}
	return strings.Join(quoted, " ")
}
