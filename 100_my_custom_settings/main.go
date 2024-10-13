package main

import (
	"fmt"
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
	"tinygo.org/x/tinyfont/gophers"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
)

var (
	white = color.RGBA{R: 0x20, G: 0x20, B: 0x20, A: 0x00}
	black = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}
)

// --------------
// WS2812B（キーボードのLED）用の設定
// --------------
type WS2812B struct {
	Pin       machine.Pin
	ws        *piolib.WS2812B
	ledColors []color.RGBA
}

func NewWS2812B(pin machine.Pin, numLEDs int) *WS2812B {
	s, _ := pio.PIO0.ClaimStateMachine()
	ws, _ := piolib.NewWS2812B(s, pin)
	ws.EnableDMA(true)
	return &WS2812B{
		ws:        ws,
		ledColors: make([]color.RGBA, numLEDs),
	}
}

func (ws *WS2812B) SetLED(index int, c color.Color) {
	if index < 0 || index >= len(ws.ledColors) {
		return
	}
	r, g, b, _ := c.RGBA()
	ws.ledColors[index] = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xFF}
}

func (ws *WS2812B) UpdateLEDs() {
	for _, c := range ws.ledColors {
		ws.ws.PutColor(c)
	}
}

func main() {

	// --------------
	// OLED（液晶）用の設定
	// --------------
	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 2.8 * machine.MHz,
		SDA:       machine.GPIO12,
		SCL:       machine.GPIO13,
	})

	display := ssd1306.NewI2C(machine.I2C0)
	display.Configure(ssd1306.Config{
		Address:  0x3C,
		Width:    128,
		Height:   64,
		Rotation: drivers.Rotation180,
	})
	display.ClearDisplay()
	time.Sleep(50 * time.Millisecond)

	// --------------
	// キーボード用の設定
	// --------------
	colPins := []machine.Pin{
		machine.GPIO5,
		machine.GPIO6,
		machine.GPIO7,
		machine.GPIO8,
	}

	rowPins := []machine.Pin{
		machine.GPIO9,
		machine.GPIO10,
		machine.GPIO11,
	}

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	ws := NewWS2812B(machine.GPIO1, len(colPins)*len(rowPins))

	data := []byte("ABCEF")
	for {
		showTextOnOLED(display, data)

		scanKeyboardAndControlLED(colPins, rowPins, ws)
		ws.UpdateLEDs()

		time.Sleep(100 * time.Millisecond)
	}
}

func showTextOnOLED(display ssd1306.Device, data []byte) {
	display.ClearBuffer()
	// 液晶の1行目にはアニメーションせずにテキストを表示する
	tinyfont.WriteLine(&display, &freemono.Bold9pt7b, 5, 10, "Hi,I'm Momo", white)
	// 2行目はアニメーションさせ、tiny fontでGopherを表示する
	data[0], data[1], data[2], data[3], data[4] = data[1], data[2], data[3], data[4], data[0]
	tinyfont.WriteLine(&display, &gophers.Regular32pt, 5, 45, string(data), white)
	display.Display()
}

func scanKeyboardAndControlLED(colPins []machine.Pin, rowPins []machine.Pin, ws *WS2812B) {
	for i := range ws.ledColors {
		ws.SetLED(i, black)
	}

	for colIndex, colPin := range colPins {
		colPin.High()
		time.Sleep(1 * time.Millisecond)

		for rowIndex, rowPin := range rowPins {
			if rowPin.Get() {
				keyNumber := colIndex*len(rowPins) + rowIndex
				fmt.Printf("sw%d pressed, keyNumber = %d\n", keyNumber+1, keyNumber)

				if keyNumber >= 0 && keyNumber < len(ws.ledColors) {
					ws.SetLED(keyNumber, white)
					fmt.Printf("Set LED %d to white\n", keyNumber)
				} else {
					fmt.Printf("Error: keyNumber %d is out of range\n", keyNumber)
				}
			}
		}
		ws.UpdateLEDs()
		colPin.Low()
	}
}
