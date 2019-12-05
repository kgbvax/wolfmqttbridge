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
	log "github.com/sirupsen/logrus"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	log.Debug("TOPIC: ", msg.Topic())
	log.Debug("MSG: ", msg.Payload())
}

func connectMQTT(host string,username string, password string) MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(host)
	opts.SetClientID("wolfmqttbridge")
	opts.SetDefaultPublishHandler(f)
	opts.SetAutoReconnect(true)
	opts.SetPassword(password)
	opts.SetUsername(username)
	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
		os.Exit(-1)
	}
	return c
}

func pub(cl MQTT.Client, topic string, payload string) {
	log.Debug("topic: ",topic, " :: ",payload)
	if token := cl.Publish(topic,1,false,payload) ; token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
}
