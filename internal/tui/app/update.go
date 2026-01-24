package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/update"
)

const (
	updateShortcutHint   = "Ctrl+Shift+U"
	updateInitialDelay   = 2 * time.Second
	updateInstallTimeout = 10 * time.Minute
	updateRestartTimeout = 20 * time.Second
)

type updateProgress struct {
	Step    string
	Percent int
	Err     error
}

type updateCheckMsg struct {
	Force bool
}

type updateCheckResultMsg struct {
	Latest    string
	CheckedAt time.Time
	Force     bool
	Err       error
}

type updateTickMsg struct{}

type updateProgressMsg struct {
	Step    string
	Percent int
}

type updateInstallResultMsg struct {
	Err error
}

type updateRestartMsg struct {
	Err error
}

func (m *Model) initUpdateState() {
	m.updatePolicy = update.DefaultPolicy()
	currentVersion := ""
	if m.client != nil {
		currentVersion = m.client.Version()
	}
	m.updateState = update.State{CurrentVersion: currentVersion}
	exe, err := os.Executable()
	if err == nil {
		spec, specErr := update.DetectInstall(context.Background(), exe)
		if specErr == nil {
			m.updateSpec = spec
			m.updateState.Channel = spec.Channel
		}
	}
	statePath, err := update.DefaultStatePath()
	if err != nil {
		if !errors.Is(err, update.ErrStateDisabled) {
			slog.Warn("update state path unavailable", "err", err)
		}
		m.updateClient = update.NPMClient{UserAgent: updateUserAgent(currentVersion)}
		return
	}
	store := update.FileStore{Path: statePath}
	loaded, err := store.Load(context.Background())
	if err != nil {
		slog.Warn("update state load failed", "err", err)
	} else {
		loaded.CurrentVersion = m.updateState.CurrentVersion
		loaded.Channel = m.updateState.Channel
		m.updateState = loaded
	}
	m.updateStore = store
	m.updateClient = update.NPMClient{UserAgent: updateUserAgent(currentVersion)}
}

func updateUserAgent(version string) string {
	v := update.NormalizeVersion(version)
	if v == "" {
		return "peakypanes/auto-update"
	}
	return fmt.Sprintf("peakypanes/%s", v)
}

func (m *Model) updateBannerInfo() (string, string, bool) {
	if m.updatePendingRestart {
		return "Restart to finish update", updateShortcutHint, true
	}
	if !m.updatePolicy.ShouldShowBanner(m.updateState) {
		return "", "", false
	}
	latest := update.NormalizeVersion(m.updateState.LatestVersion)
	label := "Update available"
	if latest != "" {
		label = fmt.Sprintf("Update available v%s", latest)
	}
	return label, updateShortcutHint, true
}

func (m *Model) updateDialogView() updateDialogView {
	latest := update.NormalizeVersion(m.updateState.LatestVersion)
	current := update.NormalizeVersion(m.updateState.CurrentVersion)
	command := update.UpdateCommand(m.updateSpec)
	remindLabel := formatUpdateRemindLabel(m.updatePolicy.PromptCooldown)
	return updateDialogView{
		CurrentVersion: current,
		LatestVersion:  latest,
		Channel:        m.updateSpec.Channel,
		Command:        command,
		CanInstall:     m.updateSpec.Channel != update.ChannelUnknown,
		RemindLabel:    remindLabel,
	}
}

func (m *Model) scheduleUpdateInit() tea.Cmd {
	interval := m.updatePolicy.CheckInterval
	if interval <= 0 {
		interval = update.DefaultCheckInterval
	}
	return tea.Batch(
		tea.Tick(updateInitialDelay, func(time.Time) tea.Msg { return updateCheckMsg{Force: false} }),
		tea.Tick(interval, func(time.Time) tea.Msg { return updateTickMsg{} }),
	)
}

