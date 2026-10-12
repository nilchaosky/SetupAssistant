// SetupAssistant 按 app.json 选择软件，逐个下载、安装、删除安装包并配置环境。
// 启动时若还不是管理员，会另开一个提权窗口继续，后续安装包沿用这次权限。
package main

//go:generate go run github.com/akavel/rsrc@v0.10.2 -manifest app.manifest -o rsrc_windows_amd64.syso

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nilchaosky/SetupAssistant/download"
	"github.com/nilchaosky/SetupAssistant/install"
	"github.com/nilchaosky/SetupAssistant/terminal"
	"github.com/nilchaosky/SetupAssistant/windows"
)

func main() {
	windows.EnsureAdmin()

	appJSON, err := os.ReadFile("app.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 app.json 失败: %v\n", err)
		os.Exit(1)
	}
	choices, err := terminal.SelectInstallTarget(appJSON)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return
		}
		fmt.Fprintf(os.Stderr, "选择失败: %v\n", err)
		os.Exit(1)
	}
	var failed []error
	for _, item := range choices {
		name := item.Name
		if item.Version != "" {
			name += " " + item.Version
		}
		if strings.TrimSpace(item.Dir) != "" {
			fmt.Printf("安装目录 %s：%s\n", name, item.Dir)
		}
		if err := download.One(download.Dir(), item); err != nil {
			fmt.Printf("下载失败 %s：%v\n", name, err)
			failed = append(failed, fmt.Errorf("%s: %w", name, err))
			continue
		}
		ok, err := install.Install(download.Dir(), "", item)
		if removeErr := download.RemoveFile(item.Package); removeErr != nil {
			fmt.Fprintf(os.Stderr, "删除安装包失败 %s：%v\n", name, removeErr)
			failed = append(failed, fmt.Errorf("%s: %w", name, removeErr))
		}
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if !ok {
			continue
		}
		if err := install.Configure("", []terminal.PendingInstall{item}); err != nil {
			fmt.Printf("环境配置失败：%v\n", err)
			failed = append(failed, err)
		}
	}
	if len(failed) > 0 {
		os.Exit(1)
	}
}
