// Package terminal 从 app.json 读取软件清单，让用户勾选软件和版本。
// 选完版本后选择盘符，拼上该软件自己的安装目录。主菜单用 F5 开始安装；安装列表为空时只提示。
// 选中两个及以上 Java 版本时，开始安装前还要选择哪一个作为 JAVA_HOME。
package terminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilchaosky/SetupAssistant/windows"
)

const (
	screenMenu = iota
	screenItems
	screenVersions
	screenSelected
	screenDir
	screenJavaHome
)

type appVersion struct {
	Version string `json:"version"`
	Package string `json:"package"`
	Dir     string `json:"dir"`
	URL     string `json:"url"`
}

type appItem struct {
	Name     string       `json:"name"`
	Versions []appVersion `json:"versions"`
}

type appCategory struct {
	Name  string    `json:"name"`
	Items []appItem `json:"items"`
}

// PendingInstall 是一条待下载、待安装的软件，字段与 app.json 对应。
type PendingInstall struct {
	Name     string // 软件名
	Version  string // 版本。latest 也会作为一行显示
	Package  string // 安装包。下载文件名，或 resource/ 下的嵌入文件
	Dir      string // 安装目录。盘符拼上 app.json 里的目录
	URL      string // 下载地址。为空则跳过下载
	JavaHome bool   // 为 true 时，这个版本安装成功后写成 JAVA_HOME
}

// SelectInstallTarget 打开清单界面，返回已选软件。每条软件自带安装目录。
// data 是 app.json 的内容。用户取消时返回 huh.ErrUserAborted。
func SelectInstallTarget(data []byte) ([]PendingInstall, error) {
	picker, err := newPicker(data)
	if err != nil {
		return nil, err
	}
	program := tea.NewProgram(picker, tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrInterrupted) {
			return nil, huh.ErrUserAborted
		}
		return nil, err
	}

	result := final.(pickerModel)
	if result.aborted {
		return nil, huh.ErrUserAborted
	}
	return append([]PendingInstall(nil), result.selected...), nil
}

type pickerModel struct {
	catalog       []appCategory
	categoryIndex int
	itemIndex     int
	screen        int
	returnScreen  int
	cursor        int
	checked       int
	width         int
	height        int
	errMsg        string
	notice        string
	aborted       bool
	done          bool
	selected      []PendingInstall
	draft         PendingInstall
	drives        []windows.Drive
	dirHint       string
}

func newPicker(data []byte) (pickerModel, error) {
	catalog, err := loadCatalog(data)
	if err != nil {
		return pickerModel{}, err
	}
	return pickerModel{
		catalog: catalog,
		checked: -1,
		width:   80,
		height:  24,
	}, nil
}

func loadCatalog(data []byte) ([]appCategory, error) {
	var catalog []appCategory
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("解析 app.json 失败: %w", err)
	}
	return catalog, nil
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.aborted = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.screen == screenMenu {
				return m, nil
			}
			m.back()
			return m, nil
		case tea.KeyTab:
			m.openSelected()
		case tea.KeyUp:
			m.move(-1)
		case tea.KeyDown:
			m.move(1)
		case tea.KeySpace:
			m.toggle()
		case tea.KeyF5:
			if m.screen != screenMenu {
				return m, nil
			}
			if len(m.selected) == 0 {
				m.setStatus("", "安装列表为空")
				return m, nil
			}
			m.startInstall()
			if m.done {
				return m, tea.Quit
			}
			return m, nil
		case tea.KeyEnter:
			m.confirm()
			if m.done {
				return m, tea.Quit
			}
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case " ", "\u00a0":
				m.toggle()
			case "q":
				m.aborted = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	switch m.screen {
	case screenItems:
		category := m.catalog[m.categoryIndex]
		return m.optionView("选择"+category.Name, category.Name, m.itemNames(), "↑↓ 移动    空格 勾选    回车 确认    tab 查看已选    esc 返回")
	case screenVersions:
		item := m.currentItem()
		return m.optionView("选择"+item.Name+"版本", "版本", m.versionNames(), "↑↓ 移动    空格 勾选    回车 确认    tab 查看已选    esc 返回")
	case screenSelected:
		return m.selectedView()
	case screenDir:
		return m.dirView()
	case screenJavaHome:
		return m.optionView("选择 JAVA_HOME", "版本", m.javaHomeNames(), "↑↓ 移动    空格 勾选    回车 确认    esc 返回")
	default:
		return m.optionView("选择要安装的内容", "选项", m.categoryNames(), "↑↓ 移动    空格 勾选    回车 确认    tab 查看已选    F5 安装")
	}
}

func (m pickerModel) optionView(title, header string, options []string, help string) string {
	lines := make([]string, len(options))
	for i, name := range options {
		lines[i] = rowLine(i == m.cursor, m.checked == i, name)
	}
	return m.frame(title, header, lines, help)
}

