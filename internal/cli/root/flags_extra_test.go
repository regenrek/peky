package root

import (
	"testing"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/spec"
)

func TestBuildFlagsEmpty(t *testing.T) {
	flags, err := buildFlags(nil)
	if err != nil {
		t.Fatalf("buildFlags error: %v", err)
	}
	if flags != nil {
		t.Fatalf("expected nil flags")
	}
}

func TestBuildFlagTypesAndDefaults(t *testing.T) {
	flag, err := buildFlag(spec.Flag{Name: "enabled", Type: "bool", Default: true})
	if err != nil {
		t.Fatalf("buildFlag bool error: %v", err)
	}
	if v := flag.(*cli.BoolFlag).Value; !v {
		t.Fatalf("expected default true")
	}

	flag, err = buildFlag(spec.Flag{Name: "name", Type: "string", Default: "ok"})
	if err != nil {
		t.Fatalf("buildFlag string error: %v", err)
	}
	if v := flag.(*cli.StringFlag).Value; v != "ok" {
		t.Fatalf("string default=%q", v)
	}

	flag, err = buildFlag(spec.Flag{Name: "count", Type: "int", Default: int64(7)})
	if err != nil {
		t.Fatalf("buildFlag int error: %v", err)
	}
	if v := flag.(*cli.IntFlag).Value; v != 7 {
		t.Fatalf("int default=%d", v)
	}

	flag, err = buildFlag(spec.Flag{Name: "ratio", Type: "float", Default: 1.5})
	if err != nil {
		t.Fatalf("buildFlag float error: %v", err)
	}
	if v := flag.(*cli.FloatFlag).Value; v != 1.5 {
		t.Fatalf("float default=%v", v)
	}

	flag, err = buildFlag(spec.Flag{Name: "wait", Type: "duration", Default: "2s"})
	if err != nil {
		t.Fatalf("buildFlag duration error: %v", err)
	}
	if v := flag.(*cli.DurationFlag).Value; v != 2*time.Second {
		t.Fatalf("duration default=%v", v)
	}

	flag, err = buildFlag(spec.Flag{Name: "tags", Type: "string_list", Default: []any{"a", "b"}})
	if err != nil {
		t.Fatalf("buildFlag string_list error: %v", err)
	}
	if v := flag.(*cli.StringSliceFlag).Value; len(v) != 2 || v[0] != "a" {
		t.Fatalf("string_list default=%v", v)
	}
}

func TestBuildFlagEnums(t *testing.T) {
	flag, err := buildFlag(spec.Flag{Name: "mode", Type: "enum", Enum: []string{"a", "b"}, Default: "a"})
	if err != nil {
		t.Fatalf("buildFlag enum error: %v", err)
	}
	strFlag := flag.(*cli.StringFlag)
	if err := strFlag.Validator("b"); err != nil {
		t.Fatalf("enum validator error: %v", err)
	}
	if err := strFlag.Validator("c"); err == nil {
		t.Fatalf("expected enum validator error")
	}

	flag, err = buildFlag(spec.Flag{Name: "levels", Type: "string_list", Enum: []string{"low", "high"}})
	if err != nil {
		t.Fatalf("buildFlag string_list enum error: %v", err)
	}
	listFlag := flag.(*cli.StringSliceFlag)
	if err := listFlag.Validator([]string{"low"}); err != nil {
		t.Fatalf("slice validator error: %v", err)
	}
	if err := listFlag.Validator([]string{"med"}); err == nil {
		t.Fatalf("expected slice validator error")
	}
}

func TestBuildFlagErrors(t *testing.T) {
	if _, err := buildFlag(spec.Flag{Name: " ", Type: "bool"}); err == nil {
		t.Fatalf("expected error for empty name")
	}
	if _, err := buildFlag(spec.Flag{Name: "x", Type: "nope"}); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestDefaultsHelpers(t *testing.T) {
	if got := boolDefault("nope"); got {
		t.Fatalf("expected false")
	}
	if got := stringDefault(123); got != "" {
		t.Fatalf("expected empty string")
	}
	if got := intDefault(3.7); got != 3 {
		t.Fatalf("int default=%d", got)
	}
	if got := floatDefault(int64(2)); got != 2 {
		t.Fatalf("float default=%v", got)
	}
	if got := durationDefault("bad"); got != 0 {
		t.Fatalf("expected zero duration")
	}
	if got := stringSliceDefault([]any{"a", 1}); len(got) != 1 || got[0] != "a" {
		t.Fatalf("stringSliceDefault=%v", got)
	}
}
