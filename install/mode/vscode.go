package mode

// VSCode 静默安装 VS Code。路径里可以有空格，所以 /DIR 要加引号。
//
//	/VERYSILENT              不显示向导和进度窗口
//	/NORESTART               装完不重启，也不询问是否重启
//	/DIR                     安装目录
//	addtopath                把 code 加进用户 PATH
//	desktopicon              创建桌面快捷方式
//	!runcode                 装完不启动 VS Code
//	!associatewithfiles      不注册文件关联
func VSCode(label, pkg, dest string) (bool, error) {
	dir, found, err := FindApp("Microsoft Visual Studio Code")
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	args := `/VERYSILENT /NORESTART /MERGETASKS=addtopath,desktopicon,!runcode,!associatewithfiles /DIR="` + dest + `"`
	if err := run(dest, pkg, args); err != nil {
		return false, err
	}
	succeed(dest)
	return true, nil
}
