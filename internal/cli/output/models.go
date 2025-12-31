package output

import "time"

type TargetRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type TargetResult struct {
	Target  TargetRef `json:"target"`
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
}

type ActionResult struct {
	Action   string                 `json:"action"`
	Status   string                 `json:"status"`
	Message  string                 `json:"message,omitempty"`
	Targets  []TargetRef            `json:"targets,omitempty"`
	Results  []TargetResult         `json:"results,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Details  map[string]any         `json:"details,omitempty"`
}

type SessionSummary struct {
	Name         string    `json:"name"`
	Layout       string    `json:"layout,omitempty"`
	Cwd          string    `json:"cwd,omitempty"`
	PaneCount    int       `json:"pane_count"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
}

type PaneSummary struct {
	ID          string    `json:"id"`
	Index       int       `json:"index"`
	Title       string    `json:"title,omitempty"`
	Command     string    `json:"command,omitempty"`
	StartCmd    string    `json:"start_command,omitempty"`
	Cwd         string    `json:"cwd,omitempty"`
	Dead        bool      `json:"dead,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
	BytesIn     uint64    `json:"bytes_in,omitempty"`
	BytesOut    uint64    `json:"bytes_out,omitempty"`
}

type PaneSummaryWithContext struct {
	PaneSummary
	SessionName string `json:"session_name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

type SessionSnapshot struct {
	Name   string        `json:"name"`
	Layout string        `json:"layout,omitempty"`
	Cwd    string        `json:"cwd,omitempty"`
	Panes  []PaneSummary `json:"panes"`
}

type ProjectSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Hidden       bool      `json:"hidden"`
	LastOpened   time.Time `json:"last_opened,omitempty"`
	SessionCount int       `json:"session_count,omitempty"`
}

type ProjectSnapshot struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Hidden   bool             `json:"hidden"`
	Sessions []SessionSnapshot `json:"sessions"`
}

type Snapshot struct {
	Projects         []ProjectSnapshot `json:"projects"`
	SelectedProject  string            `json:"selected_project_id,omitempty"`
	SelectedSession  string            `json:"selected_session_name,omitempty"`
	SelectedPaneID   string            `json:"selected_pane_id,omitempty"`
	GeneratedAt      time.Time         `json:"generated_at,omitempty"`
}

type LayoutSummary struct {
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
	Source string `json:"source,omitempty"`
}

type PaneView struct {
	PaneID    string `json:"pane_id"`
	Mode      string `json:"mode"`
	Rows      int    `json:"rows"`
	Cols      int    `json:"cols"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

type PaneSnapshotOutput struct {
	PaneID    string `json:"pane_id"`
	Rows      int    `json:"rows"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

type PaneTailFrame struct {
	PaneID    string `json:"pane_id"`
	Chunk     string `json:"chunk"`
	Encoding  string `json:"encoding,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type PaneHistoryEntry struct {
	TS      time.Time `json:"ts"`
	Action  string    `json:"action"`
	Summary string    `json:"summary,omitempty"`
	Command string    `json:"command,omitempty"`
	Status  string    `json:"status,omitempty"`
}

type PaneHistory struct {
	PaneID  string             `json:"pane_id"`
	Entries []PaneHistoryEntry `json:"entries"`
}

type PaneWaitResult struct {
	PaneID    string `json:"pane_id"`
	Pattern   string `json:"pattern"`
	Matched   bool   `json:"matched"`
	Match     string `json:"match,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms,omitempty"`
}

type PaneTagList struct {
	PaneID string   `json:"pane_id"`
	Tags   []string `json:"tags"`
}

type RelayStats struct {
	Lines        uint64    `json:"lines,omitempty"`
	Bytes        uint64    `json:"bytes,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
}

type Relay struct {
	ID         string     `json:"id"`
	FromPaneID string     `json:"from_pane_id"`
	ToPaneIDs  []string   `json:"to_pane_ids,omitempty"`
	Scope      string     `json:"scope,omitempty"`
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	Delay      string     `json:"delay,omitempty"`
	Prefix     string     `json:"prefix,omitempty"`
	TTL        string     `json:"ttl,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	Stats      RelayStats `json:"stats,omitempty"`
}

type Event struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type"`
	TS      time.Time      `json:"ts,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

type GitContext struct {
	Root   string `json:"root,omitempty"`
	Branch string `json:"branch,omitempty"`
	Head   string `json:"head,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
	Ahead  int    `json:"ahead,omitempty"`
	Behind int    `json:"behind,omitempty"`
}

type ContextPack struct {
	Snapshot  *Snapshot   `json:"snapshot,omitempty"`
	Git       *GitContext `json:"git,omitempty"`
	Errors    []string    `json:"errors,omitempty"`
	MaxBytes  int         `json:"max_bytes,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}

type WorkspaceList struct {
	Projects []ProjectSummary `json:"projects"`
	Roots    []string         `json:"roots,omitempty"`
	Total    int              `json:"total"`
}

type LayoutList struct {
	Layouts []LayoutSummary `json:"layouts"`
	Total   int             `json:"total"`
}

type LayoutExport struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type NLPlannedCommand struct {
	ID              string         `json:"id"`
	Command         string         `json:"command"`
	Args            []string       `json:"args,omitempty"`
	Flags           map[string]any `json:"flags,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	SideEffects     bool           `json:"side_effects,omitempty"`
	RequiresConfirm bool           `json:"requires_confirm,omitempty"`
}

type NLPlan struct {
	PlanID               string            `json:"plan_id"`
	Rationale            string            `json:"rationale,omitempty"`
	Commands             []NLPlannedCommand `json:"commands"`
	RequiresConfirmations []string          `json:"requires_confirmations,omitempty"`
}

type NLExecutionStep struct {
	ID         string        `json:"id"`
	Command    string        `json:"command"`
	Status     string        `json:"status"`
	StartedAt  time.Time     `json:"started_at,omitempty"`
	FinishedAt time.Time     `json:"finished_at,omitempty"`
	Result     *ActionResult `json:"result,omitempty"`
}

type NLExecution struct {
	ExecutionID string             `json:"execution_id"`
	PlanID      string             `json:"plan_id"`
	Steps       []NLExecutionStep `json:"steps"`
}
