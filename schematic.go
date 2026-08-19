// Command schematic draws the pico + HD44780 + DS1307 wiring diagram.
//
//	go run schematic.go            text renderer, colour
//	go run schematic.go -graph     asciigraph renderer
//	go run schematic.go -netlist   connection table
//
// Everything below the flag parsing is a description of the hardware. There is
// no drawing code here at all: boards are pin names, wires are pairs of pin
// references, and the package works out the geometry. Retargeting this to
// different hardware means editing the data, not re-tuning coordinates.
//
// The wiring matches main.go: the LCD data bus on GP22..GP15, RS on GP26, EN on
// GP27, contrast on GP28, and the DS1307 on I2C0 (GP0/GP1).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/0magnet/tinygo-stuff/schematic"
)

// Nets. Naming them is what colours the drawing and groups the netlist.
const (
	netGND = "GND"
	netPWR = "PWR"
	netI2C = "I2C"
	netCTL = "CTL"
	netDAT = "DAT"
)

func main() {
	graph := flag.Bool("graph", false, "render the wire field with asciigraph")
	netlist := flag.Bool("netlist", false, "print the connection table instead of a drawing")
	plain := flag.Bool("plain", false, "no ANSI colour")
	svg := flag.String("svg", "", "write an SVG to this path instead of drawing to the terminal")
	light := flag.Bool("light", false, "use the light-background palette (for print, a white terminal, or a README)")
	flag.Parse()

	s := build()

	if *light {
		s.Theme = &schematic.Light
	}

	if errs := s.Validate(); len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "schematic:", err)
		}
		os.Exit(1)
	}

	switch {
	case *svg != "":
		out, err := s.RenderSVG()
		if err != nil {
			fmt.Fprintln(os.Stderr, "schematic:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*svg, []byte(out), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "schematic:", err)
			os.Exit(1)
		}
	case *netlist:
		fmt.Print(s.Netlist())
	case *graph:
		out, err := s.RenderAsciigraph()
		if err != nil {
			fmt.Fprintln(os.Stderr, "schematic:", err)
			os.Exit(1)
		}
		fmt.Println(out)
	default:
		out, err := s.RenderText(!*plain)
		if err != nil {
			fmt.Fprintln(os.Stderr, "schematic:", err)
			os.Exit(1)
		}
		fmt.Print(out)
	}
}

