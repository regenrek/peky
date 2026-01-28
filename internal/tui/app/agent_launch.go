package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

type agentLaunchDef struct {
	Key          string
	Label        string
	ToolID       string
	DefaultCmd   string
	DefaultTitle string
}

var agentLaunchDefs = []agentLaunchDef{
	{
		Key:          "codex_cli",
		Label:        "Codex CLI",
		ToolID:       "codex",
		DefaultCmd:   "codex",
		DefaultTitle: "codex",
	},
	{
		Key:          "claude_code",
		Label:        "Claude Code",
		ToolID:       "claude",
		DefaultCmd:   "claude",
		DefaultTitle: "claude",
	},
	{
		Key:          "pi",
		Label:        "Pi",
		ToolID:       "pi",
		DefaultCmd:   "pi",
		DefaultTitle: "pi",
	},
	{
		Key:          "opencode",
		Label:        "Opencode",
		ToolID:       "opencode",
		DefaultCmd:   "opencode",
		DefaultTitle: "opencode",
	},
}

func (m *Model) agentLaunchCommandSpecs() []commandSpec {
	specs := make([]commandSpec, 0, len(agentLaunchDefs))
	for _, def := range agentLaunchDefs {
		def := def
		specs = append(specs, commandSpec{
			ID:    commandID("agent_launch_" + def.Key),
			Label: def.Label,
			Desc:  "Launch " + def.Label + " in new pane",
			Run: func(m *Model, _ commandArgs) tea.Cmd {
				return m.launchAgent(def)
			},
		})
	}
	return specs
}

func (m *Model) launchAgent(def agentLaunchDef) tea.Cmd {
	if m == nil {
		return nil
	}
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	pane := m.selectedPane()
	if pane == nil {
		m.setToast("No pane selected", toastWarning)
		return nil
	}
	cmd, title, err := m.resolveAgentLaunch(def)
	if err != nil {
		m.setToast(err.Error(), toastError)
		return nil
	}
	vertical := autoSplitVertical(pane.Width, pane.Height)
	result, err := m.splitPaneFor(session.Name, pane.ID, vertical)
	if err != nil {
		level := splitPaneErrLevel(err)
		msg := err.Error()
		if level == toastError && !strings.HasPrefix(msg, "Start failed:") {
			msg = "Add agent failed: " + msg
		}
		m.setToast(msg, level)
		return nil
	}
	sel := m.selection
	sel.Session = result.sessionName
	sel.Pane = result.newIndex
	m.applySelection(sel)
	m.selectionVersion++
	m.lastSplitVertical = vertical
	m.lastSplitSet = true
	if err := m.applyAgentTitle(result.newPaneID, title); err != nil {
		m.setToast("Agent title failed: "+err.Error(), toastWarning)
	}
	if err := m.sendAgentCommand(result.newPaneID, cmd); err != nil {
		m.setToast("Add agent failed: "+err.Error(), toastError)
		return m.requestRefreshCmd()
	}
	if err := m.setAgentTool(result.newPaneID, def.ToolID); err != nil {
		m.setToast("Agent tool failed: "+err.Error(), toastWarning)
	}
	m.setToast("Agent launched: "+def.Label, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) resolveAgentLaunch(def agentLaunchDef) (string, string, error) {
	cfg := m.agentToolConfig(def)
	cmd := strings.TrimSpace(cfg.Cmd)
	if cmd == "" {
		cmd = def.DefaultCmd
	}
	title := strings.TrimSpace(cfg.PaneTitle)
	if title == "" {
		title = def.DefaultTitle
	}
	cmd, err := validateAgentCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return cmd, title, nil
}

func (m *Model) agentToolConfig(def agentLaunchDef) layout.ToolConfig {
	cfg := layout.ToolConfig{}
	if m != nil && m.config != nil {
		cfg = toolConfigForKey(m.config.Tools, def.Key)
	}
	project := m.selectedProject()
	if project == nil {
		return cfg
	}
	local := m.projectLocalConfigForPath(project.Path)
	if local == nil {
		return cfg
	}
	override := toolConfigForKey(local.Tools, def.Key)
	return mergeToolConfig(cfg, override)
}

func toolConfigForKey(cfg layout.ToolsConfig, key string) layout.ToolConfig {
	switch key {
	case "codex_cli":
		return cfg.CodexCLI
	case "claude_code":
		return cfg.ClaudeCode
	case "pi":
		return cfg.Pi
	case "opencode":
		return cfg.Opencode
	default:
		return layout.ToolConfig{}
	}
}

func mergeToolConfig(base, override layout.ToolConfig) layout.ToolConfig {
	if strings.TrimSpace(override.Cmd) != "" {
		base.Cmd = override.Cmd
	}
	if strings.TrimSpace(override.PaneTitle) != "" {
		base.PaneTitle = override.PaneTitle
	}
	return base
}

func validateAgentCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", errors.New("agent command is empty")
	}
	for _, r := range cmd {
		if unicode.IsControl(r) {
			return "", errors.New("agent command contains control characters")
		}
	}
	return cmd, nil
}

func (m *Model) applyAgentTitle(paneID, title string) error {
	if strings.TrimSpace(title) == "" {
		return nil
	}
	if err := validateSessionName(title); err != nil {
		return err
	}
	if m == nil || m.client == nil {
		return errors.New("session client unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.client.RenamePaneByID(ctx, paneID, title)
}

func (m *Model) sendAgentCommand(paneID, cmd string) error {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return errors.New("pane id unavailable")
	}
	if m == nil || m.client == nil {
		return errors.New("session client unavailable")
	}
	payload := []byte(cmd + "\r")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.client.SendInput(ctx, paneID, payload)
}

func (m *Model) setAgentTool(paneID, toolID string) error {
	paneID = strings.TrimSpace(paneID)
	toolID = strings.TrimSpace(toolID)
	if paneID == "" || toolID == "" {
		return nil
	}
	if m == nil || m.client == nil {
		return errors.New("session client unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.client.SetPaneTool(ctx, paneID, toolID)
}
