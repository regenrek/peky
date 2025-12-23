package icons

import "testing"

func TestActiveDefaultsToUnicode(t *testing.T) {
	t.Setenv("PEAKYPANES_ICON_SET", "")
	set := Active()
	if set.WindowLabel != Unicode.WindowLabel {
		t.Fatalf("Active() = %#v, want Unicode", set.WindowLabel)
	}
}

func TestActiveUsesASCII(t *testing.T) {
	t.Setenv("PEAKYPANES_ICON_SET", "ascii")
	set := Active()
	if set.WindowLabel != ASCII.WindowLabel {
		t.Fatalf("Active() = %#v, want ASCII", set.WindowLabel)
	}
}

func TestActiveSize(t *testing.T) {
	t.Setenv("PEAKYPANES_ICON_SIZE", "small")
	if ActiveSize() != SizeSmall {
		t.Fatalf("ActiveSize(small) = %v", ActiveSize())
	}
	t.Setenv("PEAKYPANES_ICON_SIZE", "large")
	if ActiveSize() != SizeLarge {
		t.Fatalf("ActiveSize(large) = %v", ActiveSize())
	}
	t.Setenv("PEAKYPANES_ICON_SIZE", "medium")
	if ActiveSize() != SizeMedium {
		t.Fatalf("ActiveSize(medium) = %v", ActiveSize())
	}
}

func TestVariantsBySizeFallback(t *testing.T) {
	variant := Variants{Medium: "m"}
	if got := variant.BySize(SizeSmall); got != "m" {
		t.Fatalf("BySize(small) = %q", got)
	}
	if got := variant.BySize(SizeLarge); got != "m" {
		t.Fatalf("BySize(large) = %q", got)
	}
}
