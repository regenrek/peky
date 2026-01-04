package vt

import (
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
)

const cellStorePageSize = 2048

var cellPagePool = sync.Pool{
	New: func() any {
		return make([]uv.Cell, 0, cellStorePageSize)
	},
}

type cellStore struct {
	pages [][]uv.Cell

	// baseAbs is the absolute cell index of the first kept cell.
	baseAbs uint64
	// nextAbs is the absolute cell index for the next append.
	nextAbs uint64

	// frontSkip is the number of dropped cells within pages[0].
	frontSkip int
}

func newCellStore() cellStore {
	return cellStore{}
}

func (s *cellStore) NextAbs() uint64 {
	return s.nextAbs
}

func (s *cellStore) Len() int {
	if s.nextAbs <= s.baseAbs {
		return 0
	}
	return int(s.nextAbs - s.baseAbs)
}

func (s *cellStore) AppendCell(c uv.Cell) {
	if len(s.pages) == 0 || len(s.pages[len(s.pages)-1]) >= cellStorePageSize {
		page := cellPagePool.Get().([]uv.Cell)
		page = page[:0]
		s.pages = append(s.pages, page)
	}

	last := len(s.pages) - 1
	s.pages[last] = append(s.pages[last], c)
	s.nextAbs++
}

func (s *cellStore) CellAt(abs uint64) (uv.Cell, bool) {
	if abs < s.baseAbs || abs >= s.nextAbs {
		return uv.Cell{}, false
	}

	rel := abs - s.baseAbs
	pos := uint64(s.frontSkip) + rel
	pageIdx := int(pos / cellStorePageSize)
	inPage := int(pos % cellStorePageSize)
	if pageIdx < 0 || pageIdx >= len(s.pages) {
		return uv.Cell{}, false
	}
	page := s.pages[pageIdx]
	if inPage < 0 || inPage >= len(page) {
		return uv.Cell{}, false
	}
	return page[inPage], true
}

func (s *cellStore) DropPrefix(n int) {
	if n <= 0 || len(s.pages) == 0 {
		return
	}
	kept := s.Len()
	if n >= kept {
		s.Reset()
		return
	}

	s.baseAbs += uint64(n)
	s.frontSkip += n

	for len(s.pages) > 0 {
		if s.frontSkip < len(s.pages[0]) {
			break
		}
		s.frontSkip -= len(s.pages[0])
		cellPagePool.Put(s.pages[0][:0])
		s.pages[0] = nil
		s.pages = s.pages[1:]
	}
	if len(s.pages) == 0 {
		s.frontSkip = 0
	}
}

func (s *cellStore) Reset() {
	for i := range s.pages {
		if s.pages[i] != nil {
			cellPagePool.Put(s.pages[i][:0])
			s.pages[i] = nil
		}
	}
	s.pages = nil
	s.baseAbs = 0
	s.nextAbs = 0
	s.frontSkip = 0
}
