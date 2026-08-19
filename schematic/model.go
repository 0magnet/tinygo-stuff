// Package schematic draws board-to-board wiring diagrams from a description of
// what is connected to what, rather than from hand-placed drawing commands.
//
// The distinction matters. In a drawing-first design a pin exists three times —
// as a label in some ASCII art, as a line number in that art, and as a
// coordinate in the wire that lands on it — and nothing keeps those three in
// step. Moving one row of art silently points every wire somewhere else, and
// the only way to notice is to look at the picture.
//
// Here a pin exists once, as a name on a Board. Wires refer to pins by name
// ("pico.GP22"), positions are computed, and a reference to a pin that does not
// exist is an error rather than a plausible-looking diagram.
package schematic

import (
	"fmt"
	"sort"
	"strings"
)

// Side is which edge of a board a pin sits on.
type Side int

// The two pin columns of a board.
const (
	Left Side = iota
	Right
)

// Gap is a placeholder in a pin column: it occupies a row but is not a pin and
// cannot be wired. Use it to line pins up with the physical part.
const Gap = ""

// Board is one component. Left and Right are the pin names in each column,
// listed top to bottom; use Gap for a blank row.
type Board struct {
	Name  string // referenced by wires, e.g. "pico"
	Title string // shown on the drawing; defaults to Name
	Left  []string
	Right []string

	// Col is the horizontal slot this board occupies. Boards sharing a Col
	// are stacked vertically, which is how a microcontroller ends up with
	// two peripherals down one side of it. Left to zero for every board,
	// they are simply placed left to right in declaration order.
	Col int

	// Labels overrides what is printed for a pin, keyed by pin name. A wire
	// has to name exactly one pin, so a part with eight grounds needs eight
	// distinct names — but the silkscreen says GND on all eight, and the
	// drawing should too. Without this the bookkeeping leaks into the
	// picture and invents pins that do not exist on the hardware.
	Labels map[string]string

	// Passive marks a part that does not drive a signal — a resistor
	// network, a supply rail, a connector. A pin may carry several wires
	// when the extra ones go to a passive: a pull-up tapping a bus line is
	// normal, two GPIOs wired together is not.
	Passive bool

	// FirstPin, when non-zero, numbers the pins on the drawing the way the
	// datasheet does: down the left column from FirstPin, then up the right
	// column, as a DIP is numbered.
	FirstPin int

	// Computed by Layout. Origin is the top-left of the box.
	X, Y, W, H int
}

// Wire is a connection between two pins, written as "board.pin".
//
// Net groups wires that carry the same signal, which is what colours them. It
// is deliberately not derived from the pin names: "pico.GND" to "lcd.GND" and
// "pico.GP22" to "lcd.BIT0" are both wires, but only the first is a power net.
type Wire struct {
	From, To string
	Net      string
}

// Schematic is the whole drawing: the boards and what joins them.
type Schematic struct {
	Boards []*Board
	Wires  []Wire

	// Gutter is how many columns to leave between boards for wires to run
	// in. Zero means pick one from the wire count.
	Gutter int

	// Theme is the palette to draw with. Nil means Dark.
	Theme *Theme
}

// PinPos is where a pin ended up after layout.
type PinPos struct {
	Board *Board
	Name  string
	Side  Side
	Row   int // index within its column
	X, Y  int // absolute cell in the canvas

	// exitsAway is set during routing when this pin sits on the edge
	// facing away from its partner, so the wire must leave the board
	// on its own side and route around the outside.
	exitsAway bool
}

// board returns the named board, or nil.
func (s *Schematic) board(name string) *Board {
	for _, b := range s.Boards {
		if b.Name == name {
			return b
		}
	}
	return nil
}

// splitRef splits "pico.GP22" into its parts. Pin names may not contain a dot;
// board names may not either.
func splitRef(ref string) (board, pin string, err error) {
	i := strings.IndexByte(ref, '.')
	if i < 0 {
		return "", "", fmt.Errorf("pin reference %q is not of the form board.pin", ref)
	}
	return ref[:i], ref[i+1:], nil
}

