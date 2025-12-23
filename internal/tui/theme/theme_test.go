package theme

import (
	"strings"
	"testing"
)

// TestFormatSuccess tests success message formatting
func TestFormatSuccess(t *testing.T) {
	result := FormatSuccess("Operation completed")
	if result == "" {
		t.Error("FormatSuccess should return non-empty string")
	}
	if !strings.Contains(result, "✓") {
		t.Error("FormatSuccess should contain checkmark")
	}
}

// TestFormatError tests error message formatting
func TestFormatError(t *testing.T) {
	result := FormatError("Something went wrong")
	if result == "" {
		t.Error("FormatError should return non-empty string")
	}
	if !strings.Contains(result, "✗") {
		t.Error("FormatError should contain cross mark")
	}
}

// TestFormatWarning tests warning message formatting
func TestFormatWarning(t *testing.T) {
	result := FormatWarning("Be careful")
	if result == "" {
		t.Error("FormatWarning should return non-empty string")
	}
	if !strings.Contains(result, "⚠") {
		t.Error("FormatWarning should contain warning symbol")
	}
}

// TestFormatInfo tests info message formatting
func TestFormatInfo(t *testing.T) {
	result := FormatInfo("For your information")
	if result == "" {
		t.Error("FormatInfo should return non-empty string")
	}
	if !strings.Contains(result, "ℹ") {
		t.Error("FormatInfo should contain info symbol")
	}
}

// TestStylesNotNil ensures all styles are properly initialized
func TestStylesNotNil(t *testing.T) {
	styles := map[string]interface{}{
		"App":               App,
		"Title":             Title,
		"TitleAlt":          TitleAlt,
		"HelpTitle":         HelpTitle,
		"StatusMessage":     StatusMessage,
		"StatusError":       StatusError,
		"StatusWarning":     StatusWarning,
		"Dialog":            Dialog,
		"DialogTitle":       DialogTitle,
		"DialogLabel":       DialogLabel,
		"DialogValue":       DialogValue,
		"DialogNote":        DialogNote,
		"DialogChoiceKey":   DialogChoiceKey,
		"DialogChoiceSep":   DialogChoiceSep,
		"ListSelectedTitle": ListSelectedTitle,
		"ListSelectedDesc":  ListSelectedDesc,
		"ListDimmed":        ListDimmed,
		"ShortcutKey":       ShortcutKey,
		"ShortcutDesc":      ShortcutDesc,
		"ShortcutNote":      ShortcutNote,
		"ShortcutHint":      ShortcutHint,
		"LogoStyle":         LogoStyle,
		"ErrorBox":          ErrorBox,
		"ErrorTitle":        ErrorTitle,
		"ErrorMessage":      ErrorMessage,
	}

	for name, style := range styles {
		if style == nil {
			t.Errorf("Style %s should not be nil", name)
		}
	}
}

// TestColorsAreDefined ensures all colors are properly defined
func TestColorsAreDefined(t *testing.T) {
	colors := map[string]interface{}{
		"Accent":        Accent,
		"AccentSoft":    AccentSoft,
		"AccentAlt":     AccentAlt,
		"TextPrimary":   TextPrimary,
		"TextSecondary": TextSecondary,
		"TextMuted":     TextMuted,
		"TextDim":       TextDim,
		"Surface":       Surface,
		"SurfaceAlt":    SurfaceAlt,
		"SurfaceMuted":  SurfaceMuted,
		"Border":        Border,
		"BorderFocused": BorderFocused,
		"Logo":          Logo,
	}

	for name, color := range colors {
		if color == nil {
			t.Errorf("Color %s should not be nil", name)
		}
	}
}

// TestStyleRenderNotPanic ensures styles can render without panicking
func TestStyleRenderNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Style.Render() panicked: %v", r)
		}
	}()

	// Test various style renders
	_ = App.Render("test")
	_ = Title.Render("test")
	_ = StatusMessage.Render("test")
	_ = Dialog.Render("test")
	_ = LogoStyle.Render("test")
	_ = ErrorBox.Render("test")
}
