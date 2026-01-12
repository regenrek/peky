package mouse

// Rect describes a hit-test rectangle in screen coordinates.
type Rect struct {
	X int
	Y int
	W int
	H int
}

// Empty reports whether the rectangle has non-positive dimensions.
func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// Contains reports whether the point lies within the rectangle.
func (r Rect) Contains(x, y int) bool {
	if r.Empty() {
		return false
	}
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

// Selection identifies the currently selected project/session/pane.
type Selection struct {
	ProjectID string
	Session   string
	Pane      string
}

// PaneHit captures a pane hit-test match.
type PaneHit struct {
	PaneID    string
	Selection Selection
	Outer     Rect
	Topbar    Rect
	Content   Rect
}

// HeaderKind describes which header segment was clicked.
type HeaderKind int

const (
	HeaderDashboard HeaderKind = iota
	HeaderProject
	HeaderNew
)

// HeaderHit captures a header hit-test match.
type HeaderHit struct {
	Kind      HeaderKind
	ProjectID string
}