// Validate reports every problem it can find, rather than the first. A typo in
// a pin name is the most common mistake and the one a drawing-first design
// cannot catch at all.
func (s *Schematic) Validate() []error {
	var errs []error

	seen := map[string]bool{}
	for _, b := range s.Boards {
		if b.Name == "" {
			errs = append(errs, fmt.Errorf("a board has no Name"))
			continue
		}
		if seen[b.Name] {
			errs = append(errs, fmt.Errorf("two boards are named %q", b.Name))
		}
		seen[b.Name] = true

		pins := map[string]bool{}
		for _, col := range [][]string{b.Left, b.Right} {
			for _, p := range col {
				if p == Gap {
					continue
				}
				if pins[p] {
					errs = append(errs, fmt.Errorf("board %q has two pins named %q", b.Name, p))
				}
				pins[p] = true
			}
		}
	}

	// Count wires per pin so a pin driven twice is visible. This is a
	// warning in spirit, but returning it as an error keeps one code path.
	uses := map[string]int{}
	passive := map[string]bool{}
	for _, b := range s.Boards {
		passive[b.Name] = b.Passive
	}

	for i, w := range s.Wires {
		for _, ref := range []string{w.From, w.To} {
			bn, pn, err := splitRef(ref)
			if err != nil {
				errs = append(errs, fmt.Errorf("wire %d: %w", i, err))
				continue
			}
			b := s.board(bn)
			if b == nil {
				errs = append(errs, fmt.Errorf("wire %d: no board named %q", i, bn))
				continue
			}
			if !b.hasPin(pn) {
				errs = append(errs, fmt.Errorf("wire %d: board %q has no pin %q", i, bn, pn))
				continue
			}
			// A wire that lands on a passive does not count as a driver at
			// either end.
			if !passive[bn] && !endsOnPassive(s, w, passive) {
				uses[ref]++
			}
		}
		if w.From == w.To {
			errs = append(errs, fmt.Errorf("wire %d: %q connects to itself", i, w.From))
		}
	}

	// Report multiply-driven pins in a stable order.
	var multi []string
	for ref, n := range uses {
		if n > 1 && !isBusPin(ref) {
			multi = append(multi, fmt.Sprintf("%s (%d wires)", ref, n))
		}
	}
	sort.Strings(multi)
	for _, m := range multi {
		errs = append(errs, fmt.Errorf("pin carries more than one wire: %s", m))
	}

	return errs
}

// isBusPin reports whether a pin is expected to carry several wires. Power and
// ground legitimately fan out; a GPIO does not.
func isBusPin(ref string) bool {
	_, pin, err := splitRef(ref)
	if err != nil {
		return false
	}
	switch strings.ToUpper(pin) {
	case "GND", "VCC", "3V3", "5V", "VBUS", "VSYS", "VIN":
		return true
	}
	return false
}

func (b *Board) hasPin(name string) bool {
	for _, col := range [][]string{b.Left, b.Right} {
		for _, p := range col {
			if p != Gap && p == name {
				return true
			}
		}
	}
	return false
}

// title is what to print on the box.
func (b *Board) title() string {
	if b.Title != "" {
		return b.Title
	}
	return b.Name
}

// rows is the pin-column height of the board.
func (b *Board) rows() int {
	n := len(b.Left)
	if len(b.Right) > n {
		n = len(b.Right)
	}
	return n
}

// label is what to print for a pin: the override if there is one, else the
// name itself.
func (b *Board) label(pin string) string {
	if s, ok := b.Labels[pin]; ok {
		return s
	}
	return pin
}

// socket is the connector glyph drawn at the board edge. Ground and power get
// a square marker and signals a round one, which is how the original diagram
// let you find the rails at a glance.
func (b *Board) socket(pin string) string {
	if isRail(b.label(pin)) {
		return "[ ]"
	}
	return "( )"
}

// isRail reports whether a label names a supply or ground rather than a signal.
//
// Labels may carry a role after a slash — "5V/VSYS" says which of a board's two
// 5 V pins this is — so each segment is tested. Matching the whole string would
// quietly demote a supply pin to a signal the moment it was disambiguated.
func isRail(label string) bool {
	for _, part := range strings.Split(label, "/") {
		switch strings.ToUpper(strings.TrimSpace(part)) {
		case "GND", "VCC", "VDD", "VSS", "3V3", "5V", "VBUS", "VSYS", "VIN",
			"LED+", "LED-", "AREF":
			return true
		}
	}
	return false
}

// pinNumber returns the datasheet pin number for a column entry, numbering
// down the left column and back up the right, or 0 if the board is unnumbered.
func (b *Board) pinNumber(side Side, i int) int {
	if b.FirstPin == 0 {
		return 0
	}
	if side == Left {
		return b.FirstPin + i
	}
	// Right column counts back from the far end.
	return b.FirstPin + len(b.Left) + len(b.Right) - 1 - i
}

// endsOnPassive reports whether either end of the wire is a passive part.
func endsOnPassive(s *Schematic, w Wire, passive map[string]bool) bool {
	for _, ref := range []string{w.From, w.To} {
		if bn, _, err := splitRef(ref); err == nil && passive[bn] {
			return true
		}
	}
	return false
}
