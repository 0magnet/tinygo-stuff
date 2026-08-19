package schematic

import (
	"fmt"
	"sort"
	"strings"
)

// canvas is a grid of cells that can be drawn into in any order, which is what
// lets a wire leave a pin in whatever direction it needs. A plotting library
// cannot do this: it can only draw y as a function of x, so a wire can never
// double back or route around anything.
type canvas struct {
	cell  [][]rune
	color [][]string // ANSI prefix per cell, "" for none
	w, h  int
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h}
	c.cell = make([][]rune, h)
	c.color = make([][]string, h)
	for y := range c.cell {
		c.cell[y] = make([]rune, w)
		c.color[y] = make([]string, w)
		for x := range c.cell[y] {
			c.cell[y][x] = ' '
		}
	}
	return c
}

func (c *canvas) set(x, y int, r rune, col string) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	// Where two wires cross, leave the one already drawn intact and let the
	// arriving one break. That is the convention for "passes behind", and it
	// is the whole point: a '┼' here would read as a junction, which is the
	// opposite of what a crossing means.
	if existing := c.cell[y][x]; existing == '─' && r == '│' {
		return
	}
	c.cell[y][x] = r
	c.color[y][x] = col
}

func (c *canvas) text(x, y int, s, col string) {
	for i, r := range s {
		c.set(x+i, y, r, col)
	}
}

