package icons

import "testing"

func TestVariantsBySize(t *testing.T) {
	v := Variants{Small: "s", Medium: "m", Large: "l"}
	if got := v.BySize(SizeSmall); got != "s" {
		t.Fatalf("BySize(small) = %q", got)
	}
	if got := v.BySize(SizeLarge); got != "l" {
		t.Fatalf("BySize(large) = %q", got)
	}
	if got := v.BySize(SizeMedium); got != "m" {
		t.Fatalf("BySize(medium) = %q", got)
	}
	v = Variants{Medium: "", Small: "s", Large: "l"}
	if got := v.BySize(SizeMedium); got != "s" {
		t.Fatalf("BySize fallback = %q", got)
	}
}

func TestActiveIconSet(t *testing.T) {
	t.Setenv("PEKY_ICON_SET", "ascii")
	if got := Active().Caret.Medium; got != ASCII.Caret.Medium {
		t.Fatalf("Active() ascii = %q", got)
	}
	t.Setenv("PEKY_ICON_SET", "unicode")
	if got := Active().Caret.Medium; got != Unicode.Caret.Medium {
		t.Fatalf("Active() unicode = %q", got)
	}
}

func TestActiveSize(t *testing.T) {
	t.Setenv("PEKY_ICON_SIZE", "sm")
	if got := ActiveSize(); got != SizeSmall {
		t.Fatalf("ActiveSize(sm) = %v", got)
	}
	t.Setenv("PEKY_ICON_SIZE", "lg")
	if got := ActiveSize(); got != SizeLarge {
		t.Fatalf("ActiveSize(lg) = %v", got)
	}
	t.Setenv("PEKY_ICON_SIZE", "medium")
	if got := ActiveSize(); got != SizeMedium {
		t.Fatalf("ActiveSize(default) = %v", got)
	}
}
