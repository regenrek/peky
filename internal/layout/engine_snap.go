package layout

import "math"

func SnapPosition(cfg SnapConfig, desired, min, max int, state SnapState) (int, SnapState) {
	return SnapPositionWithTargets(cfg, desired, min, max, state, nil)
}

func SnapPositionWithTargets(cfg SnapConfig, desired, min, max int, state SnapState, extra []int) (int, SnapState) {
	if desired < min {
		desired = min
	}
	if desired > max {
		desired = max
	}
	if state.Active {
		if absInt(desired-state.Target) <= cfg.Hysteresis {
			return state.Target, state
		}
	}
	target, ok := nearestSnapTarget(cfg, desired, min, max, extra)
	if ok {
		return target, SnapState{Active: true, Target: target}
	}
	return desired, SnapState{}
}

func nearestSnapTarget(cfg SnapConfig, desired, min, max int, extra []int) (int, bool) {
	best := 0
	bestDist := math.MaxInt
	reset := func() {
		best = 0
		bestDist = math.MaxInt
	}
	addTarget := func(value int) {
		if value < min || value > max {
			return
		}
		dist := absInt(desired - value)
		if dist <= cfg.Threshold && dist < bestDist {
			best = value
			bestDist = dist
		}
	}
	applyGroup := func(values []int) (int, bool) {
		reset()
		for _, value := range values {
			addTarget(value)
		}
		if bestDist == math.MaxInt {
			return 0, false
		}
		return best, true
	}

	if value, ok := applyGroup([]int{(min + max) / 2}); ok {
		return value, true
	}
	if len(cfg.Ratios) > 0 {
		ratios := make([]int, 0, len(cfg.Ratios))
		for _, ratio := range cfg.Ratios {
			ratios = append(ratios, (max+min)*ratio/100)
		}
		if value, ok := applyGroup(ratios); ok {
			return value, true
		}
	}
	if cfg.GridStep > 0 {
		grid := int(math.Round(float64(desired)/float64(cfg.GridStep))) * cfg.GridStep
		if value, ok := applyGroup([]int{grid}); ok {
			return value, true
		}
	}
	if value, ok := applyGroup(extra); ok {
		return value, true
	}
	return 0, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
