package schematic

import "fmt"

// Style is one wire's colour, in both the forms the renderers need. Keeping
// them together means the terminal drawing and the SVG cannot drift apart.
type Style struct {
	ANSI string // escape sequence
	Hex  string // #rrggbb
}

// swatch is a colour known by both its 256-colour index and its hex.
type swatch struct {
	idx int
	hex string
}

func (s swatch) style() Style {
	return Style{ANSI: fmt.Sprintf("\x1b[38;5;%dm", s.idx), Hex: s.hex}
}

// Theme is the palette a drawing is rendered with.
//
// A palette is only legible against the background it was chosen for. The dark
// theme leans on light greys and bright yellows, both of which nearly vanish on
// white; the light theme uses dark, saturated equivalents. Families keeps the
// same net-to-hue-band mapping in either, so a ground is grey and a supply is
// red whichever way round the page is.
type Theme struct {
	Name string

	Background string // page fill (SVG only; a terminal has its own)
	Foreground string // board outlines and pin labels
	Muted      string // titles and pin numbers

	// Families give each net a band of related hues rather than one flat
	// colour, so wires within a group stay individually traceable.
	Families map[string][]swatch

	// Spectrum is used for any net without a family of its own, and is
	// deliberately wide so adjacent wires are easy to tell apart.
	Spectrum []swatch
}

// Dark is the default, matching a terminal on black.
var Dark = Theme{
	Name:       "dark",
	Background: "#000000",
	Foreground: "#d0d0d0",
	Muted:      "#999999",
	Families: map[string][]swatch{
		"GND": {
			{251, "#c6c6c6"}, {247, "#9e9e9e"}, {243, "#767676"}, {249, "#b2b2b2"},
			{245, "#8a8a8a"}, {241, "#626262"}, {253, "#dadada"}, {239, "#4e4e4e"},
		},
		"PWR": {
			{196, "#ff0000"}, {202, "#ff5f00"}, {208, "#ff8700"}, {160, "#d70000"},
			{166, "#d75f00"}, {203, "#ff5f5f"},
		},
		"I2C": {
			{226, "#ffff00"}, {220, "#ffd700"}, {214, "#ffaf00"}, {228, "#ffff87"},
		},
	},
	Spectrum: []swatch{
		{51, "#00ffff"}, {45, "#00d7ff"}, {39, "#00afff"}, {33, "#0087ff"},
		{82, "#5fff00"}, {46, "#00ff00"}, {48, "#00ff87"}, {87, "#5fffff"},
		{201, "#ff00ff"}, {165, "#d700ff"}, {129, "#af00ff"}, {141, "#af87ff"},
		{213, "#ff87ff"}, {171, "#d75fff"}, {49, "#00ffaf"}, {75, "#5fafff"},
	},
}

// Light is for a white page — print, a light terminal, or a README.
var Light = Theme{
	Name:       "light",
	Background: "#ffffff",
	Foreground: "#222222",
	Muted:      "#666666",
	Families: map[string][]swatch{
		// Dark greys: a ground has to read as neutral without disappearing.
		"GND": {
			{240, "#585858"}, {238, "#444444"}, {242, "#6c6c6c"}, {236, "#303030"},
			{244, "#808080"}, {234, "#1c1c1c"}, {246, "#949494"}, {232, "#080808"},
		},
		"PWR": {
			{124, "#af0000"}, {160, "#d70000"}, {130, "#af5f00"}, {88, "#870000"},
			{166, "#d75f00"}, {126, "#af0087"},
		},
		// Yellow on white is unreadable; amber and olive carry the same
		// "this is the bus" signal without the glare.
		"I2C": {
			{136, "#af8700"}, {94, "#875f00"}, {100, "#878700"}, {172, "#d78700"},
		},
	},
	Spectrum: []swatch{
		{18, "#000087"}, {22, "#005f00"}, {54, "#5f0087"}, {23, "#005f5f"},
		{94, "#875f00"}, {19, "#0000af"}, {28, "#008700"}, {90, "#870087"},
		{24, "#005f87"}, {58, "#5f5f00"}, {91, "#8700af"}, {29, "#00875f"},
		{25, "#005faf"}, {55, "#5f00af"}, {92, "#8700d7"}, {30, "#008787"},
	},
}

// theme returns the configured theme, defaulting to Dark.
func (s *Schematic) theme() *Theme {
	if s.Theme != nil {
		return s.Theme
	}
	return &Dark
}

// Styles assigns every wire a colour, walking each net's family in turn so that
// two wires of the same net are never given the same shade until the family is
// exhausted. Assignment follows declaration order, so it is stable: adding a
// wire to one net cannot recolour a different net.
func (s *Schematic) Styles() []Style {
	th := s.theme()
	used := map[string]int{}
	// Nets without a family of their own share one counter, so two such
	// nets cannot both start at the top of the spectrum and hand out the
	// same colour — a bus wire and the pull-up tapping it would otherwise
	// be drawn identically.
	spectrumUsed := 0
	out := make([]Style, len(s.Wires))
	for i, w := range s.Wires {
		if fam, ok := th.Families[w.Net]; ok && len(fam) > 0 {
			n := used[w.Net]
			used[w.Net] = n + 1
			out[i] = fam[n%len(fam)].style()
			continue
		}
		out[i] = th.Spectrum[spectrumUsed%len(th.Spectrum)].style()
		spectrumUsed++
	}
	return out
}

// styleFor maps a route back to its wire's colour. Routes are reordered during
// lane assignment, so they cannot be indexed positionally.
func styleFor(styles []Style, wires []Wire, w Wire) Style {
	for i := range wires {
		if wires[i] == w {
			return styles[i]
		}
	}
	return Style{ANSI: "\x1b[37m", Hex: "#888888"}
}
