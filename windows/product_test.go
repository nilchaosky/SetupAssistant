package windows

import (
	"path/filepath"
	"testing"
)

func TestInstallDirUsesLocationThenIcon(t *testing.T) {
	if got := installDir(`E:\Dev\Python\2.7.18\`, `D:\ignored\python.exe`); got != `E:\Dev\Python\2.7.18` {
		t.Fatalf("location = %s", got)
	}
	if got := installDir("", `D:\Dev\PL\Python\Python27\python.exe`); got != `D:\Dev\PL\Python\Python27` {
		t.Fatalf("icon = %s", got)
	}
	if got := installDir("", `C:\App\app.exe,0`); got != `C:\App` {
		t.Fatalf("icon index = %s", got)
	}
}

func TestPython27ProductCode(t *testing.T) {
	code, err := productCode(filepath.Join("..", "resource", "python", "python-2.7.18.msi"))
	if err != nil {
		t.Fatal(err)
	}
	if code != "{A5F504DF-2ED9-4A2D-A2F3-9D2750DD42D5}" {
		t.Fatalf("code = %s", code)
	}
}
