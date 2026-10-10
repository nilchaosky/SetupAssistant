package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nilchaosky/SetupAssistant/terminal"
	"github.com/nilchaosky/SetupAssistant/windows"
)

var startInstaller = windows.Start

func Run(downloadDir, installDir string, items []terminal.PendingInstall) error {
	if strings.TrimSpace(installDir) == "" {
		return errors.New("没有安装目录")
	}
	var failed []error
	for _, item := range items {
		if strings.TrimSpace(item.Package) == "" {
			fmt.Printf("跳过 %s：没有安装包\n", label(item))
			continue
		}
		if err := installOne(downloadDir, installDir, item); err != nil {
			fmt.Printf("安装失败 %s：%v\n", label(item), err)
			failed = append(failed, fmt.Errorf("%s: %w", label(item), err))
		}
	}
	return errors.Join(failed...)
}

func installOne(downloadDir, installDir string, item terminal.PendingInstall) error {
	name := filepath.Base(strings.TrimSpace(item.Package))
	exe, err := filepath.Abs(filepath.Join(downloadDir, name))
	if err != nil {
		return err
	}
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("找不到安装包 %s", exe)
	}
	dest, err := filepath.Abs(filepath.Join(installDir, filepath.FromSlash(installPath(item))))
	if err != nil {
		return err
	}
	stop := spin(dest + " 安装中")
	err = startInstaller(exe, "/S /D="+dest)
	stop()
	if err != nil {
		return err
	}
	fmt.Printf("%s 安装完成\n", dest)
	return nil
}

func spin(text string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Print("\x1b[?25l")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		fmt.Printf("\r%s %s", text, frames[0])
		for {
			select {
			case <-done:
				fmt.Print("\r\x1b[K\x1b[?25h")
				return
			case <-ticker.C:
				i = (i + 1) % len(frames)
				fmt.Printf("\r%s %s", text, frames[i])
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
		wg.Wait()
	}
}

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
