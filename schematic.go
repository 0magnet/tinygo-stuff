package main

import (
	"bufio"
	"fmt"
	"github.com/bitfield/script"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"github.com/guptarohit/asciigraph"
	"strings"
	"sync"
	"regexp"
"strconv"
)

func main() {
	picoWidth := 0
	for _, j := range strings.Split(pico, "\n") {
		if len(j) > picoWidth {
			picoWidth = len(j)
		}
	}
	rtcWidth := 0
	for _, j := range strings.Split(rtc, "\n") {
		if len(j) > rtcWidth {
			rtcWidth = len(j)
		}
	}
	graphwidth := 20
	numwires := 24
	data := make([][]float64, numwires)
	for i := numwires - 1; i >= 0; i-- {
		switch {
		case i == 0:
			b := graphwidth
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-29)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(2)) }
		case i == 1:
			b := 12
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-26)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(1)) }
		case i == 2:
			b := 14
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-27)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(0)) }
		case i == 3:
			b := -16
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-12)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(1)) }
		case i == 4:
			b := -14
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-13)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(0)) }
		case i == 5:
			b := -18
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-5)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(2)) }
		case i == 6:
			b := -4
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-4)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-6)) }
		case i == 7:
			b := -2
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-3)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-5)) }
		case i == 8:
			b := -16
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-1)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(1)) }
		case i == 9:
			b := -14
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-2)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(0)) }
		case i == 10:
			b := -8
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-16)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-7)) }
		case i == 11:
			b := -12
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-14)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-8)) }
		case i == 12:
			b := -10
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-15)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-10)) }
		case i == 13:
			b := -6
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-17)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-9)) }
		case i == 14:
			b := -4
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-18)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-11)) }
		case i == 15:
			b := -2
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-19)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-13)) }
		case i == 16:
			b := 0
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-20)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-14)) }
		case i == 17:
			b := 2
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-21)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-15)) }
		case i == 18:
			b := 4
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-22)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-16)) }
		case i == 19:
			b := 6
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-23)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-18)) }
		case i == 20:
			b := 8
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-24)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-19)) }
		case i == 21:
			b := 10
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-25)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-20)) }
		case i == 22:
			b := 16
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-29)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-24)) }
		case i == 23:
			b := 16
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-29)) }
			for x := b; x <= graphwidth+picoWidth; x++ { data[i] = append(data[i], float64(-30)) }
		default:
			for x := -graphwidth; x <= i; x++ { data[i] = append(data[i], float64(-i-7)) }
			for x := i; x <= graphwidth; x++ { data[i] = append(data[i], float64(-i-7)) }
		}
	}
	rtcwires := asciigraph.PlotMany(data, asciigraph.Precision(0), asciigraph.SeriesColors(16, 1, 0, 0, 1, 2, 3, 4,  0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 3, 3 ))

	numtopwires := 3
	data = make([][]float64, numtopwires)
	for i := numtopwires - 1; i >= 0; i-- {
		switch {
		case i == 0:
			b := 18
			for x := -picoWidth; x <= b; x++ { data[i] = append(data[i], float64(0)) }
			for x := b; x <= picoWidth; x++ { data[i] = append(data[i], float64(-3)) }
		case i == 1:
			b := 16
			for x := -picoWidth; x <= b; x++ { data[i] = append(data[i], float64(-1)) }
			for x := b; x <= picoWidth; x++ { data[i] = append(data[i], float64(-3)) }
		case i == 2:
			b := 14
			for x := -picoWidth; x <= b; x++ { data[i] = append(data[i], float64(-2)) }
			for x := b; x <= picoWidth; x++ { data[i] = append(data[i], float64(-3)) }
		default:
			for x := -picoWidth; x <= i; x++ { data[i] = append(data[i], float64(-i-7)) }
			for x := i; x <= picoWidth; x++ { data[i] = append(data[i], float64(-i-7)) }
		}
	}
	topwires = asciigraph.PlotMany(data, asciigraph.Precision(0), asciigraph.SeriesColors(2, 0, 1, 1, 0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 1, 0 ))

	numwires = 20
	data = make([][]float64, numwires)
	for i := numwires - 1; i >= 0; i-- {
		switch {
		case i == 0:
			b := graphwidth
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-36)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-6)) }
		case i == 1:
			b := 15
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-13)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-32)) }
		case i == 2:
			b := 17
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-11)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-31)) }
		case i == 3:
			b := 10
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-11)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-18)) }
		case i == 4:
			b := -4
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-13)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-6)) }
		case i == 5:
			b := -2
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-14)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-6)) }
		case i == 6:
			b := 12
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-13)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-17)) }
		case i == 7:
			b := -6
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-11)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-6)) }
		case i == 8:
			b := 0
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-18)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-21)) }
		case i == 9:
			b := 2
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-17)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-19)) }
		case i == 10:
			b := -8
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-19)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-22)) }
		case i == 11:
			b := 0
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-20)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-20)) }
		case i == 12:
			b := -10
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-22)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-23)) }
		case i == 13:
			b := 0
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-24)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-24)) }
		case i == 14:
			b := 12
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-25)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-25)) }
		case i == 15:
			b := 13
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-26)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-26)) }
		case i == 16:
			b := 14
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-27)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-27)) }
		case i == 17:
			b := -7
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-29)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-28)) }
		case i == 18:
			b := -6
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-30)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-29)) }
		case i == 19:
			b := -5
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-35)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-30)) }
		case i == 20:
			b := 17
			for x := -graphwidth; x <= b; x++ { data[i] = append(data[i], float64(-34)) }
			for x := b; x <= graphwidth; x++ { data[i] = append(data[i], float64(-32)) }
		default:
			for x := -graphwidth; x <= i; x++ { data[i] = append(data[i], float64(-i-7)) }
			for x := i; x <= graphwidth; x++ { data[i] = append(data[i], float64(-i-7)) }
		}
	}
	lcdwires := asciigraph.PlotMany(data, asciigraph.Precision(0), asciigraph.SeriesColors(16, 0, 1, 1, 0, 2, 0, 1, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 2, 3, 0, 1, 0, 2 ))
	lw := strings.Split(lcdwires, "\n")
	lcdwires = ""
	for i := 1; i < len(lw); i++ {
		lcdwires += lw[i]+"\n"
	}



	terminalSize()
	_, _ = script.Exec(`bash -c 'echo -en "\033[18t"'`).Stdout()
	wg.Wait()
	restoreFunc()
	fmt.Println()
	for i := 0; i < tcols; i++ {
		fmt.Printf("%d", i % 10)
	}
	fmt.Println()

	l := strings.Split(lcdwires, "\n")
	for i, j := range l {
		if len(j) > 8 {
			l[i] = fmt.Sprintf("%-*s", 40, j[8:])
		}
	}