func (c *canvas) String(colorize bool) string {
	var b strings.Builder
	for y := 0; y < c.h; y++ {
		line := strings.Builder{}
		cur := ""
		for x := 0; x < c.w; x++ {
			col := c.color[y][x]
			if colorize && col != cur {
				line.WriteString("\x1b[0m")
				line.WriteString(col)
				cur = col
			}
			line.WriteRune(c.cell[y][x])
		}
		if colorize && cur != "" {
			line.WriteString("\x1b[0m")
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// Colour lives in colors.go, where a theme carries the ANSI and hex forms
// together so this renderer and the SVG one cannot drift apart.

// RenderText draws the schematic as text. Set colorize to emit ANSI colour.
func (s *Schematic) RenderText(colorize bool) (string, error) {
	pos, err := s.Layout()
	if err != nil {
		return "", err
	}
	rs, err := s.routes(pos)
	if err != nil {
		return "", err
	}

	w, h := 0, 0
	for _, b := range s.Boards {
		if r := b.X + b.W; r > w {
			w = r
		}
		if d := b.Y + b.H; d > h {
			h = d
		}
	}
	// Size to what the routing actually used, not to the worst case: a
	// single detour must not leave a page of blank rows under the drawing.
	//
	// Both axes, not just the height. A wire that wraps around the right-hand
	// board runs past the rightmost box, and the canvas silently discards
	// out-of-bounds writes — so sizing from the boards alone truncates that
	// wire into a drawing that still looks finished.
	for _, r := range rs {
		for _, p := range r.pts {
			if p.X+1 > w {
				w = p.X + 1
			}
			if p.Y+1 > h {
				h = p.Y + 1
			}
		}
	}
	c := newCanvas(w+2, h+1)

	for _, b := range s.Boards {
		drawBoard(c, b)
	}

	styles := s.Styles()
	for _, r := range rs {
		drawRoute(c, r, styleFor(styles, s.Wires, r.wire).ANSI)
	}
	return c.String(colorize), nil
}

func drawBoard(c *canvas, b *Board) {
	// Border.
	for x := 0; x < b.W; x++ {
		c.set(b.X+x, b.Y, '─', "")
		c.set(b.X+x, b.Y+b.H-1, '─', "")
	}
	for y := 0; y < b.H; y++ {
		c.set(b.X, b.Y+y, '│', "")
		c.set(b.X+b.W-1, b.Y+y, '│', "")
	}
	c.set(b.X, b.Y, '┌', "")
	c.set(b.X+b.W-1, b.Y, '┐', "")
	c.set(b.X, b.Y+b.H-1, '└', "")
	c.set(b.X+b.W-1, b.Y+b.H-1, '┘', "")

	// Title, centred on the bottom row.
	t := b.title()
	c.text(b.X+(b.W-len(t))/2, b.Y+b.H-1, t, "\x1b[1m")

	lw := b.leftLabelWidth()
	for i, p := range b.Left {
		if p == Gap {
			continue
		}
		y := b.Y + b.pinRowY(i)
		c.set(b.X, y, '┤', "")
		c.text(b.X+1, y, b.socket(p), "")
		c.text(b.X+4, y, fmt.Sprintf("%-*s", lw, b.label(p)), "")
	}
	rw := b.rightLabelWidth()
	for i, p := range b.Right {
		if p == Gap {
			continue
		}
		y := b.Y + b.pinRowY(i)
		c.set(b.X+b.W-1, y, '├', "")
		c.text(b.X+b.W-4, y, b.socket(p), "")
		c.text(b.X+b.W-4-rw, y, fmt.Sprintf("%*s", rw, b.label(p)), "")
	}

	// Datasheet pin numbers at the corners, as the original marked 1/40 and
	// 20/21 on the Pico.
	if b.FirstPin != 0 {
		if n := len(b.Left); n > 0 {
			c.text(b.X+1, b.Y+1, fmt.Sprint(b.pinNumber(Left, 0)), "")
			c.text(b.X+1, b.Y+b.H-2, fmt.Sprint(b.pinNumber(Left, n-1)), "")
		}
		if n := len(b.Right); n > 0 {
			hi := fmt.Sprint(b.pinNumber(Right, 0))
			lo := fmt.Sprint(b.pinNumber(Right, n-1))
			c.text(b.X+b.W-1-len(hi), b.Y+1, hi, "")
			c.text(b.X+b.W-1-len(lo), b.Y+b.H-2, lo, "")
		}
	}
}

// drawRoute walks the polyline, drawing each segment and the corner where the
// direction changes. Because the path is a list of points rather than a fixed
// shape, a wire that has to go around the outside of a board draws with the
// same code as one that goes straight across.
func drawRoute(c *canvas, r route, col string) {
	pts := r.pts
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		a, b := pts[i], pts[i+1]
		switch {
		case a.Y == b.Y:
			lo, hi := minMax(a.X, b.X)
			for x := lo; x <= hi; x++ {
				c.set(x, a.Y, '─', col)
			}
		case a.X == b.X:
			lo, hi := minMax(a.Y, b.Y)
			for y := lo; y <= hi; y++ {
				c.set(a.X, y, '│', col)
			}
		}
	}
	// Corners: at each interior point, pick the elbow joining the incoming
	// and outgoing directions.
	for i := 1; i < len(pts)-1; i++ {
		p, prev, next := pts[i], pts[i-1], pts[i+1]
		c.set(p.X, p.Y, elbow(prev, p, next), col)
	}
}

// elbow returns the box-drawing character joining the segment arriving from
// prev with the one leaving toward next.
func elbow(prev, p, next point) rune {
	up := prev.Y < p.Y || next.Y < p.Y
	down := prev.Y > p.Y || next.Y > p.Y
	left := prev.X < p.X || next.X < p.X
	right := prev.X > p.X || next.X > p.X

	switch {
	case up && down:
		return '│'
	case left && right:
		return '─'
	case down && right:
		return '┌'
	case down && left:
		return '┐'
	case up && right:
		return '└'
	case up && left:
		return '┘'
	}
	return '·'
}

// Netlist renders the connections as a table. Same model, so it cannot
// disagree with the drawing — which is the point of keeping connectivity as
// data.
func (s *Schematic) Netlist() string {
	byNet := map[string][]Wire{}
	for _, w := range s.Wires {
		byNet[w.Net] = append(byNet[w.Net], w)
	}
	nets := make([]string, 0, len(byNet))
	for n := range byNet {
		nets = append(nets, n)
	}
	sort.Strings(nets)

	var b strings.Builder
	fmt.Fprintf(&b, "%-6s  %-16s  %s\n", "NET", "FROM", "TO")
	fmt.Fprintf(&b, "%-6s  %-16s  %s\n", "---", "----", "--")
	for _, n := range nets {
		for _, w := range byNet[n] {
			fmt.Fprintf(&b, "%-6s  %-16s  %s\n", n, w.From, w.To)
		}
	}
	return b.String()
}
