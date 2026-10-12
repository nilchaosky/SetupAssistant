package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// Shortcut 在当前用户桌面创建快捷方式。
// target 是要启动的程序，name 是快捷方式名称，不含 .lnk。
func Shortcut(target, name string) error {
	desktop, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, 0)
	if err != nil {
		return fmt.Errorf("创建桌面快捷方式失败: %w", err)
	}
	return shortcutAt(desktop, target, name)
}

func shortcutAt(desktop, target, name string) error {
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("找不到 %s", target)
	}
	link := filepath.Join(desktop, name+".lnk")
	script := strings.Join([]string{
		"$s = (New-Object -ComObject WScript.Shell).CreateShortcut('" + psQuote(link) + "')",
		"$s.TargetPath = '" + psQuote(target) + "'",
		"$s.WorkingDirectory = '" + psQuote(filepath.Dir(target)) + "'",
		"$s.Save()",
	}, "; ")
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("创建桌面快捷方式失败: %w", err)
		}
		return fmt.Errorf("创建桌面快捷方式失败: %s", msg)
	}
	return nil
}

func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
