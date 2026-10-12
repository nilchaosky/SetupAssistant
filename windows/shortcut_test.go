package windows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortcutAtPointsToTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "notepad++.exe")
	if err := os.WriteFile(target, []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	desktop := filepath.Join(dir, "desktop")
	if err := os.MkdirAll(desktop, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := shortcutAt(desktop, target, "Notepad++"); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(desktop, "Notepad++.lnk")
	if _, err := os.Stat(link); err != nil {
		t.Fatal(err)
	}

	script := "(New-Object -ComObject WScript.Shell).CreateShortcut('" + psQuote(link) + "').TargetPath"
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("target = %s", got)
	}
}
