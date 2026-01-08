package layout

import "fmt"

const LayoutBaseSize = 1000

type Axis int

const (
	AxisHorizontal Axis = iota
	AxisVertical
)

func (a Axis) String() string {
	switch a {
	case AxisHorizontal:
		return "horizontal"
	case AxisVertical:
		return "vertical"
	default:
		return "unknown"
	}
}

type Rect struct {
	X int
	Y int
	W int
	H int
}

func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

type Node struct {
	ID       string
	PaneID   string
	Axis     Axis
	Size     int
	Parent   *Node
	Children []*Node
}

func (n *Node) IsLeaf() bool {
	return n != nil && len(n.Children) == 0
}

type Tree struct {
	Root          *Node
	Panes         map[string]*Node
	ZoomedPaneID  string
	nextNodeIndex int
}

func NewTree(root *Node, panes map[string]*Node) *Tree {
	if panes == nil {
		panes = make(map[string]*Node)
	}
	return &Tree{Root: root, Panes: panes}
}

func (t *Tree) nextNodeID() string {
	t.nextNodeIndex++
	return fmt.Sprintf("n-%d", t.nextNodeIndex)
}

type Constraints struct {
	MinWidth  int
	MinHeight int
}

func DefaultConstraints() Constraints {
	return Constraints{MinWidth: 40, MinHeight: 20}
}

type SnapConfig struct {
	Threshold  int
	Hysteresis int
	GridStep   int
	Ratios     []int
}

func DefaultSnapConfig() SnapConfig {
	return SnapConfig{
		Threshold:  12,
		Hysteresis: 6,
		GridStep:   10,
		Ratios:     []int{50, 33, 67, 25, 75},
	}
}

type SnapState struct {
	Active bool
	Target int
}
