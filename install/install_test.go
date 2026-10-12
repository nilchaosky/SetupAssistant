package install

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nilchaosky/SetupAssistant/install/mode"
	"github.com/nilchaosky/SetupAssistant/terminal"
	"github.com/nilchaosky/SetupAssistant/windows"
)

type nopInstallBar struct{}

func (nopInstallBar) Finish() {}
func (nopInstallBar) Abort()  {}

var recordedEnv []string

func TestMain(m *testing.M) {
	mode.StartBar = func(string) mode.Bar { return nopInstallBar{} }
	mode.ApplyEnv = func(name, dest string) error {
		recordedEnv = append(recordedEnv, name+" "+dest)
		return nil
	}
	mode.PortInUse = func(string) (bool, error) { return false, nil }
	mode.InstalledProduct = func(string) (string, bool, error) { return "", false, nil }
	mode.FindJava = func(string) (string, bool, error) { return "", false, nil }
	mode.FindPython = func(string) (string, bool, error) { return "", false, nil }
	mode.FindApp = func(string) (string, bool, error) { return "", false, nil }
	mode.FindMySQLServer = func(string) (string, bool, error) { return "", false, nil }
	mode.FindPostgreSQL = func(string) (string, bool, error) { return "", false, nil }
	mode.OpenConfig = func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join("..", "config", name))
	}
	os.Exit(m.Run())
}

func TestRunSkipsEmptyPackageAndInstallsNotepad(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "npp.8.9.8.Installer.x64.exe")
	if err := os.WriteFile(exe, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs, gotTarget, gotName string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	origLink := mode.Shortcut
	mode.Shortcut = func(target, name string) error {
		gotTarget = target
		gotName = name
		return nil
	}
	t.Cleanup(func() {
		mode.Start = orig
		mode.Shortcut = origLink
	})

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
	if gotTarget != `D:\Editor\Notepad++\notepad++.exe` || gotName != "Notepad++" {
		t.Fatalf("shortcut = %s %s", gotTarget, gotName)
	}
}

func TestJetBrainsUsesRootSilentConfig(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "IntelliJ IDEA.exe")
	if err := os.WriteFile(exe, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotArgs, configText string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotArgs = params
		data, err := os.ReadFile(mode.JetBrainsConfigPath())
		if err != nil {
			t.Fatal(err)
		}
		configText = string(data)
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	recordedEnv = nil
	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "IntelliJ IDEA",
		Package: "IntelliJ IDEA.exe",
		Dir:     "Editor/Jetbrains/IntelliJ IDEA",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "jetbrains-silent.config")); !os.IsNotExist(err) {
		t.Fatal("should not write jetbrains-silent.config next to the installer")
	}
	if !strings.Contains(configText, "launcher64=1") || !strings.Contains(configText, "updatePATH=1") {
		t.Fatalf("config = %s", configText)
	}
	if _, err := os.Stat(mode.JetBrainsConfigPath()); !os.IsNotExist(err) {
		t.Fatal("jetbrains config should be removed after install")
	}
	want := "/S /CONFIG=" + mode.JetBrainsConfigPath() + ` /D=D:\Editor\Jetbrains\IntelliJ IDEA`
	if gotArgs != want {
		t.Fatalf("args = %s", gotArgs)
	}
	if len(recordedEnv) != 0 {
		t.Fatalf("env = %v", recordedEnv)
	}
}

func TestInnoKeepsPathAndDesktopIcon(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "VS Code.exe")
	if err := os.WriteFile(exe, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotArgs string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "VS Code",
		Package: "VS Code.exe",
		Dir:     "Editor/VS Code",
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := `/VERYSILENT /NORESTART /MERGETASKS=addtopath,desktopicon,!runcode,!associatewithfiles /DIR="D:\Editor\VS Code"`
	if gotArgs != want {
		t.Fatalf("args = %s", gotArgs)
	}
}