func build() *schematic.Schematic {
	pico := &schematic.Board{
		Name:  "pico",
		Title: "Pi-Pico",
		Left: []string{
			"GP0", "GP1", "GND1", "GP2", "GP3", "GP4", "GP5", "GND2",
			"GP6", "GP7", "GP8", "GP9", "GND3", "GP10", "GP11", "GP12",
			"GP13", "GND4", "GP14", "GP15",
		},
		Right: []string{
			"VBUS", "VSYS", "GND5", "3V3EN", "3V3", "AREF", "GP28", "GND6",
			"GP27", "GP26", "RUN", "GP22", "GND7", "GP21", "GP20", "GP19",
			"GP18", "GND8", "GP17", "GP16",
		},
	}
	// The Pico has eight pins silkscreened GND. They need distinct names
	// so a wire can say which one it lands on, but the drawing should show
	// what is printed on the board.
	pico.Labels = map[string]string{}
	for i := 1; i <= 8; i++ {
		pico.Labels[fmt.Sprintf("GND%d", i)] = "GND"
	}
	for n, l := range map[string]string{
		"GP0": "GP0/U0Rx", "GP1": "GP1/U0Tx",
		"GP26": "A0/GP26", "GP27": "A1/GP27", "GP28": "A2/GP28",
	} {
		pico.Labels[n] = l
	}
	pico.FirstPin = 1

	lcd := &schematic.Board{
		Name:  "lcd",
		Title: "HD44780",
		Left: []string{
			"GND", "VCC", "CONT", "RS", "RW", "EN",
			"BIT0", "BIT1", "BIT2", "BIT3",
			"BIT4", "BIT5", "BIT6", "BIT7",
			"LED+", "LED-",
		},
	}

	// The DS1307 talks to I2C0 on GP0/GP1, which are on the Pico's left
	// column, so it is placed to the left with its pins facing right.
	// Board order and pin side are the layout: get them to match the
	// physical arrangement and almost every wire routes straight across.
	// The one that cannot is 3V3, which is genuinely on the far side of the
	// Pico — that wire routes around the outside, as it would in reality.
	rtc := &schematic.Board{
		Name:  "rtc",
		Title: "DS1307",
		Right: []string{"GND", "VCC", "SDA", "SCL", "SQW"},
	}

	// The default build drives two displays (see multidisplays in main.go).
	// The second hangs off the Pico's left column, so it shares column 0
	// with the RTC and stacks underneath it — which is how the original
	// hand-drawn diagram arranged them.
	lcd2 := &schematic.Board{
		Name:  "lcd2",
		Title: "HD44780 #2",
		Col:   0,
		Right: []string{
			"GND", "VCC", "CONT", "RS", "RW", "EN",
			"BIT0", "BIT1", "BIT2", "BIT3",
			"BIT4", "BIT5", "BIT6", "BIT7",
			"LED+", "LED-",
		},
	}
	rtc.Col = 0
	pico.Col = 1
	lcd.Col = 2

	s := &schematic.Schematic{Boards: []*schematic.Board{rtc, lcd2, pico, lcd}}

	// LCD 8-bit data bus. Bit n is driven by the nth entry of DataPins in
	// main.go, which counts down from GP22.
	dataPins := []string{"GP22", "GP21", "GP20", "GP19", "GP18", "GP17", "GP16", "GP15"}
	for i, p := range dataPins {
		s.Wires = append(s.Wires, schematic.Wire{
			From: "pico." + p,
			To:   fmt.Sprintf("lcd.BIT%d", i),
			Net:  netDAT,
		})
	}

	s.Wires = append(s.Wires,
		// LCD control.
		schematic.Wire{From: "pico.GP26", To: "lcd.RS", Net: netCTL},
		schematic.Wire{From: "pico.GP27", To: "lcd.EN", Net: netCTL},
		schematic.Wire{From: "pico.GP28", To: "lcd.CONT", Net: netCTL},
		// RW is tied low: this display is only ever written to.
		schematic.Wire{From: "pico.GND8", To: "lcd.RW", Net: netGND},

		// LCD power and backlight.
		schematic.Wire{From: "pico.VBUS", To: "lcd.VCC", Net: netPWR},
		schematic.Wire{From: "pico.GND7", To: "lcd.GND", Net: netGND},
		schematic.Wire{From: "pico.VBUS", To: "lcd.LED+", Net: netPWR},
		schematic.Wire{From: "pico.GND6", To: "lcd.LED-", Net: netGND},

		// DS1307 on I2C0.
		schematic.Wire{From: "pico.GP0", To: "rtc.SDA", Net: netI2C},
		schematic.Wire{From: "pico.GP1", To: "rtc.SCL", Net: netI2C},
		schematic.Wire{From: "pico.3V3", To: "rtc.VCC", Net: netPWR},
		schematic.Wire{From: "pico.GND1", To: "rtc.GND", Net: netGND},
	)

	wireSecondDisplay(s)

	return s
}

// wireSecondDisplay adds the second HD44780 from the multidisplays default in
// main.go: data on GP5..GP12, RS on GP4, EN on GP3, contrast on GP2.
func wireSecondDisplay(s *schematic.Schematic) {
	data2 := []string{"GP5", "GP6", "GP7", "GP8", "GP9", "GP10", "GP11", "GP12"}
	for i, p := range data2 {
		s.Wires = append(s.Wires, schematic.Wire{
			From: "pico." + p,
			To:   fmt.Sprintf("lcd2.BIT%d", i),
			Net:  netDAT,
		})
	}
	s.Wires = append(s.Wires,
		schematic.Wire{From: "pico.GP4", To: "lcd2.RS", Net: netCTL},
		schematic.Wire{From: "pico.GP3", To: "lcd2.EN", Net: netCTL},
		schematic.Wire{From: "pico.GP2", To: "lcd2.CONT", Net: netCTL},
		schematic.Wire{From: "pico.GND2", To: "lcd2.RW", Net: netGND},
		schematic.Wire{From: "pico.GND3", To: "lcd2.GND", Net: netGND},
		schematic.Wire{From: "pico.GND4", To: "lcd2.LED-", Net: netGND},
		schematic.Wire{From: "pico.VSYS", To: "lcd2.VCC", Net: netPWR},
		schematic.Wire{From: "pico.VSYS", To: "lcd2.LED+", Net: netPWR},
	)
}
