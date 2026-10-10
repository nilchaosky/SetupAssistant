package main

//go:generate go run github.com/akavel/rsrc@v0.10.2 -manifest app.manifest -o rsrc_windows_amd64.syso

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/nilchaosky/SetupAssistant/download"
	"github.com/nilchaosky/SetupAssistant/install"
	"github.com/nilchaosky/SetupAssistant/terminal"
	"github.com/nilchaosky/SetupAssistant/windows"
)

func main() {
	windows.EnsureAdmin()

	choices, installDir, err := terminal.SelectInstallTarget()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return
		}
		fmt.Fprintf(os.Stderr, "选择失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("安装目录：%s\n", installDir)

	if err := download.Run(choices); err != nil {
		fmt.Fprintf(os.Stderr, "下载失败: %v\n", err)
		os.Exit(1)
	}
	if err := install.Run(download.Dir(), installDir, choices); err != nil {
		fmt.Fprintf(os.Stderr, "安装失败: %v\n", err)
		os.Exit(1)
	}
}
