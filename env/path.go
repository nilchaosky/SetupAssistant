package env

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const systemEnvironmentKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`

var sendMessageTimeoutW = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")

// 测试可以换成不写注册表的实现。
var addToPath = AddToPath

// setSystemString 写入系统环境变量。值没有变化时不广播。
var setSystemString = func(name, value string) error {
	changed, err := setStringValue(registry.LOCAL_MACHINE, systemEnvironmentKey, name, value)
	if err != nil {
		return fmt.Errorf("设置 %s 失败: %w", name, err)
	}
	if changed {
		broadcastEnvironment()
	}
	return nil
}

func setStringValue(root registry.Key, keyPath, name, value string) (bool, error) {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	current, _, err := key.GetStringValue(name)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return false, err
	}
	if err == nil && current == value {
		return false, nil
	}
	if err := key.SetStringValue(name, value); err != nil {
		return false, err
	}
	return true, nil
}

// AddToPath 把 dir 加进系统 PATH。目录已经在 PATH 里时不重复添加。
func AddToPath(dir string) error {
	changed, err := addPathValue(registry.LOCAL_MACHINE, systemEnvironmentKey, dir)
	if err != nil {
		return fmt.Errorf("加入 PATH 失败: %w", err)
	}
	if changed {
		broadcastEnvironment()
	}
	return nil
}

func addPathValue(root registry.Key, keyPath, dir string) (bool, error) {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	current, _, err := key.GetStringValue("Path")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return false, err
	}
	updated, changed := appendPathEntry(current, dir)
	if !changed {
		return false, nil
	}
	if err := key.SetExpandStringValue("Path", updated); err != nil {
		return false, err
	}
	return true, nil
}

// appendPathEntry 把 dir 接到 PATH 末尾。已经包含时返回原来的值。
func appendPathEntry(existing, dir string) (string, bool) {
	dir = cleanPathEntry(dir)
	if dir == "" || pathContains(existing, dir) {
		return existing, false
	}
	existing = strings.TrimRight(strings.TrimSpace(existing), ";")
	if existing == "" {
		return dir, true
	}
	return existing + ";" + dir, true
}

func pathContains(existing, dir string) bool {
	for _, part := range strings.Split(existing, ";") {
		if strings.EqualFold(cleanPathEntry(part), dir) {
			return true
		}
	}
	return false
}

func cleanPathEntry(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	return strings.TrimRight(filepath.Clean(dir), `\/`)
}

func broadcastEnvironment() {
	ptr, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	const (
		hwndBroadcast   = uintptr(0xffff)
		wmSettingChange = uintptr(0x001A)
		smtoAbortIfHung = uintptr(0x0002)
	)
	_, _, _ = sendMessageTimeoutW.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(ptr)),
		smtoAbortIfHung,
		5000,
		0,
	)
}
