package schematic

import (
	"fmt"
	"strings"
	"testing"
)

// twoBoards is a small schematic with the shapes that matter: a numbered board
// with both columns, an unnumbered one, a gap, a relabelled pin, and wires on
// more than one net.
func twoBoards() *Schematic {
	mcu := &Board{
		Name:     "mcu",
		Title:    "MCU",
		Left:     []string{"P0", "P1", Gap, "GND1"},
		Right:    []string{"VCC", "P2", "P3", "GND2"},
		FirstPin: 1,
		Labels:   map[string]string{"GND1": "GND", "GND2": "GND"},
	}
	part := &Board{
		Name:  "part",
		Title: "PART",
		Col:   1,
		Left:  []string{"IN", "PWR", "GND"},
	}
	return &Schematic{
		Boards: []*Board{mcu, part},
		Wires: []Wire{
			{From: "mcu.P0", To: "part.IN", Net: "SIG"},
			{From: "mcu.VCC", To: "part.PWR", Net: "PWR"},
			{From: "mcu.GND1", To: "part.GND", Net: "GND"},
		},
	}
}

func TestSplitRef(t *testing.T) {
	for _, tc := range []struct {
		ref, board, pin string
		wantErr         bool
	}{
		{ref: "pico.GP22", board: "pico", pin: "GP22"},
		{ref: "a.b.c", board: "a", pin: "b.c"}, // first dot wins
		{ref: "pico.", board: "pico", pin: ""},
		{ref: ".GP0", board: "", pin: "GP0"},
		{ref: "pico", wantErr: true},
		{ref: "", wantErr: true},
	} {
		b, p, err := splitRef(tc.ref)
		if tc.wantErr {
			if err == nil {
				t.Errorf("splitRef(%q) = %q, %q; want an error", tc.ref, b, p)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitRef(%q): %v", tc.ref, err)
			continue
		}
		if b != tc.board || p != tc.pin {
			t.Errorf("splitRef(%q) = %q, %q; want %q, %q", tc.ref, b, p, tc.board, tc.pin)
		}
	}
}

func TestValidateAcceptsAWellFormedSchematic(t *testing.T) {
	if errs := twoBoards().Validate(); len(errs) > 0 {
		t.Fatalf("a well-formed schematic did not validate: %v", errs)
	}
}

// A typo in a pin name is the mistake this package exists to catch: a
// drawing-first design cannot notice one at all, because it just draws a line
// to nowhere.
func TestValidateCatchesAReferenceToAPinThatDoesNotExist(t *testing.T) {
	s := twoBoards()
	s.Wires = append(s.Wires, Wire{From: "mcu.P9", To: "part.IN", Net: "SIG"})
	errs := s.Validate()
	if len(errs) == 0 {
		t.Fatal("a wire to a nonexistent pin validated")
	}
	if !containsAll(errs, "P9") {
		t.Errorf("the error does not name the bad pin: %v", errs)
	}
}

func TestValidateCatchesAReferenceToABoardThatDoesNotExist(t *testing.T) {
	s := twoBoards()
	s.Wires = append(s.Wires, Wire{From: "nosuch.P0", To: "part.IN"})
	if errs := s.Validate(); !containsAll(errs, "nosuch") {
		t.Errorf("a wire from an unknown board validated: %v", errs)
	}
}

func TestValidateCatchesMalformedReferences(t *testing.T) {
	s := twoBoards()
	s.Wires = append(s.Wires, Wire{From: "mcuP0", To: "part.IN"})
	if errs := s.Validate(); len(errs) == 0 {
		t.Error("a reference with no dot in it validated")
	}
}

func TestValidateCatchesDuplicateBoardNames(t *testing.T) {
	s := twoBoards()
	s.Boards = append(s.Boards, &Board{Name: "mcu", Left: []string{"X"}})
	if errs := s.Validate(); !containsAll(errs, "mcu") {
		t.Errorf("two boards with one name validated: %v", errs)
	}
}

func TestValidateCatchesAnUnnamedBoard(t *testing.T) {
	s := twoBoards()
	s.Boards = append(s.Boards, &Board{Left: []string{"X"}})
	if errs := s.Validate(); len(errs) == 0 {
		t.Error("a board with no name validated")
	}
}

func TestValidateCatchesDuplicatePinsOnOneBoard(t *testing.T) {
	s := twoBoards()
	s.Boards[1].Left = append(s.Boards[1].Left, "IN")
	if errs := s.Validate(); !containsAll(errs, "IN") {
		t.Errorf("a board with two pins named IN validated: %v", errs)
	}
}

// Gap is the spacer between pin groups. It appears many times on a real board
// and must not be mistaken for a duplicate pin.
func TestValidateAllowsRepeatedGaps(t *testing.T) {
	s := twoBoards()
	s.Boards[0].Left = []string{"P0", Gap, "P1", Gap, "GND1"}
	if errs := s.Validate(); len(errs) > 0 {
		t.Errorf("repeated gaps were treated as duplicate pins: %v", errs)
	}
}

// Pin numbering runs down the left column and back up the right, the way a
// datasheet numbers a DIP package. Getting the right column backwards is the
// easy mistake, so both ends of both columns are checked.
func TestPinNumberRunsDownTheLeftAndBackUpTheRight(t *testing.T) {
	b := twoBoards().Boards[0] // 4 left, 4 right, FirstPin 1
	for _, tc := range []struct {
		side Side
		i    int
		want int
	}{
		{Left, 0, 1},
		{Left, 3, 4},
		{Right, 0, 8}, // opposite the first left pin
		{Right, 3, 5}, // adjacent to the last left pin
	} {
		if got := b.pinNumber(tc.side, tc.i); got != tc.want {
			t.Errorf("pinNumber(%v, %d) = %d, want %d", tc.side, tc.i, got, tc.want)
		}
	}
}

// The two columns together must use each number exactly once, or the drawing
// disagrees with the datasheet it is meant to match.
func TestPinNumbersAreAContiguousRun(t *testing.T) {
	b := twoBoards().Boards[0]
	seen := map[int]bool{}
	for i := range b.Left {
		seen[b.pinNumber(Left, i)] = true
	}
	for i := range b.Right {
		seen[b.pinNumber(Right, i)] = true
	}
	n := len(b.Left) + len(b.Right)
	if len(seen) != n {
		t.Fatalf("%d pins produced %d distinct numbers", n, len(seen))
	}
	for want := b.FirstPin; want < b.FirstPin+n; want++ {
		if !seen[want] {
			t.Errorf("pin number %d is missing", want)
		}
	}
}

func TestPinNumberIsZeroWhenTheBoardIsUnnumbered(t *testing.T) {
	b := twoBoards().Boards[1] // no FirstPin
	if got := b.pinNumber(Left, 0); got != 0 {
		t.Errorf("an unnumbered board reported pin number %d", got)
	}
}

func TestIsRail(t *testing.T) {
	for _, name := range []string{"GND", "vcc", " 3V3 ", "VBUS", "LED+", "A0/GND"} {
		if !isRail(name) {
			t.Errorf("isRail(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"GP22", "SDA", "", "GNDX", "RS"} {
		if isRail(name) {
			t.Errorf("isRail(%q) = true, want false", name)
		}
	}
}

func TestLabelFallsBackToThePinName(t *testing.T) {
	b := twoBoards().Boards[0]
	if got := b.label("GND1"); got != "GND" {
		t.Errorf("label(GND1) = %q, want the relabelled GND", got)
	}
	if got := b.label("P0"); got != "P0" {
		t.Errorf("label(P0) = %q, want the pin name itself", got)
	}
}

func TestHasPin(t *testing.T) {
	b := twoBoards().Boards[0]
	for _, pin := range []string{"P0", "GND1", "VCC", "P3"} {
		if !b.hasPin(pin) {
			t.Errorf("hasPin(%q) = false on a pin the board has", pin)
		}
	}
	for _, pin := range []string{"P9", "", "gnd1"} {
		if b.hasPin(pin) {
			t.Errorf("hasPin(%q) = true on a pin the board does not have", pin)
		}
	}
}

// Layout is what replaced hand-tuned coordinates, so the property that matters
// is that it never puts two pins in the same place.
func TestLayoutGivesEveryPinItsOwnPosition(t *testing.T) {
	s := twoBoards()
	pos, err := s.Layout()
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	at := map[string]string{}
	for ref, p := range pos {
		key := fmt.Sprintf("%d,%d", p.X, p.Y)
		if prev, dup := at[key]; dup {
			t.Errorf("%s and %s are both at %s", prev, ref, key)
		}
		at[key] = ref
	}
	for _, ref := range []string{"mcu.P0", "mcu.VCC", "part.IN", "part.GND"} {
		if _, ok := pos[ref]; !ok {
			t.Errorf("Layout has no position for %s", ref)
		}
	}
}

// A gap is a spacer, not a pin, so it must not be laid out as one.
func TestLayoutSkipsGaps(t *testing.T) {
	pos, err := twoBoards().Layout()
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if _, ok := pos["mcu."+Gap]; ok {
		t.Error("a gap was given a pin position")
	}
}

// Every wire in the description has to reach the netlist. A wire that is
// silently dropped is the failure this table exists to make visible.
func TestNetlistListsEveryWire(t *testing.T) {
	s := twoBoards()
	out := s.Netlist()
	for _, w := range s.Wires {
		if !strings.Contains(out, w.From) || !strings.Contains(out, w.To) {
			t.Errorf("netlist is missing %s -> %s:\n%s", w.From, w.To, out)
		}
	}
	if !strings.Contains(out, "SIG") || !strings.Contains(out, "GND") {
		t.Errorf("netlist does not name the nets:\n%s", out)
	}
}

func TestNetlistIsStable(t *testing.T) {
	s := twoBoards()
	if a, b := s.Netlist(), s.Netlist(); a != b {
		t.Error("two calls to Netlist produced different tables")
	}
}

func TestRenderTextDrawsEveryBoard(t *testing.T) {
	s := twoBoards()
	out, err := s.RenderText(false)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	for _, want := range []string{"MCU", "PART", "P0", "IN"} {
		if !strings.Contains(out, want) {
			t.Errorf("the drawing does not contain %q:\n%s", want, out)
		}
	}
}

// The plain renderer is what a README or a pipe gets; it must not smuggle in
// escape sequences.
func TestRenderTextWithoutColorHasNoEscapes(t *testing.T) {
	out, err := twoBoards().RenderText(false)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("the uncolored drawing contains ANSI escapes")
	}
}

func TestRenderTextWithColorHasEscapes(t *testing.T) {
	out, err := twoBoards().RenderText(true)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Error("the colored drawing contains no ANSI escapes")
	}
}

// Rows are trimmed on the right rather than padded to a common width, which is
// what lets the drawing be pasted into a README without a block of trailing
// spaces on every line. Trailing whitespace creeping back in is the regression.
func TestRenderTextHasNoTrailingWhitespace(t *testing.T) {
	out, err := twoBoards().RenderText(false)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	for i, l := range strings.Split(out, "\n") {
		if l != strings.TrimRight(l, " \t") {
			t.Errorf("row %d ends in whitespace: %q", i, l)
		}
	}
}

// Two renders of the same description must be identical, or the committed SVG
// and the drawing in the README churn on every regeneration.
func TestRenderTextIsStable(t *testing.T) {
	s := twoBoards()
	a, err := s.RenderText(false)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	b, err := s.RenderText(false)
	if err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	if a != b {
		t.Error("two renders of one schematic differed")
	}
}

func TestRenderSVGIsWellFormed(t *testing.T) {
	out, err := twoBoards().RenderSVG()
	if err != nil {
		t.Fatalf("RenderSVG: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "<svg") {
		t.Error("the SVG does not start with an svg element")
	}
	if !strings.Contains(out, "</svg>") {
		t.Error("the svg element is not closed")
	}
	if o, c := strings.Count(out, "<g"), strings.Count(out, "</g>"); o != c {
		t.Errorf("%d group elements opened, %d closed", o, c)
	}
	for _, want := range []string{"MCU", "PART"} {
		if !strings.Contains(out, want) {
			t.Errorf("the SVG does not contain %q", want)
		}
	}
}

// The light theme exists for print and for a README on a white background, so
// it has to actually change the colors rather than just being selectable.
func TestLightThemeChangesTheOutput(t *testing.T) {
	dark := twoBoards()
	light := twoBoards()
	light.Theme = &Light

	a, err := dark.RenderSVG()
	if err != nil {
		t.Fatalf("RenderSVG: %v", err)
	}
	b, err := light.RenderSVG()
	if err != nil {
		t.Fatalf("RenderSVG (light): %v", err)
	}
	if a == b {
		t.Error("the light theme rendered identically to the dark one")
	}
}

// Validate is the gate on a nonsensical description, so the renderers are only
// asked not to fall over on one. An empty schematic draws nothing, which is the
// right answer for nothing.
func TestRenderersSurviveAnEmptySchematic(t *testing.T) {
	s := &Schematic{}
	out, err := s.RenderText(false)
	if err != nil {
		t.Fatalf("RenderText on an empty schematic: %v", err)
	}
	if strings.ContainsAny(out, "┌│└") {
		t.Errorf("an empty schematic drew a box: %q", out)
	}
	if n := s.Netlist(); strings.Count(n, "\n") > 2 {
		t.Errorf("an empty schematic produced netlist rows:\n%s", n)
	}
}

func containsAll(errs []error, subs ...string) bool {
	var all strings.Builder
	for _, err := range errs {
		all.WriteString(err.Error())
		all.WriteByte('\n')
	}
	for _, s := range subs {
		if !strings.Contains(all.String(), s) {
			return false
		}
	}
	return len(errs) > 0
}
