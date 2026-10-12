package windows

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	errorSuccess        = 0
	errorNoMoreItems    = 259
	errorMoreData       = 234
	installStateDefault = 5
)

var (
	msiDLL               = windows.NewLazySystemDLL("msi.dll")
	msiOpenDatabaseW     = msiDLL.NewProc("MsiOpenDatabaseW")
	msiDatabaseOpenViewW = msiDLL.NewProc("MsiDatabaseOpenViewW")
	msiViewExecute       = msiDLL.NewProc("MsiViewExecute")
	msiViewFetch         = msiDLL.NewProc("MsiViewFetch")
	msiRecordGetStringW  = msiDLL.NewProc("MsiRecordGetStringW")
	msiCloseHandle       = msiDLL.NewProc("MsiCloseHandle")
	msiQueryProductState = msiDLL.NewProc("MsiQueryProductStateW")
)

// InstalledProduct 查看 msi 里的 ProductCode 是否已经安装，并且程序文件还在。
// 产品记录还在、目录已经删掉时，不算已安装。
func InstalledProduct(msiPath string) (dir string, installed bool, err error) {
	code, err := productCode(msiPath)
	if err != nil {
		return "", false, err
	}
	state, _, _ := msiQueryProductState.Call(mustUTF16(code))
	if state != installStateDefault {
		return "", false, nil
	}
	dir = installedProductDir(code)
	if dir == "" {
		return "", false, nil
	}
	return dir, true, nil
}

func productCode(msiPath string) (string, error) {
	db, err := openDatabase(msiPath)
	if err != nil {
		return "", err
	}
	defer closeHandle(db)

	query, err := windows.UTF16PtrFromString("SELECT `Value` FROM `Property` WHERE `Property`='ProductCode'")
	if err != nil {
		return "", err
	}
	var view uintptr
	if err := msiStatus(msiDatabaseOpenViewW.Call(db, uintptr(unsafe.Pointer(query)), uintptr(unsafe.Pointer(&view)))); err != nil {
		return "", err
	}
	defer closeHandle(view)
	if err := msiStatus(msiViewExecute.Call(view, 0)); err != nil {
		return "", err
	}
	var record uintptr
	r, _, _ := msiViewFetch.Call(view, uintptr(unsafe.Pointer(&record)))
	if r == errorNoMoreItems {
		return "", fmt.Errorf("安装包没有 ProductCode")
	}
	if r != errorSuccess {
		return "", fmt.Errorf("读取 ProductCode 失败: %d", r)
	}
	defer closeHandle(record)
	return recordString(record, 1)
}

func openDatabase(msiPath string) (uintptr, error) {
	path, err := windows.UTF16PtrFromString(msiPath)
	if err != nil {
		return 0, err
	}
	var db uintptr
	if err := msiStatus(msiOpenDatabaseW.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&db)))); err != nil {
		return 0, err
	}
	return db, nil
}

func recordString(record uintptr, field uint32) (string, error) {
	buf := make([]uint16, 64)
	size := uint32(len(buf) - 1)
	r, _, _ := msiRecordGetStringW.Call(record, uintptr(field), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == errorMoreData {
		buf = make([]uint16, size+1)
		size = uint32(len(buf) - 1)
		r, _, _ = msiRecordGetStringW.Call(record, uintptr(field), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	}
	if r != errorSuccess {
		return "", fmt.Errorf("读取安装包字段失败: %d", r)
	}
	return windows.UTF16ToString(buf), nil
}

func installedProductDir(productCode string) string {
	keys := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + productCode,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\` + productCode,
	}
	for _, path := range keys {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		location, _, _ := key.GetStringValue("InstallLocation")
		icon, _, _ := key.GetStringValue("DisplayIcon")
		key.Close()
		if !programExists(location, icon) {
			continue
		}
		if dir := installDir(location, icon); dir != "" {
			return dir
		}
	}
	return ""
}

// installDir 从登记的位置取出安装目录。InstallLocation 优先，否则用程序文件所在目录。
func installDir(location, icon string) string {
	location = strings.TrimSpace(location)
	if location != "" {
		return strings.TrimRight(filepath.Clean(location), `\/`)
	}
	icon = strings.Trim(strings.TrimSpace(icon), `"`)
	if i := strings.LastIndex(icon, ","); i > 0 && iconIndex(icon[i+1:]) {
		icon = icon[:i]
	}
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return ""
	}
	if filepath.Ext(icon) != "" {
		icon = filepath.Dir(icon)
	}
	return strings.TrimRight(filepath.Clean(icon), `\/`)
}

func iconIndex(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func mustUTF16(value string) uintptr {
	ptr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return 0
	}
	return uintptr(unsafe.Pointer(ptr))
}

func msiStatus(code uintptr, _ uintptr, _ error) error {
	if code == errorSuccess {
		return nil
	}
	return fmt.Errorf("读取安装包失败: %d", code)
}

func closeHandle(handle uintptr) {
	if handle != 0 {
		_, _, _ = msiCloseHandle.Call(handle)
	}
}
