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

// set 'true' to enable onboard LED as a measure of rtc read time or program loop time indicator
var enableLED string //false

type LCDConfig struct {
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

// Configure multiple HD44780 displays
var multidisplays = [...]LCDConfig{{DataPins:    []m.Pin{m.GP22, m.GP21, m.GP20, m.GP19, m.GP18, m.GP17, m.GP16, m.GP15},RS: m.GP26,EN: m.GP27,RW:m.NoPin,Contrast:m.GP28,Clvl:2,Rows:2,Columns:16,CursorBlink:false,CursorOnOff:false,},{DataPins:[]m.Pin{m.GP5, m.GP6, m.GP7, m.GP8, m.GP9, m.GP10, m.GP11, m.GP12},RS:m.GP4,EN: m.GP3,RW:m.NoPin,Contrast:m.GP2,Clvl:3,Rows:2,Columns:16,CursorBlink: false,CursorOnOff: false,},}

//PWM implementation borrowed from: https://github.com/tinygo-org/tinygo/issues/2583
const numPWMPins = len(multidisplays)

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
	enableled bool
	rtcfuture  bool
	offset     int
	startTime time.Time
	parsedTime time.Time
	timeOffset time.Duration
	t          time.Time
	err        error
)

func l() string {
	return "[" + t.Format("2006-01-02 15:04:05") + "] "
}
func init() {
	startTime  = time.Now()
}

func main() {
	enableled, _ = strconv.ParseBool(enableLED)
	rtcfuture, _ = strconv.ParseBool(rtcFuture)
	offset, _ = strconv.Atoi(offSet)

	led := m.LED
	if enableled {
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
	m.I2C0.Configure(m.I2CConfig{SDA: m.GP0, SCL: m.GP1})
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

	configuredLCDs := make([]*hd44780.Device, len(multidisplays))

	for i, multidisplay := range multidisplays {
		pwmHandles[i].PWM, pwmHandles[i].channel, err = GetPWM(multidisplay.Contrast)
		if err != nil {
			m.Serial.Write([]byte(l() + err.Error() + "\n"))
		}
		err = pwmHandles[i].Configure(m.PWMConfig{
			Period: 50,
		})
		if err != nil {
			m.Serial.Write([]byte(l() + err.Error() + "\n"))
		}
		pwmHandles[i].Set(pwmHandles[i].channel, pwmHandles[i].Top() / uint32(multidisplay.Clvl))
		m.Serial.Write([]byte(l() + "Set Contrast\n"))

		lcd, err := hd44780.NewGPIO8Bit(multidisplay.DataPins, multidisplay.RS, multidisplay.EN, multidisplay.RW)
		if err != nil {
			m.Serial.Write([]byte(l() + "error initializing gpio\n"))
		}
		m.Serial.Write([]byte(l() + "initialized lcd\n"))

		err = lcd.Configure(hd44780.Config{
			Width:       multidisplay.Columns,
			Height:      multidisplay.Rows,
			CursorOnOff: multidisplay.CursorOnOff,
			CursorBlink: multidisplay.CursorBlink,
		})
		if err != nil {
			m.Serial.Write([]byte(l() + "error configuring lcd\n" + err.Error() + "\n"))
		}
		m.Serial.Write([]byte(l() + "configured lcd\n"))
		lcd.ClearDisplay()
		lcd.SetCursor(0, 0)
		configuredLCDs[i] = &lcd
		lcd.CreateCharacter(0x0, []byte{
			0b10101010,
			0b01010101,
			0b10101010,
			0b01010101,
			0b10101010,
			0b01010101,
			0b10101010,
			0b01010101,
		})
	}

	var u time.Time

	for {
		if enableled {
			led.High()
		}
		t, err = rtc.ReadTime()
		if enableled {
			led.Low()
		}
		if err != nil {
			m.Serial.Write([]byte(l() + err.Error() + "\n"))
			for _, lcd := range configuredLCDs {
				lcd.SetCursor(0, 0)
				lcd.Display()
				lcd.SetCursor(0, 1)
				lcd.Write([]byte(err.Error()))
				lcd.Display()
			}
			} else {
				//only update the display or print the time if the time is actually different
				if t.Format(time.RFC3339) != u.Format(time.RFC3339) {
					u = t
					m.Serial.Write([]byte(l() + "\n"))
					dt := strings.Split(strings.TrimRight(t.Format(time.RFC3339), "Z"), "T")
					dt[0] += strings.Repeat(" ", 16-len(dt[0]))
					dt[0] = dt[0][:len(dt[0])-3] + t.Weekday().String()[:3]
					dt[1] += strings.Repeat(" ", 16-len(dt[1]))
					for _, lcd := range configuredLCDs {
						lcd.SetCursor(0, 0)
						//2024-03-02	Sat
						lcd.Write([]byte(dt[0]))
						lcd.Display()
						lcd.SetCursor(0, 1)
						//19:33:39
						lcd.Write([]byte(dt[1]))
						lcd.Display()
						lcd.SetCursor(15, 1)
						lcd.Write([]byte{0x0})
						lcd.Display()
					}
				}
			}
		}
}
