// Package main hd44780 lcd with rtc clock
package main

import (
	m "machine"
	"strconv"
	"strings"
	"time"

	"tinygo.org/x/drivers/ds1307"
	"tinygo.org/x/drivers/hd44780"
)

//timeStamp for rtc reference set on compilation RFC3339
//rtcFuture explicitly sets RTC time to the timestamp ; follow up by flashing with a complation where rtcFuture is not true to avoid setting the original timestamp every time
// By the time the program compiles, is flashed, and boots, a small amount of time has passed since the timestamp. offSet specifies the amount of time to add to the timestamp, in seconds, to account for this difference
/*
Workflow / flashing procedure ; note that the block device will be different on different systems
set -x ; sudo echo "sudo cache" ; udisksctl mount -b /dev/sdd1 ; tinygo flash -target=pico -ldflags "-X main.timeStamp='$(date '+%Y-%m-%dT%H:%M:%SZ')' -X main.multiDisplay='true' -X main.rtcFuture='true' -X main.offSet='9'" main.go && sleep 3 && sudo chmod a+rw $(echo /dev/ttyACM?) && go run mon.go  -m $(echo /dev/ttyACM?) ; set +x
*/

// Set timestamp $(date '+%Y-%m-%dT%H:%M:%SZ')
var timeStamp string //$(date '+%Y-%m-%dT%H:%M:%SZ')

// seconds to add to timeStamp
var offSet string //9

// set 'true' if RTC is set to the future
var rtcFuture string //false

// set 'true' to enable onboard LED as a measure of rtc read time and program loop rate
var blinkLED string //true


//X is the starting index counter of the text for scrolling text effect when Mode != "clock" { max < len(Mode) }
type HD44780 struct {
	Mode string    `json:"mode"`
	Speed *time.Ticker    `json:"speed"`
	Color string    `json:"color"`
	X int    `json:"index"`
	DataPins    []m.Pin `json:"data_pins"`
	RS          m.Pin   `json:"rs"`
	EN          m.Pin   `json:"en"`
	RW          m.Pin   `json:"rw"`
	Contrast    m.Pin   `json:"contrast"`
	Clvl    uint8   `json:"contrast_percent"`
	Rows        int16   `json:"rows"`
	Columns     int16   `json:"columns"`
	CursorBlink bool    `json:"cursor_blink"`
	CursorOnOff bool    `json:"cursor_on_off"`
}

type DS1307 struct {
	SDA          m.Pin   `json:"sda"`
	SCL          m.Pin   `json:"scl"`
}

// Configure multiple HD44780 displays
var displays = [...]HD44780{{Mode: "clock",Color: hiwht+hiblubg,Speed: time.NewTicker(333 * time.Millisecond), DataPins:    []m.Pin{m.GP22, m.GP21, m.GP20, m.GP19, m.GP18, m.GP17, m.GP16, m.GP15},RS: m.GP26,EN: m.GP27,RW:m.NoPin,Contrast:m.GP28,Clvl:2,Rows:2,Columns:16,CursorBlink:false,CursorOnOff:false,},{Mode: "Hello, World! This is a scrolling text example. ",Color: blk+grnbg,Speed: time.NewTicker(333 * time.Millisecond), DataPins:[]m.Pin{m.GP5, m.GP6, m.GP7, m.GP8, m.GP9, m.GP10, m.GP11, m.GP12},RS:m.GP4,EN: m.GP3,RW:m.NoPin,Contrast:m.GP2,Clvl:3,Rows:2,Columns:16,CursorBlink: false,CursorOnOff: false,},}

// Configure DS1307 rtc i2c
var ds1307i2c = DS1307{SDA: m.GP0, SCL: m.GP1}

var console [len(displays)+2]string

//PWM implementation borrowed from: https://github.com/tinygo-org/tinygo/issues/2583
const numPWMPins = len(displays)

var pwmHandles [numPWMPins]pwmHandle

type pwmHandle struct {
	PWM
	channel uint8
}

func (ph *pwmHandle) SetPercent(percent uint8) {
	top := ph.PWM.Top()
	ph.PWM.Set(ph.channel, uint32(percent)*top/100)
}

type PWM interface {
	Top() uint32
	Set(ch uint8, value uint32)
	Channel(pin m.Pin) (uint8, error)
	Configure(m.PWMConfig) error
}

func GetPWM(pin m.Pin) (pwm PWM, channel uint8, err error) {
	slice, err := m.PWMPeripheral(pin)
	if err != nil {
		return pwm, channel, err
	}
	pwm = pwmFromSlice(slice)
	channel, err = pwm.Channel(pin)
	if err != nil {
		return pwm, channel, err
	}
	return pwm, channel, nil
}

