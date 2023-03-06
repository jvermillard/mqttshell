package main

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"nhooyr.io/websocket"
)

//go:embed index.html
var indexPage string

type terminalMsg struct {
	Type    string
	Payload string
}

func main() {
	// serve the static page
	http.HandleFunc("/index.html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexPage))
	})

	// connect to the MQTT broker
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("mqttshell-client")
	opts.SetKeepAlive(25 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	mqttClient := mqtt.NewClient(opts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Println("connected")

	var cnx *websocket.Conn = nil
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		var err error
		cnx, err = websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			panic(err)
		}
		// publish read input
		for {
			_, data, err := cnx.Read(context.TODO())
			if err != nil {
				panic(err)
			}
			hexPayload := hex.EncodeToString(data)

			msg := terminalMsg{Type: "input", Payload: hexPayload}
			mqttData, err := json.Marshal(&msg)
			if err != nil {
				panic(err)
			}
			mqttClient.Publish("mqttshell/input", 0, false, mqttData)

		}
	})

	if token := mqttClient.Subscribe("mqttshell/output", 0, func(client mqtt.Client, mqttMsg mqtt.Message) {
		if mqttMsg.Topic() != "mqttshell/output" {
			log.Println("not the right message?")
			return
		}
		msg := terminalMsg{}
		err := json.Unmarshal(mqttMsg.Payload(), &msg)
		if err != nil {
			panic(err)
		}
		if msg.Type == "output" {
			data, err := hex.DecodeString(msg.Payload)
			if err != nil {
				panic(err)
			}

			if cnx != nil {
				err = cnx.Write(context.TODO(), websocket.MessageBinary, data)
				if err != nil {
					panic(err)
				}
			} else {
				log.Println("no websocket connection")
			}
		}
	}); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Fatal(http.ListenAndServe(":8080", nil))
}
