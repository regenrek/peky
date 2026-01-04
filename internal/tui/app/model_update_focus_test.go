package app

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type nonModelUpdateMsg struct{}

type dummyTeaModel struct{}

func (dummyTeaModel) Init() tea.Cmd { return nil }

func (dummyTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return dummyTeaModel{}, nil }

func (dummyTeaModel) View() string { return "" }

func TestUpdateAppendsFocusForNonModel(t *testing.T) {
	m := newTestModelLite()
	m.focusPending = true
	m.focusSelection = selectionState{Session: "alpha-1"}

	msgType := reflect.TypeOf(nonModelUpdateMsg{})
	prev, hadPrev := updateHandlers[msgType]
	updateHandlers[msgType] = func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return dummyTeaModel{}, nil
	}
	t.Cleanup(func() {
		if hadPrev {
			updateHandlers[msgType] = prev
		} else {
			delete(updateHandlers, msgType)
		}
	})

	m.Update(nonModelUpdateMsg{})
	if m.focusPending {
		t.Fatalf("expected focus pending to be consumed")
	}
}
