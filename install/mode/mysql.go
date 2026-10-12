package mode

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// MySQL 先装 MySQL Installer，再用控制台安装服务器。
// msi 不接受服务器目录，服务器目录由控制台的 install_dir 指定。
//
// msiexec：
//
//	/i          安装这个 msi
//	/qn         不显示界面
//	/norestart  装完不重启
//
// 控制台：
//
//	password               root 密码
//	port                   监听端口
//	tcp_ip                 允许通过 TCP 连接
//	open_windows_firewall  不开放防火墙
//	install_dir            服务器安装目录。必须加引号，否则盘符里的冒号会被当成新的配置块
//	--silent               不再询问
func MySQL(label, pkg, dest string) (bool, error) {
	version, err := mysqlVersion(pkg)
	if err != nil {
		return false, err
	}
	dir, found, err := FindMySQLServer(mysqlSeries(version))
	if skip, err := skipInstalled(label, dir, found, err); skip || err != nil {
		return false, err
	}
	port, password, err := mysqlAccount()
	if err != nil {
		return false, err
	}
	if err := requireFreePort(port); err != nil {
		return false, err
	}
	bar := StartBar(dest + " 安装中")
	if err := ignoreReboot(Start(MsiexecPath(), MySQLInstallerArgs(pkg))); err != nil {
		bar.Abort()
		return false, err
	}
	console, err := FindMySQLConsole()
	if err != nil {
		bar.Abort()
		return false, err
	}
	if err := Start(console, MySQLServerArgs(version, port, password, dest)); err != nil {
		bar.Abort()
		return false, err
	}
	bar.Finish()
	succeed(dest)
	return true, nil
}

// MySQLInstallerArgs 是安装 MySQL Installer 的 msiexec 参数。
func MySQLInstallerArgs(msiPath string) string {
	return "/i " + syscall.EscapeArg(msiPath) + " /qn /norestart"
}

// MySQLServerArgs 是 MySQLInstallerConsole 的服务器安装参数。
func MySQLServerArgs(version, port, password, dest string) string {
	return "--install server;" + version + ";x64:*:password=" + password +
		";port=" + port +
		";tcp_ip=true;open_windows_firewall=false;install_dir=\"" + dest + "\"" +
		" --silent --auto-handle-prereqs"
}

var mysqlVersionPattern = regexp.MustCompile(`(?i)^mysql-installer-(?:web-)?community-(\d+\.\d+\.\d+)\.\d+\.msi$`)

func mysqlVersion(pkg string) (string, error) {
	match := mysqlVersionPattern.FindStringSubmatch(filepath.Base(pkg))
	if match == nil {
		return "", fmt.Errorf("无法从安装包名得到 MySQL 版本: %s", pkg)
	}
	return match[1], nil
}

func mysqlSeries(version string) string {
	major, rest, ok := strings.Cut(version, ".")
	if !ok {
		return version
	}
	minor, _, _ := strings.Cut(rest, ".")
	return major + "." + minor
}

func mysqlAccount() (string, string, error) {
	settings, err := readSettings("mysql-silent.config")
	if err != nil {
		return "", "", err
	}
	password := settings["root_passwd"]
	if password == "" {
		return "", "", errors.New("config/mysql-silent.config 未设置 root_passwd")
	}
	port := settings["port"]
	if port == "" {
		port = "3306"
	}
	return port, password, nil
}

func findMySQLConsole() (string, error) {
	var dirs []string
	if dir := os.Getenv("ProgramFiles(x86)"); dir != "" {
		dirs = append(dirs, dir)
	}
	if dir := os.Getenv("ProgramFiles"); dir != "" {
		dirs = append(dirs, dir)
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, "MySQL", "MySQL Installer for Windows", "MySQLInstallerConsole.exe")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", errors.New("未找到 MySQLInstallerConsole.exe")
}