func pwmFromSlice(i uint8) PWM {
	if i > 7 {
		panic("PWM out of range")
	}
	pwms := [...]PWM{
		m.PWM0, m.PWM1, m.PWM2,
		m.PWM3, m.PWM4, m.PWM5,
		m.PWM6, m.PWM7,
	}
	return pwms[i]
}

var (
	blinkled bool
	rtcfuture  bool
	offset     int
	startTime time.Time
	parsedTime time.Time
	timeOffset time.Duration
	times, semit []time.Time
	t, u time.Time
	err        error
	errs        []error
)

func l() string {
return " [" + t.Format("2006-01-02 15:04:05") + "]\n"
}
func init() {
	startTime  = time.Now()
}

func main() {
	blinkled, _ = strconv.ParseBool(blinkLED)
	rtcfuture, _ = strconv.ParseBool(rtcFuture)
	offset, _ = strconv.Atoi(offSet)

	led := m.LED
	if blinkled {
		led.Configure(m.PinConfig{Mode: m.PinOutput})
	}

	if timeStamp != "" {
		parsedTime, _ = time.Parse(time.RFC3339, timeStamp)
		parsedTime = parsedTime.Add(time.Duration(offset) * time.Second)
		timeOffset = parsedTime.Sub(startTime)
	}

	//add 1 second delay to compensate for switch noise or early interrupt causing rw errors for RTC
	time.Sleep(time.Second)


	//configure RTC
		m.I2C0.Configure(m.I2CConfig{SDA: ds1307i2c.SDA, SCL: ds1307i2c.SCL})
		rtc := ds1307.New(m.I2C0)
		t, err = rtc.ReadTime()
		if err == nil {
		//if RTC is set in the future, explicitly set parsed time
		if rtcfuture {
			rtc.SetTime(parsedTime)
			} else {
				if parsedTime.After(t) {
					rtc.SetTime(parsedTime)
				}
			}
			//When RTC time is accurate, reflash with rtcFuture=false or not specified to avoid setting the time to the original parsed time every run
		}


	m.Serial.Configure(m.UARTConfig{})
	m.Serial.Write([]byte(l() + "Configured serial interface\n"))
	m.Serial.Write([]byte(l() + "starting up\n"))

	lcds := make([]*hd44780.Device, len(displays))

	for i, display := range displays {
		defer display.Speed.Stop()
		if display.Mode != "clock" {
			displays[i].Mode = strings.Repeat(" ", int(display.Columns) * int(display.Rows)) + display.Mode
		}

		pwmHandles[i].PWM, pwmHandles[i].channel, err = GetPWM(display.Contrast)
		if err != nil {
			m.Serial.Write([]byte(l() + err.Error() + "\n"))
		}
		err = pwmHandles[i].Configure(m.PWMConfig{
			Period: 50,
		})
		if err != nil {
			m.Serial.Write([]byte(l() + err.Error() + "\n"))
		}
		pwmHandles[i].Set(pwmHandles[i].channel, pwmHandles[i].Top() / uint32(display.Clvl))
		m.Serial.Write([]byte(l() + "Set Contrast\n"))

		var lcd hd44780.Device
		if len(display.DataPins) == 8 ||  len(display.DataPins) == 4 {
			if len(display.DataPins) == 8 {
				lcd, err = hd44780.NewGPIO8Bit(display.DataPins, display.RS, display.EN, display.RW)
				if err != nil {
					m.Serial.Write([]byte(l() + "error initializing 8 bit gpio\n"))
				}
			}
			if len(display.DataPins) == 4 {
				lcd, err = hd44780.NewGPIO4Bit(display.DataPins, display.RS, display.EN, display.RW)
				if err != nil {
					m.Serial.Write([]byte(l() + "error initializing 4 bit gpio\n"))
				}
			}
		} else {
			m.Serial.Write([]byte(l() + "Wrong number of pins for gpio!\n"))
		}

		m.Serial.Write([]byte(l() + "initialized lcd\n"))

		err = lcd.Configure(hd44780.Config{
			Width:       display.Columns,
			Height:      display.Rows,
			CursorOnOff: display.CursorOnOff,
			CursorBlink: display.CursorBlink,
		})
		if err != nil {
			m.Serial.Write([]byte(l() + "error configuring lcd\n" + err.Error() + "\n"))
		}
		m.Serial.Write([]byte(l() + "configured lcd\n"))
		lcd.ClearDisplay()
		lcd.SetCursor(0, 0)
		lcds[i] = &lcd
	}
	console[len(console)-1] = "\x1b[0m\n\n"
	console[0] = "\033c\033[?25h" + l()
	for {
		if console[0] != "\033c\033[?25h" + l() {			console[0] = "\033c\033[?25h" + l()		}
		if blinkled {led.Low()}
		t, err = rtc.ReadTime()
		if blinkled {led.High()}
		if err != nil {
			for i, lcd := range lcds {
				disp(i, lcd)
			}
			continue
		}
		for i, lcd := range lcds {
			if displays[i].Speed != nil {
				ticker := displays[i].Speed
				select {
				case <-ticker.C:
					disp(i, lcd)
				default:
				}
			} else {
				disp(i, lcd)
			}
		}
	}
}

