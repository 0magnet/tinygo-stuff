package schematic

import (
	"fmt"
	"strings"
)

// SVG geometry. One text cell maps to one cellW × cellH box, so the SVG is the
// same layout as the text renderer at a larger, exact scale.
const (
	svgCellW = 9
	svgCellH = 16
	svgPad   = 12
)

// RenderSVG draws the schematic as real vector lines rather than box-drawing
// characters.
//
// This exists because a character grid is only as straight as the font renderer
// makes it. Box-drawing glyphs are laid out by advance width, and when that
// width is not a whole number of pixels the rounding error accumulates across
// the row — so a diagram that is a perfect grid in the terminal comes out of an
// HTML-to-image step with wires that visibly drift, worse the further right you
// look. Here a wire is a polyline between two computed points, so it is exactly
// straight no matter what renders it.
func (s *Schematic) RenderSVG() (string, error) {
	pos, err := s.Layout()
	if err != nil {
		return "", err
	}
	rs, err := s.routes(pos)
	if err != nil {
		return "", err
	}

	cols, rows := 0, 0
	for _, b := range s.Boards {
		if r := b.X + b.W; r > cols {
			cols = r
		}
		if d := b.Y + b.H; d > rows {
			rows = d
		}
	}
	// The routing decides the extent as much as the boards do: a wire that
	// wraps around the right-hand board runs past the rightmost box, and
	// sizing the canvas from the boards alone clips it off the edge.
	for _, r := range rs {
		for _, p := range r.pts {
			if p.X+1 > cols {
				cols = p.X + 1
			}
			if p.Y+1 > rows {
				rows = p.Y + 1
			}
		}
	}
	w := cols*svgCellW + 2*svgPad
	h := rows*svgCellH + 2*svgPad

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, w, h, w, h)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, w, h, s.theme().Background)
	fmt.Fprintf(&b, `<g font-family="DejaVu Sans Mono,monospace" font-size="%d" fill="%s" shape-rendering="crispEdges">`, svgCellH-4, s.theme().Foreground)

	th := s.theme()
	for _, board := range s.Boards {
		svgBoard(&b, board, th)
	}
	// Each wire is drawn twice: a wide stroke in the page color, then the
	// wire itself. The casing punches a gap in whatever was drawn earlier, so
	// a crossing reads as one wire ducking behind another instead of as a
	// junction. Without it two lines simply overlap and the reader cannot
	// tell a crossing from a connection.
	styles := s.Styles()
	for _, r := range rs {
		svgRoute(&b, r, th.Background, 4)
		svgRoute(&b, r, styleFor(styles, s.Wires, r.wire).Hex, 0)
	}

	b.WriteString(`</g></svg>`)
	return b.String(), nil
}

// cx and cy convert a cell coordinate to the center of that cell in pixels,
// which is where a wire should sit.
func cx(x int) int { return svgPad + x*svgCellW + svgCellW/2 }
func cy(y int) int { return svgPad + y*svgCellH + svgCellH/2 }

func svgBoard(b *strings.Builder, bd *Board, th *Theme) {
	x := svgPad + bd.X*svgCellW
	y := svgPad + bd.Y*svgCellH
	w := bd.W * svgCellW
	h := bd.H * svgCellH

	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="%s" stroke-width="1"/>`,
		x, y, w, h, th.Foreground)
	fmt.Fprintf(b, `<text x="%d" y="%d" fill="%s" text-anchor="middle">%s</text>`,
		x+w/2, y+h-4, th.Muted, escape(bd.title()))

	for i, p := range bd.Left {
		if p == Gap {
			continue
		}
		ty := cy(bd.Y + bd.pinRowY(i))
		// Pin tick straddling the border, the SVG equivalent of the text
		// view's ┤ — so a wire visibly lands on a pin rather than merely
		// stopping near the edge.
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
			x-3, ty, x+3, ty, th.Foreground)
		fmt.Fprintf(b, `<text x="%d" y="%d" dominant-baseline="middle">%s%s</text>`,
			x+5, ty, escape(bd.socket(p)), escape(bd.label(p)))
	}
	for i, p := range bd.Right {
		if p == Gap {
			continue
		}
		ty := cy(bd.Y + bd.pinRowY(i))
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
			x+w-3, ty, x+w+3, ty, th.Foreground)
		fmt.Fprintf(b, `<text x="%d" y="%d" text-anchor="end" dominant-baseline="middle">%s%s</text>`,
			x+w-5, ty, escape(bd.label(p)), escape(bd.socket(p)))
	}

	if bd.FirstPin != 0 {
		if n := len(bd.Left); n > 0 {
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="%s">%d</text>`, x+4, y+svgCellH-2, th.Muted, bd.pinNumber(Left, 0))
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="%s">%d</text>`, x+4, y+h-svgCellH-2, th.Muted, bd.pinNumber(Left, n-1))
		}
		if n := len(bd.Right); n > 0 {
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="%s" text-anchor="end">%d</text>`, x+w-4, y+svgCellH-2, th.Muted, bd.pinNumber(Right, 0))
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="%s" text-anchor="end">%d</text>`, x+w-4, y+h-svgCellH-2, th.Muted, bd.pinNumber(Right, n-1))
		}
	}
}

// svgRoute emits the wire as one polyline. The points are the same ones the
// text renderer walks, so both views agree by construction.
func svgRoute(b *strings.Builder, r route, color string, casing float64) {
	if len(r.pts) < 2 {
		return
	}
	pts := make([]string, 0, len(r.pts))
	for _, p := range r.pts {
		pts = append(pts, fmt.Sprintf("%d,%d", cx(p.X), cy(p.Y)))
	}
	width := 1.2
	cap := "butt"
	if casing > 0 {
		width = casing
		cap = "round"
	}
	fmt.Fprintf(b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-linecap="%s" stroke-linejoin="miter"/>`,
		strings.Join(pts, " "), color, width, cap)
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
