package mode

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// Python 安装 Python。2.7 是 msi，3.x 是官方 exe。
// 产品已经存在时只给出警告，不启动安装包，也不算安装失败。
//
// exe：
//
//	/quiet           不显示界面
//	InstallAllUsers  为所有用户安装
//	PrependPath      把安装目录和 Scripts 加进系统 PATH，并把 .PY 加进 PATHEXT
//	Include_pip      安装 pip
//	Include_test     不安装标准库测试
//	TargetDir        安装目录。路径有空格时要加引号
//
// msi：
//
//	/qn           不显示界面
//	/norestart    装完不重启
//	ALLUSERS      为所有用户安装
//	ADDLOCAL=ALL  安装全部组件，其中包含加入 PATH
//	TARGETDIR     安装目录。路径有空格时要加引号
func Python(label, pkg, dest string) (bool, error) {
	dir, found, err := installedPython(pkg)
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	if strings.EqualFold(filepath.Ext(pkg), ".msi") {
		args := "/i " + syscall.EscapeArg(pkg) + ` /qn /norestart ALLUSERS=1 ADDLOCAL=ALL TARGETDIR="` + dest + `"`
		if err := runInstaller(dest, MsiexecPath(), args, true); err != nil {
			return false, err
		}
	} else {
		args := `/quiet InstallAllUsers=1 PrependPath=1 Include_pip=1 Include_test=0 TargetDir="` + dest + `"`
		if err := run(dest, pkg, args); err != nil {
			return false, err
		}
	}
	succeed(dest)
	return true, nil
}

// installedPython 先看 msi 产品代码，再看 Python 注册的安装目录。
func installedPython(pkg string) (string, bool, error) {
	if strings.EqualFold(filepath.Ext(pkg), ".msi") {
		dir, installed, err := InstalledProduct(pkg)
		if err != nil || installed {
			return dir, installed, err
		}
	}
	series, err := pythonSeries(pkg)
	if err != nil {
		return "", false, err
	}
	return FindPython(series)
}

var pythonSeriesPattern = regexp.MustCompile(`(?i)python-(\d+\.\d+)`)

func pythonSeries(pkg string) (string, error) {
	match := pythonSeriesPattern.FindStringSubmatch(filepath.Base(pkg))
	if match == nil {
		return "", fmt.Errorf("无法从安装包名得到 Python 版本: %s", pkg)
	}
	return match[1], nil
}