func disp (i int, lcd *hd44780.Device) {
	if err != nil {
		lcd.SetCursor(0, 0)
		lcd.Write([]byte(err.Error()))
		lcd.Display()
		m.Serial.Write([]byte(l() + err.Error() + "\n"))
		return
	}
	var c string
	c += "LCD "+strconv.Itoa(i) + ":"
	x := displays[i].X
	color := displays[i].Color
	col := displays[i].Columns
	row := displays[i].Rows
	c += cn+color
	if displays[i].Mode == "clock" {
		dt := strings.Split(strings.TrimRight(t.Format(time.RFC3339), "Z"), "T")
		dt[0] += strings.Repeat(" ", int(col)-len(dt[0]))
		dt[0] = dt[0][:len(dt[0])-3] + t.Weekday().String()[:3]
		dt[1] += strings.Repeat(" ", int(col)-len(dt[1]))
		c += dt[1]
		if row > 1 {
			c += cn
			c += dt[0]
		}
		c += rst + "\n"
		lcd.SetCursor(0, 0)
		//19:33:39
		lcd.Write([]byte(dt[1]))
		lcd.Display()
		if row > 1 {
			lcd.SetCursor(0, 1)
			//2024-03-02	Sat
			lcd.Write([]byte(dt[0]))
			lcd.Display()
		}
	}
	if displays[i].Mode != "clock" {
		for j := 0; j < int(row); j++ {
			txtStart := (x + j*int(col)) % len(displays[i].Mode)
			txtEnd := txtStart + int(col)
			if txtEnd > len(displays[i].Mode) {
				txtEnd = len(displays[i].Mode)
			}
			c += displays[i].Mode[txtStart:txtEnd] + strings.Repeat(" ", int(col)-(txtEnd-txtStart)) + cn
		}
		c += rst + "\n"

		lcd.SetCursor(0, 0)
		lcd.Write([]byte(displays[i].Mode[x:len(displays[i].Mode)]))
		lcd.Display()
		displays[i].X++
		if displays[i].X >= len(displays[i].Mode) {
			displays[i].X = 0
		}
	}
	if console[i+1] != c {
		console[i+1] = c
		for _, c := range console {
			m.Serial.Write([]byte(c))
		}
	}
}


const (
    // cursor
    cu       = "\033[A"
    cd     = "\033[B"
    cf  = "\033[C"
    cb     = "\033[D"
    cn = "\033[E"
    cp = "\033[F"
    ch     = "\033[?25l"
    cs     = "\033[?25h"
    // clearing
    clear    = "\033[2J"
    cln      = "\033[2K"
    clnr = "\033[K"
    clnl  = "\033[1K"
    // scrolling
    su   = "\033[S"
    sd = "\033[T"
    // cursor position
    scp    = "\033[s"
    rcp = "\033[u"
    // text styles
    dim       = "\033[2m"
    italic    = "\033[3m"
    blink     = "\033[5m"
    blinkfast = "\033[6m"
    reverse   = "\033[7m"
    hidden    = "\033[8m"
	rst	=	"\033[0m"
	//text colors
	blk = "\033[30m" // Black
    red = "\033[31m" // Red
    grn = "\033[32m" // Green
    ylw = "\033[33m" // Yellow
    blu = "\033[34m" // Blue
    mag = "\033[35m" // Magenta
    cyn = "\033[36m" // Cyan
    wht = "\033[37m" // White
	hiblk = "\033[90m"
    hired = "\033[91m"
    higrn = "\033[92m"
    hiyel = "\033[93m"
    hiblu = "\033[94m"
    himag = "\033[95m"
    hicyn = "\033[96m"
    hiwht = "\033[97m"
	// background colors
	blkbg = "\033[40m"  // Black background
	redbg = "\033[41m"  // Red background
	grnbg = "\033[42m"  // Green background
	yelbg = "\033[43m"  // Yellow background
	blubg = "\033[44m"  // Blue background
	purbg = "\033[45m"  // Purple background
	cynbg = "\033[46m"  // Cyan background
	whtbg = "\033[47m"  // White background
    hiblkbg = "\033[100m"
    hiredbg = "\033[101m"
    higrnbg = "\033[102m"
    hiyelbg = "\033[103m"
    hiblubg = "\033[104m"
    hipurbg = "\033[105m"
    hicynbg = "\033[106m"
    hiwhtbg = "\033[107m"
)
