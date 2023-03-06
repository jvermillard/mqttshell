package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/creack/pty"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type terminalMsg struct {
	Type    string
	Payload string
}

func main() {

	// connect to the MQTT broker
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("mqttshell-server")
	opts.SetKeepAlive(25 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	mqttClient := mqtt.NewClient(opts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Impossible to connect")
		panic(token.Error())
	}

	log.Println("connected to the MQTT broker")

	// Start the command with a pty attached to bash.
	c := exec.Command("bash")
	ptmx, err := pty.Start(c)
	if err != nil {
		panic(err)
	}
	log.Println("pty connected")

	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.

	// need a protocol element to handle this:

	/*ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.
	*/

	// subscribe to input topic and push stream back to the pty

	if token := mqttClient.Subscribe("mqttshell/input", 0, func(client mqtt.Client, mqttMsg mqtt.Message) {
		if mqttMsg.Topic() != "mqttshell/input" {
			return
		}
		msg := terminalMsg{}
		err = json.Unmarshal(mqttMsg.Payload(), &msg)
		if err != nil {
			panic(err)
		}
		if msg.Type == "input" {
			data, err := hex.DecodeString(msg.Payload)
			if err != nil {
				panic(err)
			}
			// not sure it helps in place of directly writing
			_, err = io.Copy(ptmx, bytes.NewBuffer(data))
			if err != nil {
				panic(err)
			}
		}

	}); token.Wait() && token.Error() != nil {
		log.Println("error subscribing")
		panic(err)
	}

	for {
		buffer := make([]byte, 1024)
		n, err := ptmx.Read(buffer)
		if err != nil {
			panic(err)
		}
		hexPayload := hex.EncodeToString(buffer[:n])
		msg := terminalMsg{Type: "output", Payload: hexPayload}
		data, err := json.Marshal(&msg)
		if err != nil {
			panic(err)
		}
		mqttClient.Publish("mqttshell/output", 0, false, data)
	}
}