func (m *Model) handleUpdateCheck(msg updateCheckMsg) tea.Cmd {
	if m.updateCheckInFlight {
		return nil
	}
	now := time.Now()
	if !msg.Force && !m.updatePolicy.ShouldCheck(m.updateState.LastCheckUnixMs, now) {
		return nil
	}
	m.updateCheckInFlight = true
	return m.updateCheckCmd(msg.Force)
}

func (m *Model) handleUpdateTick() tea.Cmd {
	interval := m.updatePolicy.CheckInterval
	if interval <= 0 {
		interval = update.DefaultCheckInterval
	}
	cmd := tea.Tick(interval, func(time.Time) tea.Msg { return updateTickMsg{} })
	return tea.Batch(cmd, m.handleUpdateCheck(updateCheckMsg{Force: false}))
}

func (m *Model) updateCheckCmd(force bool) tea.Cmd {
	client := m.updateClient
	return func() tea.Msg {
		if client == nil {
			return updateCheckResultMsg{Err: errors.New("update client unavailable"), Force: force, CheckedAt: time.Now()}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		latest, err := client.LatestVersion(ctx)
		return updateCheckResultMsg{Latest: latest, Err: err, Force: force, CheckedAt: time.Now()}
	}
}

func (m *Model) handleUpdateCheckResult(msg updateCheckResultMsg) tea.Cmd {
	m.updateCheckInFlight = false
	if msg.Err != nil {
		if msg.Force {
			m.setToast("Update check failed: "+msg.Err.Error(), toastError)
		}
		return nil
	}
	m.updateState.LatestVersion = msg.Latest
	m.updateState.MarkChecked(msg.CheckedAt)
	if m.updateStore != nil {
		saveUpdateState(m.updateStore, m.updateState)
	}
	if msg.Force {
		if !m.updateState.UpdateAvailable() {
			m.setToast("PeakyPanes is up to date", toastSuccess)
		} else if m.updateState.IsSkipped() {
			m.setToast("Latest update is skipped", toastInfo)
		} else {
			return m.openUpdateDialog()
		}
	}
	if m.updatePolicy.ShouldPrompt(m.updateState, msg.CheckedAt) {
		return m.promptUpdateIfReady()
	}
	return nil
}

func saveUpdateState(store update.Store, state update.State) {
	if store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := store.Save(ctx, state); err != nil {
		slog.Warn("update state save failed", "err", err)
	}
}

func (m *Model) promptUpdateIfReady() tea.Cmd {
	if m.state == StateDashboard {
		return m.openUpdateDialog()
	}
	m.updatePromptPending = true
	return nil
}

func (m *Model) openUpdateDialog() tea.Cmd {
	if m.updatePendingRestart {
		m.setState(StateUpdateRestart)
		return nil
	}
	if !m.updateState.UpdateAvailable() || m.updateState.IsSkipped() {
		return nil
	}
	m.setState(StateUpdateDialog)
	return nil
}

func (m *Model) handleUpdateShortcut(msg tea.KeyMsg) (tea.Cmd, bool) {
	if msg.String() != "ctrl+shift+u" {
		return nil, false
	}
	return m.openUpdateDialog(), true
}

func (m *Model) updateUpdateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i", "enter":
		return m, m.startUpdateInstall()
	case "l":
		m.markUpdatePrompted()
		m.setState(StateDashboard)
		return m, nil
	case "s":
		m.updateState.SkippedVersion = m.updateState.LatestVersion
		m.markUpdatePrompted()
		m.setState(StateDashboard)
		return m, nil
	case "esc":
		m.markUpdatePrompted()
		m.setState(StateDashboard)
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) markUpdatePrompted() {
	m.updateState.MarkPrompted(time.Now())
	if m.updateStore != nil {
		saveUpdateState(m.updateStore, m.updateState)
	}
}

