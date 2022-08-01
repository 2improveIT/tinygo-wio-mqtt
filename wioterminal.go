///go:build wioterminal
/// +build wioterminal

package main

import (
	"device/sam"
	"image/color"
	"machine"

	"runtime/interrupt"

	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/drivers/rtl8720dn"

	"time"
)

type KeyPad struct {
	Up    machine.Pin
	Down  machine.Pin
	Press machine.Pin
	C     machine.Pin
}

type DisplayBuffer struct {
	width  int16
	height int16
	buffer []uint16
}

func (buffer DisplayBuffer) Size() (x, y int16) {
	return buffer.width, buffer.height
}

func (buffer DisplayBuffer) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || x > buffer.width || y < 0 || y > buffer.height {
		return
	}
	pos := int32(x) + int32(y)*int32(buffer.width)
	buffer.buffer[pos] = ili9341.RGBATo565(c)
}

func (buffer DisplayBuffer) Display() error {
	return nil
}

func (buffer DisplayBuffer) Get() []uint16 {
	return buffer.buffer
}

func InitDisplay(c color.RGBA) (*ili9341.Device, *DisplayBuffer, *KeyPad) {
	machine.SPI3.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 48000000,
	})

	btnUp := machine.WIO_5S_UP
	btnUp.Configure(machine.PinConfig{Mode: machine.PinInput})

	btnDown := machine.WIO_5S_DOWN
	btnDown.Configure(machine.PinConfig{Mode: machine.PinInput})

	btnPress := machine.WIO_5S_PRESS
	btnPress.Configure(machine.PinConfig{Mode: machine.PinInput})

	btnC := machine.WIO_KEY_C
	btnC.Configure(machine.PinConfig{Mode: machine.PinInput})

	keyPad := KeyPad{
		Up:    btnUp,
		Down:  btnDown,
		Press: btnPress,
		C:     btnC,
	}

	backlight := machine.LCD_BACKLIGHT
	backlight.Configure(machine.PinConfig{machine.PinOutput})

	display := ili9341.NewSPI(
		machine.SPI3,
		machine.LCD_DC,
		machine.LCD_SS_PIN,
		machine.LCD_RESET,
	)

	display.Configure(ili9341.Config{})
	//display.SetRotation(ili9341.Rotation270)
	display.FillScreen(c)

	width, height := display.Size()
	size := int32(width) * int32(height)
	displayBuffer := DisplayBuffer{
		width:  width,
		height: height,
		buffer: make([]uint16, size, size),
	}

	backlight.High()

	machine.InitADC()
	//lightSensor := machine.ADC{Pin: machine.WIO_LIGHT}
	//lightSensor.Configure(machine.ADCConfig{})
	//
	//mic := machine.ADC{Pin: machine.WIO_MIC}
	//mic.Configure(machine.ADCConfig{})

	return display, &displayBuffer, &keyPad
}

var (
	uart UARTx
)

func handleInterrupt(interrupt.Interrupt) {
	// should reset IRQ
	uart.Receive(byte((uart.Bus.DATA.Get() & 0xFF)))
	uart.Bus.INTFLAG.SetBits(sam.SERCOM_USART_INT_INTFLAG_RXC)
}

func SetupRTL8720DN(debug bool) (*rtl8720dn.RTL8720DN, error) {
	machine.RTL8720D_CHIP_PU.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.RTL8720D_CHIP_PU.Low()
	time.Sleep(100 * time.Millisecond)
	machine.RTL8720D_CHIP_PU.High()
	time.Sleep(1000 * time.Millisecond)
	if debug {
		waitSerial()
	}

	uart = UARTx{
		UART: &machine.UART{
			Buffer: machine.NewRingBuffer(),
			Bus:    sam.SERCOM0_USART_INT,
			SERCOM: 0,
		},
	}

	uart.Interrupt = interrupt.New(sam.IRQ_SERCOM0_2, handleInterrupt)
	uart.Configure(machine.UARTConfig{TX: machine.PB24, RX: machine.PC24, BaudRate: 614400})

	rtl := rtl8720dn.New(uart)
	rtl.Debug(debug)

	_, err := rtl.Rpc_tcpip_adapter_init()
	if err != nil {
		return nil, err
	}

	return rtl, nil
}

// Wait for user to open serial console
func waitSerial() {
	for !machine.Serial.DTR() {
		time.Sleep(100 * time.Millisecond)
	}
}

type UARTx struct {
	*machine.UART
}

func (u UARTx) Read(p []byte) (n int, err error) {
	if u.Buffered() == 0 {
		time.Sleep(1 * time.Millisecond)
		return 0, nil
	}
	return u.UART.Read(p)
}
