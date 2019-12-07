package main

/* This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	log.Debug("TOPIC/MSG", msg.Topic(), "/", msg.Payload())
	//msg.Ack()
}

func getMacAddr() (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				// Don't use random as we have a real address
				addr = i.HardwareAddr.String()
				break
			}
		}
	}
	return
}

func connectMQTT(host string, username string, password string) MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(host)
	//when testing two clients may be running thus we grab a MAC address to create a semi-static machine specific clientID
	clientId := "wolfmqttbridge-" + getMacAddr()
	log.Info("Connecting as ", clientId)
	opts.SetClientID(clientId)
	opts.SetDefaultPublishHandler(f)
	opts.SetAutoReconnect(true)
	opts.SetPassword(password)
	opts.SetUsername(username)
	opts.SetKeepAlive(120 * time.Second)
	opts.SetPingTimeout(20 * time.Second)
	opts.SetConnectionLostHandler(onLost)
	opts.SetOrderMatters(false)
	opts.SetOnConnectHandler(onConnect)
	opts.SetMaxReconnectInterval(10 * time.Second)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("failed to connect to MQTT Broker, bailing out ", token.Error())
		os.Exit(-1)
	}

	return c
}

func onConnect(client MQTT.Client) {
	log.Info("MQTT client connected.")
}

func onLost(client MQTT.Client, err error) {
	log.Warn("MQTT connection lost: ", err)
}

func pub(cl MQTT.Client, topic string, payload string) error {
	log.Debug("MQTT: ", topic, " <- ", payload)
	if token := cl.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		log.Error("failed to publish message to ", topic, " error: ", token.Error())
		return token.Error()
	}
	return nil
}
