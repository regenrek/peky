package spec

import "testing"

func TestParseRejectsEmpty(t *testing.T) {
	if _, err := Parse([]byte("")); err == nil {
		t.Fatalf("expected error for empty spec")
	}
}

func TestValidateRejectsMissingName(t *testing.T) {
	yaml := []byte("version: 1\napp: {}\ncommands: []\n")
	if _, err := Parse(yaml); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestAllCommandsAndFindByID(t *testing.T) {
	specDoc, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	commands := specDoc.AllCommands()
	if len(commands) == 0 {
		t.Fatalf("expected commands")
	}
	cmd := specDoc.FindByID("pane.send")
	if cmd == nil {
		t.Fatalf("expected pane.send command")
	}
	if cmd.Name == "" {
		t.Fatalf("expected command name")
	}
	if specDoc.FindByID("daemon.start") == nil {
		t.Fatalf("expected daemon.start command")
	}
	if specDoc.FindByID("daemon.stop") == nil {
		t.Fatalf("expected daemon.stop command")
	}
}

func TestSlashCommandsFiltersDisabled(t *testing.T) {
	specDoc := &Spec{Commands: []Command{
		{Name: "foo", ID: "foo", Slash: &SlashSpec{Enabled: false}},
		{Name: "bar", ID: "bar", Slash: &SlashSpec{Enabled: true}},
		{Name: "baz", ID: "baz"},
	}}
	commands := specDoc.SlashCommands()
	if len(commands) != 2 {
		t.Fatalf("expected 2 slash commands, got %d", len(commands))
	}
	if commands[0].ID == "foo" || commands[1].ID == "foo" {
		t.Fatalf("disabled command should be filtered out")
	}
}
