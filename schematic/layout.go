package schematic

import (
	"fmt"
	"sort"
)

// Layout places the boards side by side and works out where every pin lands.
// It is the step that used to be done by eye: nothing below is a chosen
// number, it all falls out of the pin lists.
func (s *Schematic) Layout() (map[string]PinPos, error) {
	if errs := s.Validate(); len(errs) > 0 {
		return nil, errs[0]
	}

	gutter := s.Gutter
	if gutter == 0 {
		gutter = s.neededGutter()
	}

	// Group boards into columns. Boards sharing a Col stack vertically; the
	// column is as wide as its widest board. Declaration order decides both
	// the left-to-right order of columns and the top-to-bottom order within
	// one, so the arrangement stays a property of the description.
	type column struct {
		col    int
		boards []*Board
		w      int
	}
	var cols []*column
	byCol := map[int]*column{}
	for _, b := range s.Boards {
		c, ok := byCol[b.Col]
		if !ok {
			c = &column{col: b.Col}
			byCol[b.Col] = c
			cols = append(cols, c)
		}
		c.boards = append(c.boards, b)
	}
	sort.SliceStable(cols, func(i, j int) bool { return cols[i].col < cols[j].col })

	pos := map[string]PinPos{}
	x := 0
	for _, c := range cols {
		for _, b := range c.boards {
			w, _ := b.boxSize()
			if w > c.w {
				c.w = w
			}
		}
		y := 0
		for _, b := range c.boards {
			b.X, b.Y = x, y
			b.W, b.H = b.boxSize()
			y += b.H + 1 // one blank row between stacked boards
			placePins(pos, b)
		}
		x += c.w + gutter
	}

	// Wires that have to route around the outside of a board run in a margin
	// above everything. Count them first so the margin is exactly as deep as
	// it needs to be, then shift the whole drawing down to make room.
	margin := 0
	for _, w := range s.Wires {
		a, b := pos[w.From], pos[w.To]
		if a.X > b.X {
			a, b = b, a
		}
		if a.Side == Left || b.Side == Right {
			margin++
		}
	}
	if margin > 0 {
		margin += 2
		for _, b := range s.Boards {
			b.Y += margin
		}
		for k, p := range pos {
			p.Y += margin
			pos[k] = p
		}
	}
	return pos, nil
}

// placePins records where every pin of a placed board landed.
func placePins(pos map[string]PinPos, b *Board) {
	{
		for i, name := range b.Left {
			if name == Gap {
				continue
			}
			pos[b.Name+"."+name] = PinPos{
				Board: b, Name: name, Side: Left, Row: i,
				X: b.X, Y: b.Y + b.pinRowY(i),
			}
		}
		for i, name := range b.Right {
			if name == Gap {
				continue
			}
			pos[b.Name+"."+name] = PinPos{
				Board: b, Name: name, Side: Right, Row: i,
				X: b.X + b.W - 1, Y: b.Y + b.pinRowY(i),
			}
		}
	}
}

// boxSize is the drawn size of a board, derived from its longest pin labels.
func (b *Board) boxSize() (w, h int) {
	lw, rw := b.leftLabelWidth(), b.rightLabelWidth()
	inner := lw + rw + 8 // two 3-char sockets plus a gap between the columns
	if t := len(b.title()) + 2; t > inner {
		inner = t
	}
	// header row + pin rows + footer (title) row, plus the two border rows
	return inner + 2, b.rows() + 4
}

// pinRowY is the y offset of pin row i inside the box: one border row and one
// header row above the first pin.
func (b *Board) pinRowY(i int) int { return i + 2 }

// leftLabelWidth and rightLabelWidth are needed by the renderer to place text.
func (b *Board) leftLabelWidth() int {
	w := 0
	for _, p := range b.Left {
		if n := len(b.label(p)); n > w {
			w = n
		}
	}
	return w
}

func (b *Board) rightLabelWidth() int {
	w := 0
	for _, p := range b.Right {
		if n := len(b.label(p)); n > w {
			w = n
		}
	}
	return w
}

// point is a canvas cell.
type point struct{ X, Y int }

