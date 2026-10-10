package windows

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

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
		return fmt.Errorf("退出码 %d", exitErr.ExitCode())
	}
	return err
}
