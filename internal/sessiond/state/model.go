package state

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/native"
)

// CurrentSchemaVersion identifies the persisted schema version.
const CurrentSchemaVersion = 1

// RuntimeState captures the persisted daemon state.
type RuntimeState struct {
	SchemaVersion int       `json:"schemaVersion"`
	AppVersion    string    `json:"appVersion,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Sessions      []Session `json:"sessions,omitempty"`
}

// Session is the persisted representation of a session.
type Session struct {
	Name       string    `json:"name"`
	Path       string    `json:"path,omitempty"`
	LayoutName string    `json:"layoutName,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`

	Env []string `json:"env,omitempty"`

	ActivePaneIndex string `json:"activePaneIndex,omitempty"`

	Panes []Pane `json:"panes,omitempty"`
}

// Pane is the persisted representation of a pane.
type Pane struct {
	Index        string `json:"index"`
	Title        string `json:"title,omitempty"`
	Command      string `json:"command,omitempty"`
	StartCommand string `json:"startCommand,omitempty"`

	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`

	RestoreFailed bool   `json:"restoreFailed,omitempty"`
	RestoreError  string `json:"restoreError,omitempty"`
}

// FromSnapshots converts native snapshots into persisted state.
func FromSnapshots(appVersion string, sessions []native.SessionSnapshot) RuntimeState {
	state := RuntimeState{
		SchemaVersion: CurrentSchemaVersion,
		AppVersion:    strings.TrimSpace(appVersion),
		UpdatedAt:     time.Now(),
	}
	if len(sessions) == 0 {
		return state
	}
	state.Sessions = make([]Session, 0, len(sessions))
	for _, snap := range sessions {
		session := Session{
			Name:       strings.TrimSpace(snap.Name),
			Path:       strings.TrimSpace(snap.Path),
			LayoutName: strings.TrimSpace(snap.LayoutName),
			CreatedAt:  snap.CreatedAt,
			Env:        copyStrings(snap.Env),
		}
		if len(snap.Panes) > 0 {
			session.Panes = make([]Pane, len(snap.Panes))
			for i, pane := range snap.Panes {
				if session.ActivePaneIndex == "" && pane.Active {
					session.ActivePaneIndex = pane.Index
				}
				session.Panes[i] = Pane{
					Index:         pane.Index,
					Title:         strings.TrimSpace(pane.Title),
					Command:       strings.TrimSpace(pane.Command),
					StartCommand:  strings.TrimSpace(pane.StartCommand),
					Left:          pane.Left,
					Top:           pane.Top,
					Width:         pane.Width,
					Height:        pane.Height,
					RestoreFailed: pane.RestoreFailed,
					RestoreError:  strings.TrimSpace(pane.RestoreError),
				}
			}
		}
		state.Sessions = append(state.Sessions, session)
	}
	state.Normalize()
	return state
}

// Normalize orders sessions/panes and clamps geometry for stable output.
func (s *RuntimeState) Normalize() {
	if s == nil {
		return
	}
	if len(s.Sessions) > 1 {
		sort.Slice(s.Sessions, func(i, j int) bool {
			left := s.Sessions[i]
			right := s.Sessions[j]
			if !left.CreatedAt.Equal(right.CreatedAt) {
				return left.CreatedAt.Before(right.CreatedAt)
			}
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return left.Path < right.Path
		})
	}
	base := native.LayoutBaseSize
	for si := range s.Sessions {
		session := &s.Sessions[si]
		session.ActivePaneIndex = strings.TrimSpace(session.ActivePaneIndex)
		if len(session.Panes) > 1 {
			sort.Slice(session.Panes, func(i, j int) bool {
				left := session.Panes[i]
				right := session.Panes[j]
				li, lok := parseIndex(left.Index)
				ri, rok := parseIndex(right.Index)
				if lok && rok {
					if li != ri {
						return li < ri
					}
				} else if lok != rok {
					return lok
				}
				return left.Index < right.Index
			})
		}
		for pi := range session.Panes {
			pane := &session.Panes[pi]
			pane.Left = clamp(pane.Left, 0, base)
			pane.Top = clamp(pane.Top, 0, base)
			pane.Width = clamp(pane.Width, 1, base)
			pane.Height = clamp(pane.Height, 1, base)
		}
		if len(session.Panes) == 0 {
			session.ActivePaneIndex = ""
			continue
		}
		if !paneIndexExists(session.Panes, session.ActivePaneIndex) {
			session.ActivePaneIndex = session.Panes[0].Index
		}
	}
}

func parseIndex(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func paneIndexExists(panes []Pane, index string) bool {
	for _, pane := range panes {
		if pane.Index == index {
			return true
		}
	}
	return false
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
