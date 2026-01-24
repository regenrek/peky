package viewlayout

import "testing"

func TestDashboardLayoutStandardPromptDelta(t *testing.T) {
	withPrompt, ok := Dashboard(40, 20, true, true, false)
	if !ok {
		t.Fatalf("expected layout with prompt")
	}
	withoutPrompt, ok := Dashboard(40, 20, false, true, false)
	if !ok {
		t.Fatalf("expected layout without prompt")
	}
	if withPrompt.HeaderGap != 1 || withoutPrompt.HeaderGap != 1 {
		t.Fatalf("expected header gap 1, got with=%d without=%d", withPrompt.HeaderGap, withoutPrompt.HeaderGap)
	}
	if withPrompt.BodyHeight != withoutPrompt.BodyHeight-1 {
		t.Fatalf("expected body height delta 1, got with=%d without=%d", withPrompt.BodyHeight, withoutPrompt.BodyHeight)
	}
	if withPrompt.PekyPromptHeight != 1 || withoutPrompt.PekyPromptHeight != 0 {
		t.Fatalf("unexpected peky prompt height with=%d without=%d", withPrompt.PekyPromptHeight, withoutPrompt.PekyPromptHeight)
	}
}

func TestDashboardLayoutPromptHeaderGapFlip(t *testing.T) {
	withPrompt, ok := Dashboard(40, 10, true, true, false)
	if !ok {
		t.Fatalf("expected layout with prompt")
	}
	withoutPrompt, ok := Dashboard(40, 10, false, true, false)
	if !ok {
		t.Fatalf("expected layout without prompt")
	}
	if withoutPrompt.HeaderGap != 1 {
		t.Fatalf("expected header gap 1 without prompt, got %d", withoutPrompt.HeaderGap)
	}
	if withPrompt.HeaderGap != 0 {
		t.Fatalf("expected header gap 0 with prompt, got %d", withPrompt.HeaderGap)
	}
	if withPrompt.BodyHeight != withoutPrompt.BodyHeight {
		t.Fatalf("expected body height parity, got with=%d without=%d", withPrompt.BodyHeight, withoutPrompt.BodyHeight)
	}
}
