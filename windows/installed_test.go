package windows

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestSameProductMatchesEditionAndRejectsInsiders(t *testing.T) {
	if !sameProduct("Notepad++ (64-bit x64)", "Notepad++") {
		t.Fatal("notepad edition")
	}
	if !sameProduct("IntelliJ IDEA 2025.2", "IntelliJ IDEA") {
		t.Fatal("idea version")
	}
	if !sameProduct("Microsoft Visual Studio Code", "Microsoft Visual Studio Code") {
		t.Fatal("vscode")
	}
	if sameProduct("Microsoft Visual Studio Code Insiders", "Microsoft Visual Studio Code") {
		t.Fatal("insiders should stay separate")
	}
	if sameProduct("PyCharm", "IntelliJ IDEA") {
		t.Fatal("different product")
	}
}

func TestJavaHomeReadsFeatureRelease(t *testing.T) {
	const base = `Software\SetupAssistantJavaTest\JavaSoft`
	key, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\JDK\21.0.2`, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	if err := key.SetStringValue("JavaHome", `E:\Dev\Java\21\`); err != nil {
		key.Close()
		t.Fatal(err)
	}
	key.Close()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, base+`\JDK\21.0.2`)
		_ = registry.DeleteKey(registry.CURRENT_USER, base+`\JDK`)
		_ = registry.DeleteKey(registry.CURRENT_USER, base)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\SetupAssistantJavaTest`)
	})

	dir, found, err := javaHomeAt(registry.CURRENT_USER, `Software\SetupAssistantJavaTest\JavaSoft`, "21")
	if err != nil {
		t.Fatal(err)
	}
	if !found || dir != `E:\Dev\Java\21` {
		t.Fatalf("java = %s found = %v", dir, found)
	}
}

func TestInstallDirFromImageSkipsBin(t *testing.T) {
	got := installDirFromImage(`"E:\Database\MySQL\bin\mysqld.exe" --defaults-file="C:\my.ini" MySQL80`)
	if got != `E:\Database\MySQL` {
		t.Fatalf("dir = %s", got)
	}
}

func TestAppDirUsesDisplayIcon(t *testing.T) {
	const base = `Software\SetupAssistantUninstallTest`
	exe := filepath.Join(t.TempDir(), "notepad++.exe")
	if err := os.WriteFile(exe, []byte("notepad"), 0o644); err != nil {
		t.Fatal(err)
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\Notepad`, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	if err := key.SetStringValue("DisplayName", "Notepad++ (64-bit x64)"); err != nil {
		key.Close()
		t.Fatal(err)
	}
	if err := key.SetStringValue("DisplayIcon", exe+",0"); err != nil {
		key.Close()
		t.Fatal(err)
	}
	key.Close()
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, base+`\Notepad`)
		_ = registry.DeleteKey(registry.CURRENT_USER, base)
	})

	dir, found, err := appDirIn(registry.CURRENT_USER, base, "Notepad++")
	if err != nil {
		t.Fatal(err)
	}
	if !found || dir != filepath.Dir(exe) {
		t.Fatalf("dir = %s found = %v", dir, found)
	}

	if err := os.Remove(exe); err != nil {
		t.Fatal(err)
	}
	_, found, err = appDirIn(registry.CURRENT_USER, base, "Notepad++")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("removed program should not count as installed")
	}
}
