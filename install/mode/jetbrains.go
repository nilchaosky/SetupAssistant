package mode

import (
	"fmt"
	"os"
	"path/filepath"
)

// JetBrains 静默安装 IntelliJ IDEA、PyCharm、GoLand 或 WebStorm。
// 安装包只能读取磁盘上的配置，所以先把嵌入的配置写到临时文件，结束后删除。
// /D 必须放在最后，而且不能加引号。
//
//	/S        静默安装。不带 /CONFIG 时，安装包不会创建桌面快捷方式，也不会修改 PATH
//	/CONFIG=  静默配置。launcher64=1 创建桌面快捷方式，updatePATH=1 把命令行启动器加进 PATH，updateContextMenu=0 不加右键菜单。PATH 只在这里写入，不参与安装后的环境配置
//	/D=       安装目录
func JetBrains(label, name, pkg, dest string) (bool, error) {
	dir, found, err := FindApp(name)
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	config, err := writeJetBrainsConfig()
	if err != nil {
		return false, err
	}
	defer os.Remove(config)
	if err := run(dest, pkg, "/S /CONFIG="+config+" /D="+dest); err != nil {
		return false, err
	}
	succeed(dest)
	return true, nil
}

// JetBrainsConfigPath 是安装期间写出的静默配置路径。
func JetBrainsConfigPath() string {
	return filepath.Join(os.TempDir(), "SetupAssistant-jetbrains-silent.config")
}

func writeJetBrainsConfig() (string, error) {
	data, err := OpenConfig("jetbrains-silent.config")
	if err != nil {
		return "", fmt.Errorf("未找到 config/jetbrains-silent.config")
	}
	path := JetBrainsConfigPath()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
