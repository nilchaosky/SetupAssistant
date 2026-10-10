package terminal

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuMatchesAppJSON(t *testing.T) {
	picker := testPicker(t)
	got := picker.categoryNames()
	want := []string{"脚本语言", "数据库", "工具类"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v", got)
	}
	if !reflect.DeepEqual(picker.catalog[0].itemNames(), []string{"Java", "Go"}) {
		t.Fatalf("languages = %#v", picker.catalog[0].itemNames())
	}
	if picker.catalog[1].Items[0].Name != "mysql" {
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
	if picker.done || picker.screen != screenDrives || len(picker.drives) == 0 {
		t.Fatalf("F5 should open drives, screen=%d done=%v drives=%#v", picker.screen, picker.done, picker.drives)
	}
	view := picker.View()
	for _, drive := range picker.drives {
		if !strings.Contains(view, drive.Path) {
			t.Fatalf("drive %s missing:\n%s", drive.Path, view)
		}
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.done || picker.screen != screenMenu {
		t.Fatalf("esc should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyF5})
	picker = press(picker,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
	)
	if !picker.done || picker.installDir != picker.drives[0].Path {
		t.Fatalf("confirm drive, done=%v dir=%q drives=%#v", picker.done, picker.installDir, picker.drives)
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
	if !reflect.DeepEqual(picker.selected, []PendingInstall{{
		Name:    "Notepad++",
		Version: "v8.9.8",
		Package: "npp.8.9.8.Installer.x64.exe",
		Dir:     "Editor/Notepad++",
		URL:     "https://github.com/notepad-plus-plus/notepad-plus-plus/releases/download/v8.9.8/npp.8.9.8.Installer.x64.exe",
	}}) {
		t.Fatalf("selected = %#v", picker.selected)
	}
	if picker.screen != screenMenu || picker.done {
		t.Fatalf("adding should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}
}

func TestBottomNoticeStaysAfterSelection(t *testing.T) {
	picker := press(testPicker(t),
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}},
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
		tea.KeyMsg{Type: tea.KeyTab},
	)
	if picker.screen != screenSelected {
		t.Fatalf("screen = %d, want selected list", picker.screen)
	}
	if !reflect.DeepEqual(picker.selected, []PendingInstall{{Name: "Go"}, {Name: "mysql"}}) {
		t.Fatalf("selected = %#v", picker.selected)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !reflect.DeepEqual(picker.selected, []PendingInstall{{Name: "mysql"}}) {
		t.Fatalf("after cancel = %#v", picker.selected)
	}

	picker = press(picker, tea.KeyMsg{Type: tea.KeyEsc})
	if picker.screen != screenMenu || picker.done {
		t.Fatalf("esc should return to menu, screen=%d done=%v", picker.screen, picker.done)
	}
}

func testPicker(t *testing.T) pickerModel {
	t.Helper()
	picker, err := newPicker()
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