func TestMSIUsesMsiexec(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "mysql-installer-community-8.0.46.0.msi")
	if err := os.WriteFile(msi, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	console := `C:\Program Files (x86)\MySQL\MySQL Installer for Windows\MySQLInstallerConsole.exe`
	var got [][2]string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		got = append(got, [2]string{exe, params})
		return nil
	}
	origConsole := mode.FindMySQLConsole
	mode.FindMySQLConsole = func() (string, error) { return console, nil }
	t.Cleanup(func() {
		mode.Start = orig
		mode.FindMySQLConsole = origConsole
	})

	recordedEnv = nil
	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "MySQL",
		Package: "mysql-installer-community-8.0.46.0.msi",
		Dir:     "Database/MySQL",
	}})
	if err != nil {
		t.Fatal(err)
	}
	absMSI, err := filepath.Abs(msi)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("starts = %d", len(got))
	}
	if got[0][0] != mode.MsiexecPath() || got[0][1] != mode.MySQLInstallerArgs(absMSI) {
		t.Fatalf("msiexec = %s %s", got[0][0], got[0][1])
	}
	want := mode.MySQLServerArgs("8.0.46", "3306", "123456", `D:\Database\MySQL`)
	if got[1][0] != console || got[1][1] != want {
		t.Fatalf("console = %s %s", got[1][0], got[1][1])
	}
	if strings.Contains(got[1][1], "type=config") || strings.Contains(got[1][1], "root_passwd=") {
		t.Fatalf("old console syntax: %s", got[1][1])
	}
	if !strings.Contains(got[1][1], `install_dir="D:\Database\MySQL"`) {
		t.Fatalf("install dir must stay quoted: %s", got[1][1])
	}
	if len(recordedEnv) != 1 || recordedEnv[0] != `MySQL D:\Database\MySQL` {
		t.Fatalf("env = %v", recordedEnv)
	}
}

func TestPostgreSQLUsesUnattendedInstall(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "postgresql-18.6-5-windows-x64.exe")
	if err := os.WriteFile(exe, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	recordedEnv = nil
	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "PostgreSQL",
		Package: "postgresql-18.6-5-windows-x64.exe",
		Dir:     "Database/PostgreSQL",
	}})
	if err != nil {
		t.Fatal(err)
	}
	absExe, err := filepath.Abs(exe)
	if err != nil {
		t.Fatal(err)
	}
	if gotExe != absExe {
		t.Fatalf("exe = %s", gotExe)
	}
	want := `--mode unattended --unattendedmodeui none --prefix "D:\Database\PostgreSQL" --superpassword "123456" --serverport 5432 --disable-components stackbuilder`
	if gotArgs != want {
		t.Fatalf("args = %s", gotArgs)
	}
	if len(recordedEnv) != 1 || recordedEnv[0] != `PostgreSQL D:\Database\PostgreSQL` {
		t.Fatalf("env = %v", recordedEnv)
	}
}

func TestMSITreatsRebootRequestAsSuccess(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "mysql-installer-community-8.0.46.0.msi")
	if err := os.WriteFile(msi, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := mode.Start
	calls := 0
	mode.Start = func(exe, params string) error {
		calls++
		if calls == 1 {
			return &windows.ExitError{Code: 3010}
		}
		return nil
	}
	origConsole := mode.FindMySQLConsole
	mode.FindMySQLConsole = func() (string, error) { return `C:\MySQLInstallerConsole.exe`, nil }
	t.Cleanup(func() {
		mode.Start = orig
		mode.FindMySQLConsole = origConsole
	})

	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "MySQL",
		Package: "mysql-installer-community-8.0.46.0.msi",
		Dir:     "Database/MySQL",
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMySQLRejectsOccupiedPort(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "mysql-installer-community-8.0.46.0.msi")
	if err := os.WriteFile(msi, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := false
	orig := mode.Start
	mode.Start = func(string, string) error {
		started = true
		return nil
	}
	origPort := mode.PortInUse
	mode.PortInUse = func(port string) (bool, error) {
		if port != "3306" {
			t.Fatalf("port = %s", port)
		}
		return true, nil
	}
	t.Cleanup(func() {
		mode.Start = orig
		mode.PortInUse = origPort
	})

	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "MySQL",
		Version: "8",
		Package: "mysql-installer-community-8.0.46.0.msi",
		Dir:     "Database/MySQL",
	}})
	if err == nil || err.Error() != "MySQL 8: 端口 3306 已被占用" {
		t.Fatalf("err = %v", err)
	}
	if started {
		t.Fatal("installer started")
	}
}