func (m *Model) startUpdateInstall() tea.Cmd {
	if m.updateSpec.Channel == update.ChannelUnknown {
		m.setToast("Automatic updates unavailable for this install", toastInfo)
		return nil
	}
	m.updateState.SkippedVersion = ""
	m.markUpdatePrompted()
	m.updateProgress = updateProgress{Step: "Preparing", Percent: 5}
	m.updateRunCh = startUpdateRun(m.updateSpec)
	m.setState(StateUpdateProgress)
	return waitUpdateRunMsg(m.updateRunCh)
}

func startUpdateRun(spec update.InstallSpec) <-chan tea.Msg {
	ch := make(chan tea.Msg, 4)
	go func() {
		defer close(ch)
		ch <- updateProgressMsg{Step: "Installing", Percent: 20}
		installer := update.NewInstaller()
		ctx, cancel := context.WithTimeout(context.Background(), updateInstallTimeout)
		defer cancel()
		err := installer.Install(ctx, spec)
		if err != nil {
			ch <- updateInstallResultMsg{Err: err}
			return
		}
		ch <- updateProgressMsg{Step: "Update installed", Percent: 100}
		ch <- updateInstallResultMsg{}
	}()
	return ch
}

func waitUpdateRunMsg(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) handleUpdateProgress(msg updateProgressMsg) tea.Cmd {
	m.updateProgress.Step = msg.Step
	m.updateProgress.Percent = msg.Percent
	return waitUpdateRunMsg(m.updateRunCh)
}

func (m *Model) handleUpdateInstallResult(msg updateInstallResultMsg) tea.Cmd {
	m.updateRunCh = nil
	if msg.Err != nil {
		m.setToast("Update failed: "+msg.Err.Error(), toastError)
		m.setState(StateUpdateDialog)
		return nil
	}
	m.updatePendingRestart = true
	m.setState(StateUpdateRestart)
	return nil
}

func (m *Model) updateUpdateProgress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *Model) updateUpdateRestart(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r", "enter":
		return m, m.restartAfterUpdateCmd()
	case "esc":
		m.setState(StateDashboard)
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) restartAfterUpdateCmd() tea.Cmd {
	current := ""
	if m.client != nil {
		current = m.client.Version()
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), updateRestartTimeout)
		defer cancel()
		if current != "" {
			if err := sessiond.RestartDaemon(ctx, current); err != nil {
				return updateRestartMsg{Err: err}
			}
		}
		exe, err := os.Executable()
		if err != nil {
			return updateRestartMsg{Err: err}
		}
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Start(); err != nil {
			return updateRestartMsg{Err: err}
		}
		return updateRestartMsg{}
	}
}

func (m *Model) handleUpdateRestart(msg updateRestartMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Restart failed: "+msg.Err.Error(), toastError)
		return nil
	}
	m.updatePendingRestart = false
	m.updateState.MarkPrompted(time.Now())
	if m.updateStore != nil {
		saveUpdateState(m.updateStore, m.updateState)
	}
	return tea.Sequence(m.shutdownCmd(), tea.Quit)
}

type updateDialogView struct {
	CurrentVersion string
	LatestVersion  string
	Channel        update.Channel
	Command        string
	CanInstall     bool
	RemindLabel    string
}

func formatUpdateRemindLabel(cooldown time.Duration) string {
	if cooldown <= 0 {
		cooldown = update.DefaultPromptCooldown
	}
	hours := int(cooldown.Hours())
	if hours >= 24 && hours%24 == 0 {
		days := hours / 24
		if days == 1 {
			return "Remind in 1 day"
		}
		return fmt.Sprintf("Remind in %d days", days)
	}
	if hours > 0 {
		return fmt.Sprintf("Remind in %d hours", hours)
	}
	minutes := int(cooldown.Minutes())
	if minutes > 0 {
		if minutes == 1 {
			return "Remind in 1 minute"
		}
		return fmt.Sprintf("Remind in %d minutes", minutes)
	}
	return "Later"
}
