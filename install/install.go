// Package install 按清单顺序安装已经下载好的安装包。
// 安装位置是每条软件自己的目录。installDir 非空时加在目录前面。
// 没有安装包的条目跳过，某一项失败后继续安装后面的软件。
// 静默安装写不了环境变量的软件，在这一条安装成功后立刻配置。
// Java 只把勾选为 JAVA_HOME 的版本写入 JAVA_HOME。
// JetBrains、VS Code、Python 的 PATH 已由静默参数写入，不参与安装后配置。
package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nilchaosky/SetupAssistant/install/mode"
	"github.com/nilchaosky/SetupAssistant/terminal"
)

// Run 依次安装 items。一条安装成功后立刻配置它的环境变量，再处理下一条。
// downloadDir 是安装包所在目录。installDir 非空时作为每条软件目录的前缀。
func Run(downloadDir, installDir string, items []terminal.PendingInstall) error {
	var failed []error
	for _, item := range items {
		ok, err := Install(downloadDir, installDir, item)
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", label(item), err))
			continue
		}
		if !ok {
			continue
		}
		if err := Configure(installDir, []terminal.PendingInstall{item}); err != nil {
			fmt.Printf("环境配置失败：%v\n", err)
			failed = append(failed, err)
		}
	}
	return errors.Join(failed...)
}

// Install 安装一条软件，不配置环境变量。
// 返回值表示这次是否真正安装成功。已经安装的软件只警告，不算成功安装。
func Install(downloadDir, installDir string, item terminal.PendingInstall) (bool, error) {
	if strings.TrimSpace(item.Package) == "" {
		fmt.Printf("跳过 %s：没有安装包\n", label(item))
		return false, nil
	}
	ok, err := installOne(downloadDir, installDir, item)
	if err != nil {
		fmt.Printf("安装失败 %s：%v\n", label(item), err)
		return false, err
	}
	return ok, nil
}

// Configure 按传入顺序逐条写入环境变量。
func Configure(installDir string, items []terminal.PendingInstall) error {
	return configureEnv(installDir, items)
}

// installOne 准备安装包和目标目录，再按软件名执行对应的安装过程。
// 返回值表示这次是否真正安装成功。已经安装的软件只警告，不算成功安装。
func installOne(downloadDir, installDir string, item terminal.PendingInstall) (bool, error) {
	pkg, err := packageFile(downloadDir, item.Package)
	if err != nil {
		return false, err
	}
	dest, err := targetDir(installDir, item)
	if err != nil {
		return false, err
	}
	switch item.Name {
	case "Java":
		return mode.Java(label(item), pkg, dest)
	case "Python":
		return mode.Python(label(item), pkg, dest)
	case "Notepad++":
		return mode.Notepad(label(item), pkg, dest)
	case "VS Code":
		return mode.VSCode(label(item), pkg, dest)
	case "IntelliJ IDEA", "PyCharm", "GoLand", "WebStorm":
		return mode.JetBrains(label(item), item.Name, pkg, dest)
	case "MySQL":
		return mode.MySQL(label(item), pkg, dest)
	case "PostgreSQL":
		return mode.PostgreSQL(label(item), pkg, dest)
	default:
		return false, fmt.Errorf("不支持的软件 %s", item.Name)
	}
}

// configureEnv 写入环境变量。
// 只有静默安装无法配置环境、并且这次真正安装成功的软件才会处理。
// Java 只配置勾选为 JAVA_HOME 的版本。
func configureEnv(installDir string, items []terminal.PendingInstall) error {
	var failed []error
	for _, item := range items {
		if !needsEnv(item) {
			continue
		}
		dest, err := targetDir(installDir, item)
		if err != nil {
			failed = append(failed, err)
			continue
		}
		if err := mode.ApplyEnv(item.Name, dest); err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", label(item), err))
		}
	}
	return errors.Join(failed...)
}

// needsEnv 判断是否要在安装结束后写环境变量。
// JetBrains 用静默配置的 updatePATH，VS Code 用 addtopath，Python 用 PrependPath 或 ADDLOCAL，这些都不在这里处理。
func needsEnv(item terminal.PendingInstall) bool {
	switch item.Name {
	case "Java":
		return item.JavaHome
	case "MySQL", "PostgreSQL":
		return true
	default:
		return false
	}
}

// openResource 读取 resource/ 下的安装包。测试可以换成内存里的内容。
var openResource = os.ReadFile

// packageFile 返回要执行的安装包路径。resource/ 下的文件先写到 downloadDir。
func packageFile(downloadDir, pkg string) (string, error) {
	pkg = strings.TrimSpace(pkg)
	name := filepath.Base(filepath.FromSlash(pkg))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("找不到安装包 %s", pkg)
	}
	dest, err := filepath.Abs(filepath.Join(downloadDir, name))
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(filepath.ToSlash(pkg), "resource/") {
		data, err := openResource(filepath.ToSlash(pkg))
		if err != nil {
			return "", fmt.Errorf("找不到安装包 %s", pkg)
		}
		if err := os.MkdirAll(downloadDir, 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(dest, data, 0o755); err != nil {
			return "", err
		}
		return dest, nil
	}
	if _, err := os.Stat(dest); err != nil {
		return "", fmt.Errorf("找不到安装包 %s", dest)
	}
	return dest, nil
}

// targetDir 返回这条软件的安装目录。installDir 非空时加在前面。
func targetDir(installDir string, item terminal.PendingInstall) (string, error) {
	path := filepath.FromSlash(installPath(item))
	if root := strings.TrimSpace(installDir); root != "" {
		path = filepath.Join(root, path)
	}
	return filepath.Abs(path)
}

// installPath 返回软件目录。dir 为空时用软件名。
func installPath(item terminal.PendingInstall) string {
	if dir := strings.TrimSpace(item.Dir); dir != "" {
		return dir
	}
	return item.Name
}

func label(item terminal.PendingInstall) string {
	if item.Version == "" {
		return item.Name
	}
	return item.Name + " " + item.Version
}
