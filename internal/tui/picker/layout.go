package picker

// LayoutChoice represents a selectable layout in the picker.
type LayoutChoice struct {
	Label      string
	Desc       string
	LayoutName string
}

func (l LayoutChoice) Title() string       { return l.Label }
func (l LayoutChoice) Description() string { return l.Desc }
func (l LayoutChoice) FilterValue() string { return l.Label }