//	p := strings.Split(pico, "\n")

var trimmedtopwires string
tw := strings.Split(topwires, "\n")
for i := 0; i < numtopwires; i++ {
	trimmedtopwires += tw[i]+"\n"
}

p := strings.Split(trimmedtopwires+pico, "\n")
	for i, _ := range p {
		if i < 3 {
			p[i] = p[i][7:]
		}
		if i > 3 {
			break
		}
	}
alignOffset := len(strings.Split(rtc, "\n"))-2

	d := strings.Split(dcl, "\n")
	r := strings.Split(rtc+lcd, "\n")
	offset := 0
	rtw := strings.Split(rtcwires, "\n")

	rtw = append(rtw[:len(rtw) - 2], rtw[len(rtw) - 2+1:]...)
	for i, line := range rtw {
		if len(r) > i {
			fmt.Printf("%s", r[i]+line[8:])
		}
		/*
		// asciigraph does not produce even width output
		//it's impossible to correctly calculate the actual length of the text displayed in the terminal from the string because of ansi styling.
		//, the cursor position is fetched from the terminal and subsequent iterations will pad the string
		*/
		if ccol == 0 {
			cursorPos()
			_, _ = script.Exec(`bash -c 'echo -en "\e[6n"'`).Stdout()
			wg.Wait()
			restoreFunc()
			crow0, ccol0 = crow, ccol
			fmt.Println(p[i])
		} else {
			cursorPos()
			_, _ = script.Exec(`bash -c 'echo -en "\e[6n"'`).Stdout()
			wg.Wait()
			restoreFunc()
//			fmt.Printf("\x1b[%d;%dH", crow+i, ccol) //go to position
//			fmt.Printf("\x1b[%dG", ccol)
		if len(p) > i {
			p[i] = fmt.Sprintf("%*s", ccol0 - ccol,  "")+p[i]
			fmt.Printf(p[i])
		}
		if i -3 < len(l) && i-3 >= 0 {
			if len(l[i-3]) > 8 {
				fmt.Printf(l[i-3])
			}
		}
		if i -alignOffset < len(d) && i-alignOffset >= 0 {
			fmt.Printf(d[i-alignOffset])
		}
		fmt.Println()
		offset = i +1
	}
	}
	maxLines := len(r)
	if len(p) > maxLines {
		maxLines = len(p)
	}
	for i := offset; i < maxLines; i++ {
		if i < len(r) {
			fmt.Printf(r[i])
		}
		if i < len(p) {
			p[i] = fmt.Sprintf("%*s", ccol0+len(p[i])-len(r[i])-1, p[i])
//			fmt.Printf("\x1b[%dG", ccol0)
			fmt.Printf(p[i])
		}
		fmt.Println()
}


