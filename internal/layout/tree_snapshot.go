package layout

// TreeSnapshot captures a layout tree in a JSON-safe format.
type TreeSnapshot struct {
	Root         NodeSnapshot `json:"root"`
	ZoomedPaneID string       `json:"zoomed_pane_id,omitempty"`
}

// NodeSnapshot captures a layout node without parent pointers.
type NodeSnapshot struct {
	PaneID   string         `json:"pane_id,omitempty"`
	Axis     Axis           `json:"axis"`
	Size     int            `json:"size"`
	Children []NodeSnapshot `json:"children,omitempty"`
}

// SnapshotTree converts a layout tree into a snapshot.
func SnapshotTree(tree *Tree) *TreeSnapshot {
	if tree == nil || tree.Root == nil {
		return nil
	}
	return &TreeSnapshot{
		Root:         snapshotNode(tree.Root),
		ZoomedPaneID: tree.ZoomedPaneID,
	}
}

func snapshotNode(node *Node) NodeSnapshot {
	if node == nil {
		return NodeSnapshot{}
	}
	snap := NodeSnapshot{
		PaneID: node.PaneID,
		Axis:   node.Axis,
		Size:   node.Size,
	}
	if len(node.Children) == 0 {
		return snap
	}
	snap.Children = make([]NodeSnapshot, 0, len(node.Children))
	for _, child := range node.Children {
		snap.Children = append(snap.Children, snapshotNode(child))
	}
	return snap
}

// TreeFromSnapshot rebuilds a layout tree from a snapshot.
func TreeFromSnapshot(snapshot *TreeSnapshot) *Tree {
	if snapshot == nil {
		return nil
	}
	panes := make(map[string]*Node)
	root := buildNodeFromSnapshot(snapshot.Root, nil, panes)
	if root == nil {
		return nil
	}
	tree := NewTree(root, panes)
	tree.ZoomedPaneID = snapshot.ZoomedPaneID
	return tree
}

func buildNodeFromSnapshot(snapshot NodeSnapshot, parent *Node, panes map[string]*Node) *Node {
	node := &Node{
		PaneID: snapshot.PaneID,
		Axis:   snapshot.Axis,
		Size:   snapshot.Size,
		Parent: parent,
	}
	if snapshot.PaneID != "" {
		panes[snapshot.PaneID] = node
	}
	if len(snapshot.Children) == 0 {
		return node
	}
	node.Children = make([]*Node, 0, len(snapshot.Children))
	for _, child := range snapshot.Children {
		node.Children = append(node.Children, buildNodeFromSnapshot(child, node, panes))
	}
	return node
}
