// Package wizard provides the training configuration wizard UI
package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vigo999/ms-cli/internal/train"
)

// Mode represents the wizard mode
type Mode int

const (
	ModeSelectProject Mode = iota
	ModeSelectTrainMode
	ModeEditField
	ModeInputValue
	ModeConfirm
	ModeDone
)

// Model represents the wizard state
type Model struct {
	mode         Mode
	width        int
	height       int

	// Project selection
	projects     []train.TrainProject
	projectCursor int

	// Training mode selection
	trainMode    train.TrainMode
	modeCursor   int

	// Field editing
	currentProject *train.TrainProject
	originalProject *train.TrainProject
	fields       []train.ConfigField
	fieldCursor  int
	fieldOptions []string // ["保持当前配置", "输入新值"]
	fieldOptionCursor int

	// Input mode
	isInputMode  bool
	inputBuffer  string
	inputPrompt  string
	inputValidation func(string) error

	// Messages from app
	message      string
	errMsg       string
}

// New creates a new wizard model
func New(projects []train.TrainProject) Model {
	return Model{
		mode:              ModeSelectProject,
		projects:          projects,
		projectCursor:     0,
		modeCursor:        0,
		fieldCursor:       0,
		fieldOptionCursor: 0,
		fieldOptions:      []string{"保持当前配置", "输入新值"},
		trainMode:         train.ModeCompare,
	}
}

// Init initializes the wizard
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.mode {
	case ModeSelectProject:
		return m.handleSelectProjectKey(msg)
	case ModeSelectTrainMode:
		return m.handleSelectModeKey(msg)
	case ModeEditField:
		return m.handleEditFieldKey(msg)
	case ModeInputValue:
		return m.handleInputValueKey(msg)
	case ModeConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m Model) handleSelectProjectKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case "down", "j":
		// +1 for "重新设置一套全新配置" option
		maxCursor := len(m.projects)
		if m.projectCursor < maxCursor {
			m.projectCursor++
		}
	case "enter":
		if m.projectCursor < len(m.projects) {
			// Select existing project
			proj := m.projects[m.projectCursor]
			m.currentProject = &proj
			m.originalProject = copyProject(&proj)
			m.trainMode = proj.Mode
			m.fields = train.GetConfigFields(proj.Mode)
			m.mode = ModeEditField
		} else {
			// New project - go to mode selection
			m.mode = ModeSelectTrainMode
		}
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleSelectModeKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.modeCursor > 0 {
			m.modeCursor--
		}
	case "down", "j":
		if m.modeCursor < 2 { // 3 modes: Compare, NPUOnly, GPUOnly
			m.modeCursor++
		}
	case "enter":
		m.trainMode = train.TrainMode(m.modeCursor)
		m.currentProject = train.NewDefaultProject(m.trainMode)
		m.originalProject = nil
		m.fields = train.GetConfigFields(m.trainMode)
		m.mode = ModeEditField
	case "esc":
		m.mode = ModeSelectProject
	}
	return m, nil
}

func (m Model) handleEditFieldKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.fieldOptionCursor > 0 {
			m.fieldOptionCursor--
		}
	case "down", "j":
		if m.fieldOptionCursor < len(m.fieldOptions)-1 {
			m.fieldOptionCursor++
		}
	case "enter":
		if m.fieldOptionCursor == 0 {
			// Keep current - move to next field
			m.nextField()
		} else {
			// Enter new value
			m.isInputMode = true
			m.inputBuffer = ""
			m.mode = ModeInputValue
		}
	case "tab":
		// Skip to next field
		m.nextField()
	case "esc":
		if m.originalProject != nil {
			m.mode = ModeSelectProject
		} else {
			m.mode = ModeSelectTrainMode
		}
	}
	return m, nil
}

func (m Model) handleInputValueKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate and save
		if m.inputValidation != nil {
			if err := m.inputValidation(m.inputBuffer); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
		}
		// Save value and return to field edit
		m.saveFieldValue(m.inputBuffer)
		m.isInputMode = false
		m.mode = ModeEditField
		m.nextField()
	case "esc":
		// Cancel input
		m.isInputMode = false
		m.mode = ModeEditField
		m.errMsg = ""
	default:
		// Handle text input
		if msg.Type == tea.KeyRunes {
			m.inputBuffer += string(msg.Runes)
		} else if msg.String() == "backspace" {
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
		}
	}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = ModeDone
		return m, nil
	case "n", "N":
		m.mode = ModeEditField
	case "esc":
		m.mode = ModeEditField
	}
	return m, nil
}

func (m *Model) nextField() {
	m.fieldCursor++
	m.fieldOptionCursor = 0
	if m.fieldCursor >= len(m.fields) {
		// All fields done
		m.mode = ModeConfirm
	}
}

func (m *Model) saveFieldValue(value string) {
	if m.currentProject == nil || m.fieldCursor >= len(m.fields) {
		return
	}

	field := m.fields[m.fieldCursor]
	fieldValue := &train.FieldValue{StringValue: value}

	// Parse based on field type
	switch field.Type {
	case train.FieldTypeEnvVars:
		fieldValue.StringSlice = train.ParseEnvVars(value)
		fieldValue.StringValue = ""
	case train.FieldTypeGlobalConfig:
		// Global config is handled differently
		return
	}

	train.SetFieldValue(m.currentProject, field, fieldValue)
}

