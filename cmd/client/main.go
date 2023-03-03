package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/term"
)

type terminalMsg struct {
	Type    string
	Payload string
}

func main() {
	// connect to the MQTT broker
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("mqttshell-client")
	opts.SetKeepAlive(25 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	mqttClient := mqtt.NewClient(opts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Println("connected")

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	if token := mqttClient.Subscribe("mqttshell/output", 0, func(client mqtt.Client, mqttMsg mqtt.Message) {
		if mqttMsg.Topic() != "mqttshell/output" {
			log.Println("not the right message?")
			return
		}
		msg := terminalMsg{}
		err = json.Unmarshal(mqttMsg.Payload(), &msg)
		if err != nil {
			panic(err)
		}
		if msg.Type == "output" {
			data, err := hex.DecodeString(msg.Payload)
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(os.Stdout, bytes.NewBuffer(data))
			if err != nil {
				panic(err)
			}
		}
	}); token.Wait() && token.Error() != nil {
		panic(err)
	}

	for {
		buffer := make([]byte, 1024)
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			panic(err)
		}
		hexPayload := hex.EncodeToString(buffer[:n])
		msg := terminalMsg{Type: "input", Payload: hexPayload}
		data, err := json.Marshal(&msg)
		if err != nil {
			panic(err)
		}
		mqttClient.Publish("mqttshell/input", 0, false, data)
	}
}