/*
numwires = 256
data = make([][]float64, numwires)
for i := numwires - 1; i >= 0; i-- {
		for x := -graphwidth; x <= graphwidth; x++ { data[i] = append(data[i], float64(-i)) }
}
fmt.Println(asciigraph.PlotMany(data, asciigraph.Precision(0), asciigraph.SeriesColors(0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95,96,97,98,99,100,101,102,103,104,105,106,107,108,109,110,111,112,113,114,115,116,117,118,119,120,121,122,123,124,125,126,127,128,129,130,131,132,133,134,135,136,137,138,139,140,141,142,143,144,145,146,147,148,149,150,151,152,153,154,155,156,157,158,159,160,161,162,163,164,165,166,167,168,169,170,171,172,173,174,175,176,177,178,179,180,181,182,183,184,185,186,187,188,189,190,191,192,193,194,195,196,197,198,199,200,201,202,203,204,205,206,207,208,209,210,211,212,213,214,215,216,217,218,219,220,221,222,223,224,225,226,227,228,229,230,231,232,233,234,235,236,237,238,239,240,241,242,243,244,245,246,247,248,249,250,251,252,253,254,255, )))
*/
}

var (
	wg sync.WaitGroup
	restoreFunc func()
	err error
	trows int
	tcols int
	crow int
	ccol int
	crow0 int
	ccol0 int
	topwires string
)

func seriesColorsWrapper(n int) []int {
    colors := make([]int, 0, n)
    for i := 0; i < n; i++ {
        colors = append(colors, i)
    }

    return colors
}

func cursorPos() {
	wg.Add(1)
	go func() {
		restoreFunc, err = rawMode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		defer restoreFunc()
		r := bufio.NewReader(os.Stdin)
		ansiSequence := ""
		for {
			b, err := r.ReadByte()
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "Error: reading key from Stdin: %s\n", err)
				os.Exit(1)
			}
			if b == 'q' {
				break
			}
			ansiSequence += string(b)
			if strings.HasSuffix(ansiSequence, "R") {
				break
			}
		}
		crow, ccol = parseCursorPosition(ansiSequence)
		wg.Done()
	}()
}

func terminalSize() {
	wg.Add(1)
	go func() {
		restoreFunc, err = rawMode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		defer restoreFunc()

		r := bufio.NewReader(os.Stdin)
		ansiSequence := ""
		for {
			b, err := r.ReadByte()
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "Error: reading key from Stdin: %s\n", err)
				os.Exit(1)
			}
			if b == 'q' {
				break
			}
			ansiSequence += string(b)
			if strings.HasSuffix(ansiSequence, "t") {
				break
			}
		}
		tcols, trows = parseTerminalSize(ansiSequence)
		wg.Done()
	}()
}

