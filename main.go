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
	"encoding/json"
	"fmt"
	"github.com/bgentry/speakeasy"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	graylog "github.com/gemnasium/logrus-graylog-hook"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strings"
	"time"
)

//import _ "github.com/motemen/go-loghttp/global"

var app = kingpin.New("wolfmqttbridge", "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation.")
var debug = app.Flag("debug", "Enable debug mode. Env: DEBUG").Envar("DEBUG").Short('d').Bool()
var grayLogAddr = app.Flag("graylogGELFAdr", "Address of GELF logging server as 'address:port'. Env: GRAYLOG").Envar("GRAYLOG").Short('g').String()
var wolf_user = app.Flag("user", "username at wolf-smartset.com. Env: WOLF_USER").Envar("WOLF_USER").String()
var wolf_pw = app.Flag("password", "Password for wolf-smartset.com. Env: WOLF_PW").Envar("WOLF_PW").String()

var listParamCmd = app.Command("list", "list parameters available in gateway")
var brCmd = app.Command("br", "start bridge").Default()
var mqttHost = brCmd.Flag("broker", "address of MQTT broker to connect to, e.g. tcp://mqtt.eclipse.org:1883. Env: BROKER").Envar("BROKER").String()
var mqttUsername = brCmd.Flag("mqttUser", "username for mqtt broker. Env: BROKER_USER").Envar("BROKER_USER").String()
var mqttPassword = brCmd.Flag("mqttPassword", "password for mqtt broker user. Env: BROKER_PW").Envar("BROKER_PW").String()
var haDiscoveryTopic = brCmd.Flag("haDiscovery", "Home Assistant MQTT discovery topic, defaults to 'homeassistant'").Default("homeassistant").String()

var mqttRootTopic = brCmd.Flag("topic", "root topic, defaults to /wolf").Envar("WOLF_MQTT_ROOT_TOPIC").Default("/wolf").String()

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0").Author("vax@kgbvax.net")
	kingpin.CommandLine.Help = "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation."
	kingpin.CommandLine.HelpFlag.Short('h')
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debug == true {
		log.SetLevel(log.DebugLevel)
	}

	if len(*grayLogAddr) > 0 {
		log.Info("Logging to Graylog: ", *grayLogAddr)
		hook := graylog.NewAsyncGraylogHook(*grayLogAddr, map[string]interface{}{})
		defer hook.Flush()
		log.AddHook(hook)
	}

	if wolf_pw == nil {
		*wolf_pw = askPw()
	}

	log.Debug("obtain auth token ", "user", *wolf_user)
	aTok, err := getAuthToken(*wolf_user, *wolf_pw)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	log.Debug("create session")
	sessId, err := createSession(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	go sessionRefresh(aTok.AccessToken, sessId)

	log.Debug("get system list")
	sysList, err := getSystemList(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	if len(sysList) < 1 {
		fmt.Println("System list is empty, nothing to do.")
		os.Exit(0)
	}

	//blindly pick the first system
	system := sysList[0]

	log.Info("System ID: ", system.ID)
	log.Info("System Name: ", system.Name)
	log.Info("Gateway ID: ", system.GatewayID)
	log.Info("Gateway Software Version: ", system.GatewaySoftwareVersion)

	doTheHustle(cmd, aTok, sessId, system)
}

// Ask for a user's password
func askPw() string {
	var err error
	pw, err := speakeasy.Ask("password: ")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	return pw
}

func doTheHustle(cmd string, token AuthToken, sessId int, system System) {
	log.Debug("main cmd: ", cmd)
	switch cmd {
	case listParamCmd.FullCommand():
		{
			guiDescription, _ := getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
			printGuiParameters(guiDescription)
		}

	case brCmd.FullCommand():
		{
			log.Debug("start bridge")
			lastUpdate := "2019-11-22" //some date in the past
			log.Debug("connecting to mqtt broker at ", *mqttHost)
			client := connectMQTT(*mqttHost, *mqttUsername, *mqttPassword)
			guiDescription, err := getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
			if err != nil {
				log.Error(err)
			}
			params := getPollParams(guiDescription)
			registerHADiscovery(params, client, *haDiscoveryTopic)

			for {
				var valIdList []int64
				for _, param := range params {
					valIdList = append(valIdList, param.ValueID)
				}

				paramterValuesResponse, _ := getParameterValues(token.AccessToken, sessId, valIdList, lastUpdate, system)
				lastUpdate = paramterValuesResponse.LastAccess
				for _, valueStruct := range paramterValuesResponse.Values {
					found := false
					for _, param := range params { //join with parameter meta
						if param.ValueID == valueStruct.ValueID {
							found = true
							value := valueStruct.Value
							if len(param.ListItems) > 0 { // transform according to list item
								for _, item := range param.ListItems {
									if item.Value == value {
										value = item.DisplayText
									}
								}
							}
							localTopic := makeTopic(param.Name)

							log.Debug("valueStruct response ", localTopic, "=", value)
							pub(client, localTopic, value)
						}
					}
					if found == false {
						log.Error("valueStruct not found in parameterDescription, valueId=", valueStruct.ValueID)
					}
				}
				time.Sleep(20 * time.Second)
			}
		}
	}

}

func makeTopic(paramName string) string {
	return *mqttRootTopic+"/"+sanitizeParamName(paramName)+"/state"
}

func sanitizeParamName(paramName string) string {
	return strings.Join(strings.Fields(paramName), "_")
}

type MqttDiscoveryMsg struct {
	Name                string `json:"name"`
	State_topic         string `json:"state_topic"`
	Unit_of_measurement string `json:"unit_of_measurement"`
	Unique_id           string `json:"unique_id"`
	Qos                 int    `json:"qos"`
	SwVersion			string `json:"sw_version"`
}

func registerHADiscovery(descriptors []ParameterDescriptor, client mqtt.Client, discoveryTopic string) {
	var wolfPrefix = "wolf-"
    discoPrefix:="homeassistant"

	for _, param := range descriptors {
		var newDisco = &MqttDiscoveryMsg{}
		newDisco.Name = param.Name
		if len(param.Unit) > 0 {
			newDisco.Unit_of_measurement = param.Unit
		}
		newDisco.Unique_id = wolfPrefix + param.Name
		newDisco.State_topic = makeTopic(param.Name)
		newDisco.Qos=2
		newDisco.SwVersion="1.0"
		configTopic:=discoPrefix+"/sensor/"+newDisco.Unique_id+"/config"
		json,err := json.Marshal(newDisco)
		if err != nil {
			log.Error(err)
		} else {
			pub(client,configTopic,string(json))
		}

	}

}
