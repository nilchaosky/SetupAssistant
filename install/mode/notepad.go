package mode

import "path/filepath"

// Notepad 静默安装 Notepad++，并补上安装包不会创建的桌面快捷方式。
// /D 必须放在最后，而且不能加引号，否则带空格的路径会装错位置。
//
//	/S   静默安装
//	/D=  安装目录
func Notepad(label, pkg, dest string) (bool, error) {
	dir, found, err := FindApp("Notepad++")
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	if err := run(dest, pkg, "/S /D="+dest); err != nil {
		return false, err
	}
	if err := Shortcut(filepath.Join(dest, "notepad++.exe"), "Notepad++"); err != nil {
		return false, err
	}
	succeed(dest)
	return true, nil
}