// route is one wire reduced to a polyline. Keeping it as a path rather than a
// fixed three-segment shape is what lets a pin on the far side of its board be
// routed around the outside instead of straight through the board art.
type route struct {
	wire     Wire
	from, to PinPos
	pts      []point
	lane     int // x of the main vertical segment (asciigraph uses this)
	y0, y1   int
	wrapped  bool
}

// routes assigns every wire a lane such that two wires whose vertical segments
// would overlap never share one. This is the job the old code did by choosing
// a breakpoint per wire by hand.
func (s *Schematic) routes(pos map[string]PinPos) ([]route, error) {
	rs := make([]route, 0, len(s.Wires))
	for _, w := range s.Wires {
		a, ok := pos[w.From]
		if !ok {
			return nil, fmt.Errorf("unplaced pin %q", w.From)
		}
		b, ok := pos[w.To]
		if !ok {
			return nil, fmt.Errorf("unplaced pin %q", w.To)
		}
		// Draw left to right: whichever pin is further left is the source.
		if a.X > b.X {
			a, b = b, a
		}
		rs = append(rs, route{wire: w, from: a, to: b, y0: a.Y, y1: b.Y})
	}

	// Assign lanes in source order. A bundle running from a block of
	// consecutive pins to another block then gets consecutive lanes and
	// nests, rather than interleaving and crossing itself — which is what
	// makes a hand-drawn diagram readable and a naive one a mesh.
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].y0 != rs[j].y0 {
			return rs[i].y0 < rs[j].y0
		}
		return rs[i].y1 < rs[j].y1
	})

	// occupied[lane] = the y intervals already used in that lane.
	type span struct{ lo, hi int }
	occupied := map[int][]span{}

	// Wrapped wires run above the boards, one row each so they nest
	// rather than overlap.
	wrapRow := -1
	for _, b := range s.Boards {
		if wrapRow < 0 || b.Y < wrapRow {
			wrapRow = b.Y
		}
	}
	wrapRow-- // first free row above the topmost board

	// Detours run below everything, one row each.
	detourRow := 0
	for _, b := range s.Boards {
		if bottom := b.Y + b.H; bottom > detourRow {
			detourRow = bottom
		}
	}
	detourRow++

	for i := range rs {
		r := &rs[i]
		lo, hi := minMax(r.y0, r.y1)
		gutterStart := r.from.X + 2
		gutterEnd := r.to.X - 2
		if gutterEnd < gutterStart {
			gutterEnd = gutterStart
		}

		placed := false
		for lane := gutterStart; lane <= gutterEnd; lane++ {
			clash := false
			for _, sp := range occupied[lane] {
				if lo <= sp.hi && sp.lo <= hi {
					clash = true
					break
				}
			}
			if clash {
				continue
			}
			occupied[lane] = append(occupied[lane], span{lo, hi})
			r.lane = lane
			placed = true
			break
		}
		if !placed {
			// Every lane is taken across this wire's span. Fall back to
			// the middle and accept a crossing rather than dropping it.
			r.lane = (gutterStart + gutterEnd) / 2
		}

		// A pin on the edge facing away from its partner cannot be reached
		// directly: the straight path would run through its own board. Send
		// it out of its own side and over the top instead. Row -1 upwards
		// is reserved margin, allocated per wrapped wire so they nest.
		r.from.exitsAway = r.from.Side == Left
		r.to.exitsAway = r.to.Side == Right
		r.wrapped = r.from.exitsAway || r.to.exitsAway

		switch {
		case r.wrapped:
			r.pts = buildPath(*r, wrapRow)
			wrapRow--
		case s.crossesBoard(r.from, r.to):
			// Nothing is wrong with either pin; a board simply stands in
			// the way. Go under everything rather than through it.
			r.pts = buildDetour(*r, detourRow)
			detourRow++
		default:
			r.pts = buildPath(*r, wrapRow)
		}
	}
	return rs, nil
}

