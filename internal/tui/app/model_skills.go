package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/skills"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/userpath"
)

const skillsTargetPickerHeading = "Install Skills"

type skillsInstallResultMsg struct {
	Result skills.InstallResult
	Err    error
}

func (m *Model) setupSkillsTargetPicker() {
	m.skillsTargetPicker = picker.NewMultiSelectPicker(skillsTargetPickerHeading)
}

func (m *Model) openSkillsInstall() {
	options := buildSkillsTargetOptions()
	if len(options) == 0 {
		m.setToast("No skill targets available", toastWarning)
		return
	}
	m.skillsTargetItems = options
	items := make([]list.Item, 0, len(options))
	for _, option := range options {
		items = append(items, option)
	}
	m.skillsTargetPicker.SetItems(items)
	m.setSkillsTargetPickerSize()
	m.setState(StateSkillsTargetPicker)
}

func (m *Model) setSkillsTargetPickerSize() {
	m.setDialogMenuSize(&m.skillsTargetPicker, skillsTargetPickerHeading, 36, 72, 8, 16, 2)
}

func (m *Model) updateSkillsTargetPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case " ":
		m.toggleSkillsTargetSelection()
		return m, nil
	case "enter":
		targets := m.selectedSkillsTargets()
		if len(targets) == 0 {
			m.setToast("Select at least one target", toastWarning)
			return m, nil
		}
		m.setState(StateDashboard)
		if m.skillsInstallInFlight {
			m.setToast("Skills install already running", toastInfo)
			return m, nil
		}
		m.skillsInstallInFlight = true
		m.setToast("Installing skills...", toastInfo)
		return m, m.installSkillsCmd(targets)
	default:
	}

	var cmd tea.Cmd
	m.skillsTargetPicker, cmd = m.skillsTargetPicker.Update(msg)
	return m, cmd
}

func (m *Model) toggleSkillsTargetSelection() {
	item, ok := m.skillsTargetPicker.SelectedItem().(*picker.MultiSelectItem)
	if !ok || item == nil {
		return
	}
	item.Selected = !item.Selected
}

func (m *Model) selectedSkillsTargets() []skills.Target {
	targets := make([]skills.Target, 0, len(m.skillsTargetItems))
	for _, item := range m.skillsTargetItems {
		if item.Selected {
			targets = append(targets, skills.Target(item.ID))
		}
	}
	return targets
}

func (m *Model) installSkillsCmd(targets []skills.Target) tea.Cmd {
	targetList := append([]skills.Target(nil), targets...)
	return func() tea.Msg {
		bundle, err := skills.LoadBundle()
		if err != nil {
			return skillsInstallResultMsg{Err: err}
		}
		result, err := skills.Install(bundle, skills.InstallOptions{Targets: targetList})
		return skillsInstallResultMsg{Result: result, Err: err}
	}
}

func (m *Model) handleSkillsInstallResult(msg skillsInstallResultMsg) tea.Cmd {
	m.skillsInstallInFlight = false
	if msg.Err != nil {
		m.setToast("Skills install failed: "+msg.Err.Error(), toastError)
		return nil
	}
	message, level := summarizeSkillsInstall(msg.Result)
	m.setToast(message, level)
	return nil
}

func summarizeSkillsInstall(result skills.InstallResult) (string, toastLevel) {
	if len(result.Records) == 0 {
		return "No skills installed", toastInfo
	}
	installed := 0
	skipped := 0
	targets := make(map[skills.Target]struct{})
	for _, record := range result.Records {
		switch record.Status {
		case "installed":
			installed++
			targets[record.Target] = struct{}{}
		default:
			skipped++
		}
	}
	if installed == 0 {
		return fmt.Sprintf("No installs (%d skipped)", skipped), toastInfo
	}
	targetLabels := make([]string, 0, len(targets))
	for target := range targets {
		targetLabels = append(targetLabels, skills.TargetLabel(target))
	}
	sort.Strings(targetLabels)
	skillLabel := "skills"
	if installed == 1 {
		skillLabel = "skill"
	}
	msg := fmt.Sprintf("Installed %d %s for %s", installed, skillLabel, strings.Join(targetLabels, ", "))
	if skipped > 0 {
		msg += fmt.Sprintf(" (%d skipped)", skipped)
	}
	return msg, toastSuccess
}

func buildSkillsTargetOptions() []*picker.MultiSelectItem {
	targets := skills.Targets()
	options := make([]*picker.MultiSelectItem, 0, len(targets))
	for _, target := range targets {
		root := ""
		if dest, err := skills.TargetRoot(target); err == nil {
			root = userpath.ShortenUser(dest)
		}
		options = append(options, &picker.MultiSelectItem{
			ID:       string(target),
			Label:    skills.TargetLabel(target),
			Desc:     root,
			Selected: target == skills.TargetCodex,
		})
	}
	return options
}