// View renders the wizard
func (m Model) View() string {
	switch m.mode {
	case ModeSelectProject:
		return m.viewSelectProject()
	case ModeSelectTrainMode:
		return m.viewSelectMode()
	case ModeEditField:
		return m.viewEditField()
	case ModeInputValue:
		return m.viewInputValue()
	case ModeConfirm:
		return m.viewConfirm()
	case ModeDone:
		return "配置完成！正在启动训练..."
	default:
		return ""
	}
}

func (m Model) viewSelectProject() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render(".train目录下检测到下列已配置好的方案:"))
	b.WriteString("\n\n")

	for i, p := range m.projects {
		cursor := "  "
		if m.projectCursor == i {
			cursor = "▸ "
		}

		name := fmt.Sprintf("[%d] %s  [%s]", i+1, p.ID, p.Mode.String())
		summary := p.GetProjectSummary()

		if m.projectCursor == i {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(name)
		}

		b.WriteString(cursor + name + "\n")
		b.WriteString(fmt.Sprintf("    %s\n\n", summary))
	}

	// New project option
	cursor := "  "
	if m.projectCursor == len(m.projects) {
		cursor = "▸ "
	}
	newOption := "[重新设置一套全新配置]"
	if m.projectCursor == len(m.projects) {
		newOption = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(newOption)
	}
	b.WriteString(cursor + newOption + "\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("请使用 ↑/↓ 选择, Enter 确认"))

	return b.String()
}

func (m Model) viewSelectMode() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("请选择训练模式:"))
	b.WriteString("\n\n")

	modes := []string{
		"NPU,GPU对比训练",
		"NPU单独训练",
		"GPU单独训练",
	}

	for i, mode := range modes {
		cursor := "  "
		if m.modeCursor == i {
			cursor = "▸ "
		}

		label := fmt.Sprintf("[%d] %s", i+1, mode)
		if m.modeCursor == i {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(label)
		}

		b.WriteString(cursor + label + "\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("请使用 ↑/↓ 选择, Enter 确认"))

	return b.String()
}

func (m Model) viewEditField() string {
	if m.fieldCursor >= len(m.fields) {
		return ""
	}

	field := m.fields[m.fieldCursor]
	fieldValue := train.GetFieldValue(m.currentProject, field)
	currentDisplay := train.GetFieldDisplayValue(fieldValue, field.Type)

	var b strings.Builder

	// Title
	title := fmt.Sprintf("配置项 [%d/%d]: %s", m.fieldCursor+1, len(m.fields), field.Label)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("━", 40))
	b.WriteString("\n\n")

	// Description
	b.WriteString(field.Description)
	b.WriteString("\n\n")

	// Options: first line shows current value (default selected), second line for input
	// If current value is empty, show a placeholder
	firstOption := currentDisplay
	if strings.TrimSpace(currentDisplay) == "" {
		firstOption = "< 无 >"
	}
	options := []string{firstOption, "<输入新值>"}
	for i, opt := range options {
		cursor := "  "
		if m.fieldOptionCursor == i {
			cursor = "▸ "
		}

		label := fmt.Sprintf("[%d] %s", i+1, opt)
		if m.fieldOptionCursor == i {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(label)
		}

		b.WriteString(cursor + label + "\n")
	}

	b.WriteString("\n")
	hint := "↑/↓ 选择选项 | Enter 确认 | Tab 跳过 | Esc 返回"
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(hint))

	return b.String()
}

func (m Model) viewInputValue() string {
	if m.fieldCursor >= len(m.fields) {
		return ""
	}

	field := m.fields[m.fieldCursor]

	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("输入新值"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("━", 40))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%s:\n", field.Label))
	b.WriteString(m.inputBuffer)
	b.WriteString("█") // cursor

	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("错误: " + m.errMsg))
	}

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Enter 确认 | Esc 取消"))

	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("配置确认"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("━", 40))
	b.WriteString("\n\n")

	if m.currentProject != nil {
		b.WriteString(fmt.Sprintf("模式: %s\n\n", m.currentProject.Mode.String()))

		// Show summary of all fields
		for _, field := range m.fields {
			value := train.GetFieldValue(m.currentProject, field)
			display := train.GetFieldDisplayValue(value, field.Type)
			b.WriteString(fmt.Sprintf("%-20s %s\n", field.Label+":", display))
		}
	}

	b.WriteString("\n")
	b.WriteString("确认使用此配置启动训练? [Y/n]\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Y 确认 | N 返回编辑 | Esc 取消"))

	return b.String()
}

// GetFinalProject returns the configured project
func (m Model) GetFinalProject() *train.TrainProject {
	return m.currentProject
}

// IsDone returns true if wizard is complete
func (m Model) IsDone() bool {
	return m.mode == ModeDone
}

// Helper functions

func copyProject(p *train.TrainProject) *train.TrainProject {
	if p == nil {
		return nil
	}
	// Deep copy
	copy := *p
	if p.NPUConfig != nil {
		npuCopy := *p.NPUConfig
		copy.NPUConfig = &npuCopy
	}
	if p.GPUConfig != nil {
		gpuCopy := *p.GPUConfig
		copy.GPUConfig = &gpuCopy
	}
	copy.Exclude = append([]string{}, p.Exclude...)
	return &copy
}