// buildPath turns a route into the polyline the renderer draws.
func buildPath(r route, wrapRow int) []point {
	if !r.wrapped {
		return []point{
			{r.from.X + 1, r.y0},
			{r.lane, r.y0},
			{r.lane, r.y1},
			{r.to.X - 1, r.y1},
		}
	}

	pts := []point{}
	// Leave the source pin on whichever side it physically sits.
	if r.from.exitsAway {
		out := r.from.Board.X - 2
		pts = append(pts,
			point{r.from.X - 1, r.y0},
			point{out, r.y0},
			point{out, wrapRow},
		)
	} else {
		pts = append(pts,
			point{r.from.X + 1, r.y0},
			point{r.lane, r.y0},
			point{r.lane, wrapRow},
		)
	}
	// Enter the target the same way.
	if r.to.exitsAway {
		in := r.to.Board.X + r.to.Board.W + 1
		pts = append(pts,
			point{in, wrapRow},
			point{in, r.y1},
			point{r.to.X + 1, r.y1},
		)
	} else {
		pts = append(pts,
			point{r.to.X - 2, wrapRow},
			point{r.to.X - 2, r.y1},
			point{r.to.X - 1, r.y1},
		)
	}
	return pts
}

func minMax(a, b int) (int, int) {
	if a > b {
		return b, a
	}
	return a, b
}

// neededGutter works out how wide the wire channels actually have to be,
// instead of assuming the worst case of one lane per wire.
//
// Lanes are only shared by wires whose vertical runs do not overlap, so the
// requirement is the largest number of wires that are simultaneously "in
// flight" at any row — the interval-graph clique number. Laying the whole
// drawing out at worst case is what made it mostly whitespace.
func (s *Schematic) neededGutter() int {
	// Rough pin row for each pin, in board-local terms. Absolute positions
	// are not known yet, which is fine: overlap in y is what matters and
	// that is decided by row within a column, not by x.
	rowOf := map[string]int{}
	for _, b := range s.Boards {
		for i, p := range b.Left {
			if p != Gap {
				rowOf[b.Name+"."+p] = b.pinRowY(i)
			}
		}
		for i, p := range b.Right {
			if p != Gap {
				rowOf[b.Name+"."+p] = b.pinRowY(i)
			}
		}
	}

	// Sweep: +1 where a wire's span starts, -1 after it ends.
	delta := map[int]int{}
	for _, w := range s.Wires {
		a, aok := rowOf[w.From]
		b, bok := rowOf[w.To]
		if !aok || !bok {
			continue
		}
		lo, hi := minMax(a, b)
		delta[lo]++
		delta[hi+1]--
	}
	rows := make([]int, 0, len(delta))
	for r := range delta {
		rows = append(rows, r)
	}
	sort.Ints(rows)

	cur, peak := 0, 0
	for _, r := range rows {
		cur += delta[r]
		if cur > peak {
			peak = cur
		}
	}
	// Two columns of clearance either side of the lanes so wires leave and
	// enter their pins cleanly.
	return peak + 4
}

// crossesBoard reports whether a straight run between two pins would pass
// through some third board standing between them.
//
// Wrapping only handled a pin on the wrong edge of its own board. A wire from a
// supply on the far left to a peripheral on the far right has both pins facing
// the right way and still cannot go straight, because something is in the way —
// and drawing it straight puts a line through the middle of that part, which
// reads as a connection that is not there.
func (s *Schematic) crossesBoard(from, to PinPos) bool {
	lo, hi := minMax(from.Y, to.Y)
	for _, b := range s.Boards {
		if b == from.Board || b == to.Board {
			continue
		}
		left, right := b.X, b.X+b.W-1
		if right <= from.X || left >= to.X {
			continue // not between the two pins
		}
		// Overlap in y means the run would cross the box rather than pass
		// above or below it.
		if lo <= b.Y+b.H-1 && b.Y <= hi {
			return true
		}
	}
	return false
}

// buildDetour routes a wire beneath the boards: out of the source pin, down to
// a free row under everything, across, then up into the target.
func buildDetour(r route, row int) []point {
	out := r.from.X + 2
	in := r.to.X - 2
	if in < out {
		in = out
	}
	return []point{
		{r.from.X + 1, r.y0},
		{out, r.y0},
		{out, row},
		{in, row},
		{in, r.y1},
		{r.to.X - 1, r.y1},
	}
}
