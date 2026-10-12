package windows

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

// ExitError 是安装程序返回的非零退出码。
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("退出码 %d", e.Code)
}

// Start 用当前进程的权限启动安装包并等待结束。
// params 原样放进命令行，不再额外加引号，这样 /D= 才能识别带空格的目录。
func Start(exe, params string) error {
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:    syscall.EscapeArg(exe) + " " + params,
		HideWindow: true,
	}
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	return err
}