func rawMode() (func(), error) {

	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("rawMode: error getting terminal flags: %w", err)
	}

	copy := *termios

	termios.Lflag = termios.Lflag &^ (unix.ECHO | unix.ICANON)

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, termios); err != nil {
		return nil, fmt.Errorf("rawMode: error setting terminal flags: %w", err)
	}

	return func() {
		if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, &copy); err != nil {
			fmt.Fprintf(os.Stderr, "rawMode: error restoring original console settings: %s", err)
		}
	}, nil
}

func parseCursorPosition(ansiSequence string) (int, int) {
	// Extract row and column from the ANSI escape sequence
	parts := strings.Split(ansiSequence, ";")
	rowStr := strings.TrimPrefix(parts[0], "\x1b[")
	columnStr := strings.TrimSuffix(parts[1], "R")

	// Convert row and column to integers
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(columnStr)

	return row, col
}


func parseTerminalSize(ansiSequence string) (int, int) {
	// Define a regular expression to match the ANSI sequence
	re := regexp.MustCompile(`\[(\d+);(\d+);(\d+)t`)
	// FindSubmatch returns the matched strings and submatches.
	match := re.FindStringSubmatch(ansiSequence)
	if len(match) != 4 {
		return 0, 0
	}
	// Convert the matched rows and columns to integers
	rows, _ := strconv.Atoi(match[2])
	cols, _ := strconv.Atoi(match[3])
	return cols, rows
}


const pico =
`               +-----+              .
+--------------| USB |--------------+
|        GP25  +-----+              |
|1       [LED]                   40 |
| ( )GP0/U0Rx               VBUS( ) |
| ( )GP1/U0Tx               VSYS( ) |
| [ ]GND                     GND[ ] |
| ( )GP2                      x3( ) |
| ( )GP3                     3V3( ) |
| ( )GP4                    AREF( ) |
| ( )GP5                 A2/GP28( ) |
| [ ]GND                     GND[ ] |
| ( )GP6        +---+    A1/GP27( ) |
| ( )GP7        |   |    A0/GP26( ) |
| ( )GP8        |   |        RUN( ) |
| ( )GP9        +---+       GP22( ) |
| [ ]GND                     GND[ ] |
| ( )GP10                   GP21( ) |
| ( )GP11         \/        GP20( ) |
| ( )GP12        ()()       GP19( ) |
| ( )GP13        ()()       GP18( ) |
| [ ]GND          ()         GND[ ] |
| ( )GP14                   GP17( ) |
| ( )GP15        DEBUG      GP16( ) |
|20           [ ] [ ] [ ]         21|
|            MISO SCK RST           |
| Pi-Pico                           |
+-----------------------------------+`

const rtc =
`+----------+
|          |
|        1 |
|   GND[ ] |
|   VCC[ ] |
|   SDA[ ] |
|   SCL[ ] |
|   SQW[ ] |
|        5 |
| DS1307   |
+----------+`




const lcd = `
+----------+
|          |
|        1 |
|   GND[ ] |
|   VCC[ ] |
|  CONT[ ] |
|    RS[ ] |
|    RW[ ] |
|    EN[ ] |
|  BIT0[ ] |
|  BIT1[ ] |
|  BIT2[ ] |
|  BIT3[ ] |
|  BIT4[ ] |
|  BIT5[ ] |
|  BIT6[ ] |
|  BIT7[ ] |
|  LED+[ ] |
|  LED-[ ] |
|        16|
| HD44780  |
+----------+`
const dcl = `
+----------+
|          |
|  1       |
| [ ]GND   |
| [ ]VCC   |
| [ ]CONT  |
| [ ]RS    |
| [ ]RW    |
| [ ]EN    |
| [ ]BIT0  |
| [ ]BIT1  |
| [ ]BIT2  |
| [ ]BIT3  |
| [ ]BIT4  |
| [ ]BIT5  |
| [ ]BIT6  |
| [ ]BIT7  |
| [ ]LED+  |
| [ ]LED-  |
| 16       |
| HD44780  |
+----------+ `
