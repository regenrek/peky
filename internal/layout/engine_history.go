package layout

type History struct {
	Past   []*Tree
	Future []*Tree
	Limit  int
}

func (h *History) Record(snapshot *Tree) {
	if h == nil || snapshot == nil {
		return
	}
	h.Past = append(h.Past, snapshot)
	h.Future = nil
	if h.Limit > 0 && len(h.Past) > h.Limit {
		h.Past = h.Past[len(h.Past)-h.Limit:]
	}
}

func (h *History) Undo(current *Tree) (*Tree, bool) {
	if h == nil || len(h.Past) == 0 {
		return nil, false
	}
	last := h.Past[len(h.Past)-1]
	h.Past = h.Past[:len(h.Past)-1]
	if current != nil {
		h.Future = append(h.Future, current.Clone())
	}
	return last.Clone(), true
}

func (h *History) Redo(current *Tree) (*Tree, bool) {
	if h == nil || len(h.Future) == 0 {
		return nil, false
	}
	next := h.Future[len(h.Future)-1]
	h.Future = h.Future[:len(h.Future)-1]
	if current != nil {
		h.Past = append(h.Past, current.Clone())
	}
	return next.Clone(), true
}

func (h *History) Clear() {
	if h == nil {
		return
	}
	h.Past = nil
	h.Future = nil
}
