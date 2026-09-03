package schematic

import (
	"strings"

	"github.com/guptarohit/asciigraph"
)

// RenderAsciigraph draws the wire field with asciigraph, one series per wire.
//
// This is the original approach kept deliberately: asciigraph makes a good
// bundle of wires. What changes is where the numbers come from. Every series
// below is derived from the pin rows worked out by Layout and the lane chosen
// by routes — previously each wire needed two y levels and a breakpoint found
// by trial and error, and a color whose position in a list had to be counted
// by hand.
//
// The limitation is worth stating plainly, because it is why RenderText exists
// too: a series is y as a function of x, so a wire can only step from one row
// to another. It cannot double back, and two wires that want the same lane
// simply overlap. For a left-to-right harness that is fine, and it is what the
// pico/LCD/RTC diagram always was.
func (s *Schematic) RenderAsciigraph(opts ...asciigraph.Option) (string, error) {
	pos, err := s.Layout()
	if err != nil {
		return "", err
	}
	rs, err := s.routes(pos)
	if err != nil {
		return "", err
	}
	if len(rs) == 0 {
		return "", nil
	}

	// Work out the x span the wires live in, and the row range they touch.
	minX, maxX := rs[0].from.X, rs[0].to.X
	for _, r := range rs {
		if r.from.X < minX {
			minX = r.from.X
		}
		if r.to.X > maxX {
			maxX = r.to.X
		}
	}
	width := maxX - minX + 1
	if width < 2 {
		width = 2
	}

	// asciigraph plots larger values higher, but rows count downwards, so
	// negate. One series per wire, stepping at its lane.
	data := make([][]float64, len(rs))
	colors := make([]asciigraph.AnsiColor, len(rs))
	seen := map[string]asciigraph.AnsiColor{}

	for i, r := range rs {
		step := r.lane - minX
		if step < 0 {
			step = 0
		}
		if step > width-1 {
			step = width - 1
		}
		series := make([]float64, width)
		for x := 0; x < width; x++ {
			if x < step {
				series[x] = float64(-r.y0)
			} else {
				series[x] = float64(-r.y1)
			}
		}
		data[i] = series
		colors[i] = graphColorFor(r.wire.Net, seen)
	}

	o := append([]asciigraph.Option{
		asciigraph.Precision(0),
		asciigraph.SeriesColors(colors...),
	}, opts...)

	out := asciigraph.PlotMany(data, o...)

	// asciigraph prefixes each line with an axis gutter. The old code sliced
	// it off with a hand-measured constant (j[8:]); measure it instead.
	return trimAxis(out), nil
}

// trimAxis removes asciigraph's y-axis label column.
//
// The old code did this with a hand-measured byte offset (j[8:]), which worked
// only because that plot was pure ASCII. The axis is drawn with box-drawing
// runes and the series carry ANSI color, so both a byte offset and a rune
// offset are wrong. Measure in visible columns instead, and cut the same
// number of visible columns while passing escape sequences through.
func trimAxis(plot string) string {
	lines := strings.Split(plot, "\n")

	cut := -1
	for _, ln := range lines {
		if col := visibleIndexAny(ln, "┤┼┐└┘├"); col >= 0 {
			if cut < 0 || col < cut {
				cut = col
			}
		}
	}
	if cut < 0 {
		return plot
	}
	for i, ln := range lines {
		lines[i] = cutVisible(ln, cut+1)
	}
	return strings.Join(lines, "\n")
}

// visibleIndexAny returns the visible column of the first rune from chars,
// skipping ANSI escape sequences, or -1.
func visibleIndexAny(s, chars string) int {
	col := 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\x1b' {
			for i < len(rs) && rs[i] != 'm' {
				i++
			}
			continue
		}
		if strings.ContainsRune(chars, rs[i]) {
			return col
		}
		col++
	}
	return -1
}

// cutVisible drops the first n visible columns, keeping any escape sequences
// so color state is not lost.
func cutVisible(s string, n int) string {
	var b strings.Builder
	col := 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\x1b' {
			start := i
			for i < len(rs) && rs[i] != 'm' {
				i++
			}
			if i < len(rs) {
				b.WriteString(string(rs[start : i+1]))
			}
			continue
		}
		if col >= n {
			b.WriteRune(rs[i])
		}
		col++
	}
	return b.String()
}

var graphNetColors = map[string]asciigraph.AnsiColor{
	"GND": asciigraph.Gray,
	"PWR": asciigraph.Red,
	"I2C": asciigraph.Yellow,
	"CTL": asciigraph.Magenta,
	"DAT": asciigraph.Cyan,
}

var graphFallback = []asciigraph.AnsiColor{
	asciigraph.Green, asciigraph.Blue, asciigraph.Cyan,
	asciigraph.Magenta, asciigraph.Yellow, asciigraph.Red,
}

func graphColorFor(net string, seen map[string]asciigraph.AnsiColor) asciigraph.AnsiColor {
	if c, ok := graphNetColors[net]; ok {
		return c
	}
	if c, ok := seen[net]; ok {
		return c
	}
	c := graphFallback[len(seen)%len(graphFallback)]
	seen[net] = c
	return c
}
