package terminal

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nilchaosky/SetupAssistant/windows"
)

func TestMenuMatchesAppJSON(t *testing.T) {
	picker := testPicker(t)
	got := picker.categoryNames()
	want := []string{"脚本语言", "数据库", "编辑器"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v", got)
	}
	if !reflect.DeepEqual(picker.catalog[0].itemNames(), []string{"Java", "Go", "Python"}) {
		t.Fatalf("languages = %#v", picker.catalog[0].itemNames())
	}
	if picker.catalog[1].Items[0].Name != "MySQL" || picker.catalog[1].Items[1].Name != "PostgreSQL" {
		t.Fatalf("database = %#v", picker.catalog[1].Items)
	}
	if picker.catalog[2].Items[0].Name != "Notepad++" {
		t.Fatalf("tools = %#v", picker.catalog[2].Items)
	}
}

func TestMenuEscDoesNotFinishAndF5Installs(t *testing.T) {
	picker := press(testPicker(t), tea.KeyMsg{Type: tea.KeyEsc})
	if picker.done || picker.aborted || picker.screen != screenMenu {
		t.Fatalf("esc on menu should stay, screen=%d done=%v aborted=%v", picker.screen, picker.done, picker.aborted)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	if picker.done || picker.screen != screenMenu || !strings.Contains(picker.View(), "安装列表为空") {
		t.Fatalf("empty F5 should stay and warn, screen=%d done=%v\n%s", picker.screen, picker.done, picker.View())
	}

	picker = press(picker,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	if !picker.done || picker.screen == screenDir {
		t.Fatalf("F5 should start install, screen=%d done=%v", picker.screen, picker.done)
	}
}

func TestConfirmLanguageOpensListAndEscReturns(t *testing.T) {
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if picker.screen != screenItems {
		t.Fatalf("screen = %d, want item list", picker.screen)
	}
	view := picker.View()
	if !strings.Contains(view, "Java") || !strings.Contains(view, "Go") {
		t.Fatalf("language list missing:\n%s", view)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.screen != screenMenu || picker.done {
		t.Fatalf("esc should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}
}

func TestConfirmToolOpensVersionFromJSON(t *testing.T) {
	root := existingRoot(t)
	useDrive(t, root)
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if picker.screen != screenVersions {
		t.Fatalf("screen = %d, want versions", picker.screen)
	}
	view := picker.View()
	if !strings.Contains(view, "v8.9.8") {
		t.Fatalf("version missing:\n%s", view)
	}

	picker = press(picker,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if picker.screen != screenDir || !strings.Contains(picker.View(), `Notepad++`) {
		t.Fatalf("version should ask for a directory:\n%s", picker.View())
	}
	picker = press(picker,
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if !reflect.DeepEqual(picker.selected, []PendingInstall{{
		Name:    "Notepad++",
		Version: "v8.9.8",
		Package: "npp.8.9.8.Installer.x64.exe",
		Dir:     filepath.Join(root, "Editor", "Notepad++"),
		URL:     "https://github.com/notepad-plus-plus/notepad-plus-plus/releases/download/v8.9.8/npp.8.9.8.Installer.x64.exe",
	}}) {
		t.Fatalf("selected = %#v", picker.selected)
	}
	if picker.screen != screenMenu || picker.done {
		t.Fatalf("adding should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}
}

func TestBottomNoticeStaysAfterSelection(t *testing.T) {
	useDrive(t, existingRoot(t))
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyDown},
	)
	view := picker.View()
	if !strings.Contains(view, "tab 查看已选") {
		t.Fatalf("help line missing:\n%s", view)
	}
	if strings.Contains(view, "已加入") {
		t.Fatalf("added notice should be removed:\n%s", view)
	}
}

func TestSelectedModulesCanBeListedAndCancelled(t *testing.T) {
	useDrive(t, existingRoot(t))
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyTab},
	)
	if picker.screen != screenSelected {
		t.Fatalf("screen = %d, want selected list", picker.screen)
	}
	if !reflect.DeepEqual(picker.selected, []PendingInstall{
		{Name: "Go"},
		{
			Name:    "MySQL",
			Version: "8.0.46",
			Package: "mysql-installer-community-8.0.46.0.msi",
			Dir:     filepath.Join(existingRoot(t), "Database", "MySQL"),
			URL:     "https://cdn.mysql.com/Downloads/MySQLInstaller/mysql-installer-community-8.0.46.0.msi",
		},
	}) {
		t.Fatalf("selected = %#v", picker.selected)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !reflect.DeepEqual(picker.selected, []PendingInstall{{
		Name:    "MySQL",
		Version: "8.0.46",
		Package: "mysql-installer-community-8.0.46.0.msi",
		Dir:     filepath.Join(existingRoot(t), "Database", "MySQL"),
		URL:     "https://cdn.mysql.com/Downloads/MySQLInstaller/mysql-installer-community-8.0.46.0.msi",
	}}) {
		t.Fatalf("after cancel = %#v", picker.selected)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.screen != screenMenu || picker.done {
		t.Fatalf("esc should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}
}

func TestTwoJavasChooseHomeBeforeInstall(t *testing.T) {
	picker := testPicker(t)
	picker.selected = []PendingInstall{
		{Name: "Java", Version: "8", Dir: "Dev/Java/8"},
		{Name: "Java", Version: "21", Dir: "Dev/Java/21"},
	}
	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	if picker.done || picker.screen != screenJavaHome {
		t.Fatalf("two javas should ask for JAVA_HOME, screen=%d done=%v", picker.screen, picker.done)
	}
	view := picker.View()
	if !strings.Contains(view, "Java 8") || !strings.Contains(view, "Java 21") {
		t.Fatalf("java home choices missing:\n%s", view)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEnter})
	if picker.done || !strings.Contains(picker.View(), "请选择 JAVA_HOME") {
		t.Fatalf("empty confirm should stay, done=%v\n%s", picker.done, picker.View())
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.done || picker.screen != screenMenu {
		t.Fatalf("esc should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	picker.checked = 1
	picker = press(picker, tea.KeyMsg{Type: tea.KeyEnter})
	if !picker.done || picker.selected[0].JavaHome || !picker.selected[1].JavaHome {
		t.Fatalf("chosen home = %#v", picker.selected)
	}
}

func TestOneJavaBecomesHomeWithoutPrompt(t *testing.T) {
	picker := testPicker(t)
	picker.selected = []PendingInstall{
		{Name: "Java", Version: "8", Dir: "Dev/Java/8"},
		{Name: "Go"},
	}
	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	if !picker.done || picker.screen == screenJavaHome {
		t.Fatalf("one java should finish, screen=%d done=%v", picker.screen, picker.done)
	}
	if !picker.selected[0].JavaHome || picker.selected[1].JavaHome {
		t.Fatalf("home = %#v", picker.selected)
	}
}

func TestVersionConfirmAsksForDirectory(t *testing.T) {
	useDrive(t, `D:\`)
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	view := picker.View()
	if picker.screen != screenDir || picker.dirHint != "Editor/Notepad++" || !strings.Contains(view, `D:\Editor\Notepad++`) {
		t.Fatalf("dir screen = %d hint = %q\n%s", picker.screen, picker.dirHint, view)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEnter})
	if picker.screen != screenDir || !strings.Contains(picker.View(), "请选择安装目录") {
		t.Fatalf("empty confirm should stay:\n%s", picker.View())
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.screen != screenVersions || picker.done {
		t.Fatalf("esc should return to versions, screen=%d done=%v", picker.screen, picker.done)
	}

	picker = press(picker,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if picker.screen != screenMenu || len(picker.selected) != 1 || picker.selected[0].Dir != `D:\Editor\Notepad++` {
		t.Fatalf("selected = %#v screen=%d", picker.selected, picker.screen)
	}
}

func TestJoinDriveDir(t *testing.T) {
	if got := joinDriveDir(`D:\`, "Jetbrains/IntelliJ IDEA"); got != `D:\Jetbrains\IntelliJ IDEA` {
		t.Fatalf("idea = %s", got)
	}
	if got := joinDriveDir(`D:\`, "Java/8"); got != `D:\Java\8` {
		t.Fatalf("java = %s", got)
	}
}

func useDrive(t *testing.T, root string) {
	t.Helper()
	orig := listFixedDrives
	listFixedDrives = func() ([]windows.Drive, error) {
		return []windows.Drive{{Path: root}}, nil
	}
	t.Cleanup(func() { listFixedDrives = orig })
}

func existingRoot(t *testing.T) string {
	t.Helper()
	vol := filepath.VolumeName(t.TempDir())
	if vol == "" {
		t.Fatal("temp dir has no volume")
	}
	return vol + `\`
}

func testPicker(t *testing.T) pickerModel {
	t.Helper()
	data, err := os.ReadFile("../app.json")
	if err != nil {
		t.Fatal(err)
	}
	picker, err := newPicker(data)
	if err != nil {
		t.Fatal(err)
	}
	return picker
}

func press(picker pickerModel, keys ...tea.Msg) pickerModel {
	var model tea.Model = picker
	for _, key := range keys {
		model, _ = model.Update(key)
	}
	return model.(pickerModel)
}

func (category appCategory) itemNames() []string {
	names := make([]string, len(category.Items))
	for i, item := range category.Items {
		names[i] = item.Name
	}
	return names
}
