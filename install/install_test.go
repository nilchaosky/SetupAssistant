package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nilchaosky/SetupAssistant/terminal"
)

func TestRunSkipsEmptyPackageAndInstallsNotepad(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "npp.8.9.8.Installer.x64.exe")
	if err := os.WriteFile(exe, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs string
	orig := startInstaller
	startInstaller = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { startInstaller = orig })

	err := Run(dir, `D:\`, []terminal.PendingInstall{
		{Name: "Go"},
		{
			Name:    "Notepad++",
			Version: "v8.9.8",
			Package: "npp.8.9.8.Installer.x64.exe",
			Dir:     "Editor/Notepad++",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantExe, err := filepath.Abs(exe)
	if err != nil {
		t.Fatal(err)
	}
	if gotExe != wantExe {
		t.Fatalf("exe = %s", gotExe)
	}
	if !filepath.IsAbs(gotExe) {
		t.Fatalf("exe is not absolute: %s", gotExe)
	}
	if gotArgs != `/S /D=D:\Editor\Notepad++` {
		t.Fatalf("args = %s", gotArgs)
	}
	if !strings.HasSuffix(gotArgs, `D:\Editor\Notepad++`) {
		t.Fatalf("/D= must stay last: %s", gotArgs)
	}
}

func TestRunReportsMissingInstaller(t *testing.T) {
	orig := startInstaller
	startInstaller = func(exe, params string) error {
		t.Fatal("should not start")
		return nil
	}
	t.Cleanup(func() { startInstaller = orig })

	err := Run(t.TempDir(), `E:\`, []terminal.PendingInstall{{
		Name:    "Notepad++",
		Package: "npp.8.9.8.Installer.x64.exe",
	}})
	if err == nil || !strings.Contains(err.Error(), "找不到安装包") {
		t.Fatalf("err = %v", err)
	}
}
