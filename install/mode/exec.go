// Package mode 按软件实现各自的安装过程。
package mode

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nilchaosky/SetupAssistant/env"
	"github.com/nilchaosky/SetupAssistant/progress"
	"github.com/nilchaosky/SetupAssistant/windows"
)

// Bar 是安装进度。Finish 拉满，Abort 停在当前进度。
type Bar interface {
	Finish()
	Abort()
}

// 测试可以换成不启动安装包、不写环境变量的实现。
var (
	Start            = windows.Start
	StartBar         = func(label string) Bar { return progress.StartInstall(label) }
	ApplyEnv         = env.Apply
	PortInUse        = windows.PortInUse
	InstalledProduct = windows.InstalledProduct
	Shortcut         = windows.Shortcut
	FindMySQLConsole = findMySQLConsole
	FindJava         = windows.JavaHome
	FindPython       = windows.PythonHome
	FindApp          = windows.AppDir
	FindMySQLServer  = windows.MySQLServerDir
	FindPostgreSQL   = windows.PostgreSQLDir
	OpenConfig       = func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join("config", name))
	}
)

func run(dest, command, args string) error {
	return runInstaller(dest, command, args, false)
}

// runInstaller 显示进度并等待安装包结束。rebootOK 时把 msiexec 的重启请求视为成功。
func runInstaller(dest, command, args string, rebootOK bool) error {
	bar := StartBar(dest + " 安装中")
	err := Start(command, args)
	if rebootOK {
		err = ignoreReboot(err)
	}
	if err != nil {
		bar.Abort()
		return err
	}
	bar.Finish()
	return nil
}

func ignoreReboot(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *windows.ExitError
	if errors.As(err, &exitErr) && (exitErr.Code == 3010 || exitErr.Code == 1641) {
		return nil
	}
	return err
}

func succeed(dest string) {
	fmt.Printf("%s 安装完成\n", dest)
}

// skipInstalled 在产品已经存在时打印警告并跳过安装。只有查找失败才返回 error。
func skipInstalled(label, dir string, found bool, err error) (bool, error) {
	if err != nil {
		return false, fmt.Errorf("检查是否已安装失败: %w", err)
	}
	if !found {
		return false, nil
	}
	if dir == "" {
		fmt.Printf("警告 %s：产品已安装\n", label)
		return true, nil
	}
	fmt.Printf("警告 %s：产品已安装于 %s\n", label, dir)
	return true, nil
}

func requireFreePort(port string) error {
	busy, err := PortInUse(port)
	if err != nil {
		return err
	}
	if busy {
		return fmt.Errorf("端口 %s 已被占用", port)
	}
	return nil
}

// MsiexecPath 返回系统自带的 Windows Installer。
func MsiexecPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "msiexec.exe")
}

func readSettings(name string) (map[string]string, error) {
	data, err := OpenConfig(name)
	if err != nil {
		return nil, fmt.Errorf("未找到 config/%s", name)
	}
	settings := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		settings[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return settings, nil
}
