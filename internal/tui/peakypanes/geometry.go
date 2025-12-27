package peakypanes

type rect struct {
	X int
	Y int
	W int
	H int
}

func (r rect) empty() bool {
	return r.W <= 0 || r.H <= 0
}

func (r rect) contains(x, y int) bool {
	if r.empty() {
		return false
	}
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}
