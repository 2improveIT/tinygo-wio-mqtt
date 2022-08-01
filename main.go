package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"tinygo.org/x/drivers/net"
	"tinygo.org/x/drivers/net/mqtt"
	"tinygo.org/x/tinyfont/proggy"
	"tinygo.org/x/tinyterm"
)

var (
	black         = color.RGBA{0, 0, 0, 255}
	display, _, _ = InitDisplay(black)
	terminal      = tinyterm.NewTerminal(display)

	font = &proggy.TinySZ8pt7b
)

var (
	ssid     string = "<SSID>"
	password string = "<PASSWORD>"
	server   string = "tcp://<IP-ADDRESS>:1883"

	debug = false
)

var (
	topicPublish   = "heartbeat"
	topicSubscribe = "heartbeat"
)

func main() {
	display.FillScreen(black)
	terminal.Configure(&tinyterm.Config{
		Font:       font,
		FontHeight: 10,
		FontOffset: 6,
	})
	err := run()
	for err != nil {
		fmt.Fprintf(terminal, "error: %s\r\n", err.Error())
	}
}

func subHandler(client mqtt.Client, msg mqtt.Message) {
	fmt.Fprintf(terminal, "Received:\r\n")
	fmt.Fprintf(terminal, "[%s]  ", msg.Topic())
	fmt.Fprintf(terminal, "%s\r\n", msg.Payload())
}

func run() error {
	fmt.Fprintf(terminal, "Connecting to: %s...\r\n", ssid)
	rtl, err := SetupRTL8720DN(false)
	if err != nil {
		return err
	}
	net.UseDriver(rtl)
	err = rtl.ConnectToAP(ssid, password)
	if err != nil {
		return err
	}
	fmt.Fprintf(terminal, "Get IP ...\r\n")
	ip, subnet, gateway, err := rtl.GetIP()
	if err != nil {
		return err
	}
	fmt.Fprintf(terminal, "IP Address : %s\r\n", ip)
	fmt.Fprintf(terminal, "Mask       : %s\r\n", subnet)
	fmt.Fprintf(terminal, "Gateway    : %s\r\n", gateway)

	rand.Seed(time.Now().UnixNano())

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server).SetClientID("wio-client")

	fmt.Fprintf(terminal, "Connecting to MQTT: %s ...\r\n", server)
	cl := mqtt.NewClient(opts)
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		failMessage(token.Error().Error())
		time.Sleep(5 * time.Second)
	}

	token := cl.Subscribe(topicSubscribe, 0, subHandler)
	token.Wait()
	if token.Error() != nil {
		failMessage(token.Error().Error())
	}

	for {
		data := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		token := cl.Publish(topicPublish, 0, false, data)
		token.Wait()
		if err := token.Error(); err != nil {
			return err
		}
		time.Sleep(30 * time.Second)
	}
}

func failMessage(msg string) {
	for {
		fmt.Fprintf(terminal, "Error: %s\r\n", msg)
		time.Sleep(5 * time.Second)
	}
}