func TestJavaInstallsIntoChosenDirectory(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "jdk-8u202-windows-x64.exe")
	if err := os.WriteFile(exe, []byte("jdk"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	recordedEnv = nil
	err := Run(dir, `E:\`, []terminal.PendingInstall{{
		Name:     "Java",
		Version:  "8",
		Package:  "jdk-8u202-windows-x64.exe",
		Dir:      "Language/Java",
		JavaHome: true,
	}})
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
	if gotArgs != `/s ADDLOCAL="ToolsFeature,SourceFeature" INSTALLDIR="E:\Language\Java"` {
		t.Fatalf("args = %s", gotArgs)
	}
	if len(recordedEnv) != 1 || recordedEnv[0] != `Java E:\Language\Java` {
		t.Fatalf("env = %v", recordedEnv)
	}
}

func TestOnlyChosenJavaHomeIsConfiguredAfterInstalls(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"jdk-8u202-windows-x64.exe", "jdk-21_windows-x64_bin.exe", "python-3.13.16-amd64.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("installer"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	orig := mode.Start
	mode.Start = func(exe, params string) error { return nil }
	t.Cleanup(func() { mode.Start = orig })

	recordedEnv = nil
	err := Run(dir, `E:\`, []terminal.PendingInstall{
		{Name: "Java", Version: "8", Package: "jdk-8u202-windows-x64.exe", Dir: "Dev/Java/8"},
		{Name: "Java", Version: "21", Package: "jdk-21_windows-x64_bin.exe", Dir: "Dev/Java/21", JavaHome: true},
		{Name: "Python", Version: "3.13.16", Package: "python-3.13.16-amd64.exe", Dir: "Dev/Python/3.13.16"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`Java E:\Dev\Java\21`}
	if !reflect.DeepEqual(recordedEnv, want) {
		t.Fatalf("env = %v", recordedEnv)
	}
}

func TestPython3InstallsQuietly(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "python-3.13.16-amd64.exe")
	if err := os.WriteFile(exe, []byte("python"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	err := Run(dir, `E:\`, []terminal.PendingInstall{{
		Name:    "Python",
		Version: "3.13.16",
		Package: "python-3.13.16-amd64.exe",
		Dir:     "Dev/Python/3.13.16",
	}})
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
	if gotArgs != `/quiet InstallAllUsers=1 PrependPath=1 Include_pip=1 Include_test=0 TargetDir="E:\Dev\Python\3.13.16"` {
		t.Fatalf("args = %s", gotArgs)
	}
}

func TestExistingMSIProductIsReported(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "python-2.7.18.msi")
	if err := os.WriteFile(msi, []byte("python"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := mode.InstalledProduct
	mode.InstalledProduct = func(path string) (string, bool, error) {
		if path != msi && !strings.HasSuffix(path, "python-2.7.18.msi") {
			t.Fatalf("msi = %s", path)
		}
		return `D:\Dev\PL\Python\Python27`, true, nil
	}
	started := false
	origStart := mode.Start
	mode.Start = func(string, string) error {
		started = true
		return nil
	}
	t.Cleanup(func() {
		mode.InstalledProduct = orig
		mode.Start = origStart
	})

	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := Run(dir, `E:\`, []terminal.PendingInstall{{
		Name:    "Python",
		Version: "2.7.18",
		Package: "python-2.7.18.msi",
		Dir:     "Dev/Python/2.7.18",
	}})
	writer.Close()
	os.Stdout = stdout
	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("err = %v", runErr)
	}
	if started {
		t.Fatal("installer started")
	}
	if !strings.Contains(output.String(), "警告 Python 2.7.18：产品已安装于 D:\\Dev\\PL\\Python\\Python27") {
		t.Fatalf("output = %s", output.String())
	}
}

func TestInstalledCopyWarnsAndSkips(t *testing.T) {
	notFound := func(string) (string, bool, error) { return "", false, nil }
	cases := []struct {
		name    string
		version string
		pkg     string
		dir     string
		want    string
		arm     func()
	}{
		{
			name: "Java", version: "8", pkg: "jdk-8u202-windows-x64.exe", dir: "Dev/Java/8",
			want: `警告 Java 8：产品已安装于 D:\Dev\Java\8`,
			arm: func() {
				mode.FindJava = func(major string) (string, bool, error) {
					if major != "8" {
						t.Fatalf("java = %s", major)
					}
					return `D:\Dev\Java\8`, true, nil
				}
			},
		},
		{
			name: "Python", version: "3.13.16", pkg: "python-3.13.16-amd64.exe", dir: "Dev/Python/3.13.16",
			want: `警告 Python 3.13.16：产品已安装于 D:\Dev\Python\3.13`,
			arm: func() {
				mode.FindPython = func(series string) (string, bool, error) {
					if series != "3.13" {
						t.Fatalf("python = %s", series)
					}
					return `D:\Dev\Python\3.13`, true, nil
				}
			},
		},
		{
			name: "Notepad++", version: "v8.9.8", pkg: "npp.exe", dir: "Editor/Notepad++",
			want: `警告 Notepad++ v8.9.8：产品已安装于 D:\Editor\Notepad++`,
			arm: func() {
				mode.FindApp = func(prefix string) (string, bool, error) {
					if prefix != "Notepad++" {
						t.Fatalf("app = %s", prefix)
					}
					return `D:\Editor\Notepad++`, true, nil
				}
			},
		},
		{
			name: "VS Code", version: "latest", pkg: "VS Code.exe", dir: "Editor/VS Code",
			want: `警告 VS Code latest：产品已安装于 D:\Editor\VS Code`,
			arm: func() {
				mode.FindApp = func(prefix string) (string, bool, error) {
					if prefix != "Microsoft Visual Studio Code" {
						t.Fatalf("app = %s", prefix)
					}
					return `D:\Editor\VS Code`, true, nil
				}
			},
		},
		{
			name: "IntelliJ IDEA", version: "latest", pkg: "IntelliJ IDEA.exe", dir: "Editor/Jetbrains/IntelliJ IDEA",
			want: `警告 IntelliJ IDEA latest：产品已安装于 D:\Editor\Jetbrains\IntelliJ IDEA`,
			arm: func() {
				mode.FindApp = func(prefix string) (string, bool, error) {
					if prefix != "IntelliJ IDEA" {
						t.Fatalf("app = %s", prefix)
					}
					return `D:\Editor\Jetbrains\IntelliJ IDEA`, true, nil
				}
			},
		},
		{
			name: "MySQL", version: "8.0.46", pkg: "mysql-installer-community-8.0.46.0.msi", dir: "Database/MySQL",
			want: `警告 MySQL 8.0.46：产品已安装于 D:\Database\MySQL`,
			arm: func() {
				mode.FindMySQLServer = func(series string) (string, bool, error) {
					if series != "8.0" {
						t.Fatalf("mysql = %s", series)
					}
					return `D:\Database\MySQL`, true, nil
				}
			},
		},
		{
			name: "PostgreSQL", version: "18.6", pkg: "postgresql-18.6-5-windows-x64.exe", dir: "Database/PostgreSQL",
			want: `警告 PostgreSQL 18.6：产品已安装于 E:\Database\PostgreSQL`,
			arm: func() {
				mode.FindPostgreSQL = func(major string) (string, bool, error) {
					if major != "18" {
						t.Fatalf("postgres = %s", major)
					}
					return `E:\Database\PostgreSQL`, true, nil
				}
			},
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, item.pkg), []byte("installer"), 0o644); err != nil {
				t.Fatal(err)
			}
			mode.FindJava = notFound
			mode.FindPython = notFound
			mode.FindApp = notFound
			mode.FindMySQLServer = notFound
			mode.FindPostgreSQL = notFound
			item.arm()
			t.Cleanup(func() {
				mode.FindJava = notFound
				mode.FindPython = notFound
				mode.FindApp = notFound
				mode.FindMySQLServer = notFound
				mode.FindPostgreSQL = notFound
			})
			started := false
			orig := mode.Start
			mode.Start = func(string, string) error {
				started = true
				return nil
			}
			t.Cleanup(func() { mode.Start = orig })

			stdout := os.Stdout
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = writer
			t.Cleanup(func() { os.Stdout = stdout })
			runErr := Run(dir, `E:\`, []terminal.PendingInstall{{
				Name:    item.name,
				Version: item.version,
				Package: item.pkg,
				Dir:     item.dir,
			}})
			writer.Close()
			os.Stdout = stdout
			var output bytes.Buffer
			if _, err := io.Copy(&output, reader); err != nil {
				t.Fatal(err)
			}
			if runErr != nil {
				t.Fatalf("err = %v", runErr)
			}
			if started {
				t.Fatal("installer started")
			}
			if !strings.Contains(output.String(), item.want) {
				t.Fatalf("output = %s", output.String())
			}
		})
	}
}

func TestProductCheckErrorFailsInstall(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "python-2.7.18.msi")
	if err := os.WriteFile(msi, []byte("python"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := mode.InstalledProduct
	mode.InstalledProduct = func(string) (string, bool, error) {
		return "", false, errors.New("打不开安装包")
	}
	t.Cleanup(func() { mode.InstalledProduct = orig })

	err := Run(dir, `E:\`, []terminal.PendingInstall{{
		Name:    "Python",
		Version: "2.7.18",
		Package: "python-2.7.18.msi",
		Dir:     "Dev/Python/2.7.18",
	}})
	if err == nil || !strings.Contains(err.Error(), "检查是否已安装失败") {
		t.Fatalf("err = %v", err)
	}
}

func TestPython2UsesMSITargetDir(t *testing.T) {
	dir := t.TempDir()
	msi := filepath.Join(dir, "python-2.7.18.msi")
	if err := os.WriteFile(msi, []byte("python"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotExe, gotArgs string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		gotArgs = params
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	err := Run(dir, `E:\`, []terminal.PendingInstall{{
		Name:    "Python",
		Version: "2.7.18",
		Package: "python-2.7.18.msi",
		Dir:     "Dev/Python/2.7.18",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(gotExe, filepath.Join(os.Getenv("SystemRoot"), "System32", "msiexec.exe")) &&
		!strings.HasSuffix(strings.ToLower(gotExe), `\msiexec.exe`) {
		t.Fatalf("exe = %s", gotExe)
	}
	if !strings.Contains(gotArgs, "python-2.7.18.msi") || !strings.Contains(gotArgs, `TARGETDIR="E:\Dev\Python\2.7.18"`) || !strings.Contains(gotArgs, "ADDLOCAL=ALL") {
		t.Fatalf("args = %s", gotArgs)
	}
}

func TestResourcePackageIsWrittenBeforeInstall(t *testing.T) {
	dir := t.TempDir()
	origResource := openResource
	openResource = func(name string) ([]byte, error) {
		if name != "resource/java/jdk-8u202-windows-x64.exe" {
			t.Fatalf("resource = %s", name)
		}
		return []byte("jdk"), nil
	}
	var gotExe string
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		gotExe = exe
		return nil
	}
	t.Cleanup(func() {
		openResource = origResource
		mode.Start = orig
	})

	err := Run(dir, `D:\`, []terminal.PendingInstall{{
		Name:    "Java",
		Version: "8",
		Package: "resource/java/jdk-8u202-windows-x64.exe",
		Dir:     "Language/Java",
	}})
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(filepath.Join(dir, "jdk-8u202-windows-x64.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if gotExe != want {
		t.Fatalf("exe = %s", gotExe)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "jdk" {
		t.Fatalf("file = %q", data)
	}
}

func TestRunReportsMissingInstaller(t *testing.T) {
	orig := mode.Start
	mode.Start = func(exe, params string) error {
		t.Fatal("should not start")
		return nil
	}
	t.Cleanup(func() { mode.Start = orig })

	err := Run(t.TempDir(), `E:\`, []terminal.PendingInstall{{
		Name:    "Notepad++",
		Package: "npp.8.9.8.Installer.x64.exe",
	}})
	if err == nil || !strings.Contains(err.Error(), "找不到安装包") {
		t.Fatalf("err = %v", err)
	}
}