func (m pickerModel) selectedView() string {
	lines := make([]string, 0, len(m.selected))
	if len(m.selected) == 0 {
		lines = append(lines, mutedStyle.Render("  还没有选择安装模块"))
	}
	for i, item := range m.selected {
		lines = append(lines, rowLine(i == m.cursor, true, item.listLabel()))
	}
	return m.frame("已选择的安装模块", "模块", lines, "↑↓ 移动    空格 取消    tab 返回    esc 返回")
}

func (m pickerModel) frame(title, header string, lines []string, help string) string {
	ruleWidth := max(20, m.width-2)
	status := ""
	switch {
	case m.errMsg != "":
		status = errStyle.Render(m.errMsg)
	case m.notice != "":
		status = noticeStyle.Render(m.notice)
	}
	parts := []string{
		titleStyle.Render(title),
		"",
		headerStyle.Render(header),
		mutedStyle.Render(strings.Repeat("─", ruleWidth)),
		strings.Join(lines, "\n"),
		"",
		status,
		helpStyle.Render(help),
		"",
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *pickerModel) move(delta int) {
	count := len(m.currentOptions())
	if count == 0 {
		return
	}
	m.cursor = (m.cursor + delta + count) % count
}

func (m pickerModel) currentOptions() []string {
	switch m.screen {
	case screenItems:
		return m.itemNames()
	case screenVersions:
		return m.versionNames()
	case screenSelected:
		return m.selectedLabels()
	case screenDir:
		return m.dirOptions()
	case screenJavaHome:
		return m.javaHomeNames()
	default:
		return m.categoryNames()
	}
}

func (m pickerModel) categoryNames() []string {
	names := make([]string, len(m.catalog))
	for i, category := range m.catalog {
		names[i] = category.Name
	}
	return names
}

func (m pickerModel) itemNames() []string {
	items := m.catalog[m.categoryIndex].Items
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}

func (m pickerModel) selectedLabels() []string {
	labels := make([]string, len(m.selected))
	for i, item := range m.selected {
		labels[i] = item.listLabel()
	}
	return labels
}

func (item PendingInstall) label() string {
	if item.Version == "" {
		return item.Name
	}
	return item.Name + " " + item.Version
}

func (item PendingInstall) listLabel() string {
	if item.Dir == "" {
		return item.label()
	}
	return item.label() + "  " + item.Dir
}

func (m pickerModel) versionChoices() []appVersion {
	choices := make([]appVersion, 0)
	for _, version := range m.currentItem().Versions {
		if version.Version == "" {
			continue
		}
		choices = append(choices, version)
	}
	return choices
}

func (m pickerModel) versionNames() []string {
	choices := m.versionChoices()
	names := make([]string, len(choices))
	for i, version := range choices {
		names[i] = version.Version
	}
	return names
}

func (m pickerModel) currentItem() appItem {
	return m.catalog[m.categoryIndex].Items[m.itemIndex]
}

func (m *pickerModel) toggle() {
	if m.screen == screenSelected {
		m.cancelSelected()
		return
	}
	if m.checked == m.cursor {
		m.checked = -1
		return
	}
	m.checked = m.cursor
}

func (m *pickerModel) confirm() {
	switch m.screen {
	case screenMenu:
		m.confirmMenu()
	case screenItems:
		m.confirmItem()
	case screenVersions:
		m.confirmVersion()
	case screenDir:
		m.confirmDir()
	case screenJavaHome:
		m.confirmJavaHome()
	}
}

func (m *pickerModel) confirmMenu() {
	if m.checked < 0 || m.checked >= len(m.catalog) {
		m.setStatus("", "请选择要安装的内容")
		return
	}
	if len(m.catalog[m.checked].Items) == 0 {
		m.addSelected(PendingInstall{Name: m.catalog[m.checked].Name})
		return
	}
	m.categoryIndex = m.checked
	m.screen = screenItems
	m.cursor = 0
	m.checked = -1
}

func (m *pickerModel) confirmItem() {
	items := m.catalog[m.categoryIndex].Items
	if m.checked < 0 || m.checked >= len(items) {
		m.setStatus("", "请选择"+m.catalog[m.categoryIndex].Name)
		return
	}
	m.itemIndex = m.checked
	if len(m.versionNames()) == 0 {
		m.addSelected(PendingInstall{Name: items[m.checked].Name})
		return
	}
	m.screen = screenVersions
	m.cursor = 0
	m.checked = -1
}

func (m *pickerModel) confirmVersion() {
	versions := m.versionChoices()
	if m.checked < 0 || m.checked >= len(versions) {
		m.setStatus("", "请选择版本")
		return
	}
	chosen := versions[m.checked]
	m.draft = PendingInstall{
		Name:    m.currentItem().Name,
		Version: chosen.Version,
		Package: chosen.Package,
		URL:     chosen.URL,
	}
	if slashHint(chosen.Dir) == "" {
		m.addSelected(m.draft)
		return
	}
	drives, err := listFixedDrives()
	if err != nil {
		m.setStatus("", "读取盘符失败")
		return
	}
	if len(drives) == 0 {
		m.setStatus("", "没有可用盘符")
		return
	}
	m.drives = drives
	m.dirHint = chosen.Dir
	m.screen = screenDir
	m.cursor = 0
	m.checked = -1
	m.setStatus("", "")
}

var listFixedDrives = windows.ListDrives

func (m pickerModel) dirView() string {
	return m.optionView("选择"+m.draft.label()+"安装目录", "目录", m.dirOptions(), "↑↓ 移动    空格 勾选    回车 确认    esc 返回")
}

func (m pickerModel) dirOptions() []string {
	options := make([]string, len(m.drives))
	for i, drive := range m.drives {
		options[i] = joinDriveDir(drive.Path, m.dirHint)
	}
	return options
}

func slashHint(hint string) string {
	hint = strings.Trim(strings.TrimSpace(hint), `\/`)
	if hint == "" {
		return ""
	}
	return filepath.FromSlash(hint)
}

// joinDriveDir 把盘符根路径和 app.json 里的目录拼成安装目录。
func joinDriveDir(root, hint string) string {
	hint = slashHint(hint)
	if hint == "" {
		return root
	}
	return filepath.Join(root, hint)
}

func (m *pickerModel) confirmDir() {
	options := m.dirOptions()
	if m.checked < 0 || m.checked >= len(options) {
		m.setStatus("", "请选择安装目录")
		return
	}
	m.draft.Dir = options[m.checked]
	m.addSelected(m.draft)
}

func (m *pickerModel) startInstall() {
	if len(m.javaIndexes()) >= 2 {
		m.screen = screenJavaHome
		m.cursor = 0
		m.checked = -1
		m.setStatus("", "")
		return
	}
	m.finishInstall()
}

func (m *pickerModel) confirmJavaHome() {
	indexes := m.javaIndexes()
	if m.checked < 0 || m.checked >= len(indexes) {
		m.setStatus("", "请选择 JAVA_HOME")
		return
	}
	for i := range m.selected {
		m.selected[i].JavaHome = false
	}
	m.selected[indexes[m.checked]].JavaHome = true
	m.done = true
}

// finishInstall 在只有一个 Java 时把它标成 JAVA_HOME，然后结束选择。
func (m *pickerModel) finishInstall() {
	indexes := m.javaIndexes()
	for i := range m.selected {
		m.selected[i].JavaHome = false
	}
	if len(indexes) == 1 {
		m.selected[indexes[0]].JavaHome = true
	}
	m.done = true
}

func (m pickerModel) javaIndexes() []int {
	indexes := make([]int, 0)
	for i, item := range m.selected {
		if item.Name == "Java" {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (m pickerModel) javaHomeNames() []string {
	indexes := m.javaIndexes()
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = m.selected[index].label()
	}
	return names
}

func (m *pickerModel) addSelected(item PendingInstall) {
	if containsInstall(m.selected, item) {
		m.setStatus("", item.label()+" 已在安装列表中")
		return
	}
	m.selected = append(m.selected, item)
	m.setStatus("", "")
	m.screen = screenMenu
	m.cursor = 0
	m.checked = -1
}

func (m *pickerModel) cancelSelected() {
	if m.cursor < 0 || m.cursor >= len(m.selected) {
		return
	}
	name := m.selected[m.cursor].label()
	m.selected = append(m.selected[:m.cursor], m.selected[m.cursor+1:]...)
	if m.cursor >= len(m.selected) {
		m.cursor = len(m.selected) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.setStatus("已取消："+name, "")
}

func (m *pickerModel) setStatus(notice, errMsg string) {
	m.notice = notice
	m.errMsg = errMsg
}

func (m *pickerModel) openSelected() {
	if m.screen == screenSelected {
		m.back()
		return
	}
	m.returnScreen = m.screen
	m.screen = screenSelected
	m.cursor = 0
}

func (m *pickerModel) back() {
	switch m.screen {
	case screenSelected:
		m.screen = m.returnScreen
	case screenDir:
		m.screen = screenVersions
		m.setStatus("", "")
	case screenVersions:
		m.screen = screenItems
	case screenJavaHome:
		m.screen = screenMenu
		m.setStatus("", "")
	default:
		m.screen = screenMenu
	}
	m.cursor = 0
	m.checked = -1
}

func containsInstall(items []PendingInstall, target PendingInstall) bool {
	for _, item := range items {
		if item.Name == target.Name && item.Version == target.Version {
			return true
		}
	}
	return false
}

func rowLine(cursor, on bool, label string) string {
	marker := "  "
	if cursor {
		marker = cursorStyle.Render(">") + " "
	}
	return marker + checkbox(on, label, cursor)
}

func checkbox(on bool, label string, active bool) string {
	mark := "[ ]"
	if on {
		mark = "[x]"
	}
	text := mark + " " + label
	if active {
		return activeStyle.Render(text)
	}
	return text
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	headerStyle = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	noticeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
