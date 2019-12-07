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
	MQTT "github.com/eclipse/paho.mqtt.golang"
	graylog "github.com/gemnasium/logrus-graylog-hook"
	"github.com/matryer/runner"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strings"
	"time"
)

//import _ "github.com/motemen/go-loghttp/global"

var app = kingpin.New("wolfmqttbridge", "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation.")
var debug = app.Flag("debug", "Enable debug mode. Env: DEBUG").Envar("DEBUG").Short('d').Bool()
var trace = app.Flag("trace", "Enable trace mode. Env: TRACE").Envar("TRACE").Bool()
var grayLogAddr = app.Flag("graylogGELFAdr", "Address of GELF logging server as 'address:port'. Env: GRAYLOG").Envar("GRAYLOG").Short('g').String()
var wolfUser = app.Flag("user", "username at wolf-smartset.com. Env: WOLF_USER").Envar("WOLF_USER").String()
var wolfPw = app.Flag("password", "Password for wolf-smartset.com. Env: WOLF_PW").Envar("WOLF_PW").String()

var listParamCmd = app.Command("list", "list parameters available in gateway")
var brCmd = app.Command("br", "start bridge").Default()
var mqttHost = brCmd.Flag("broker", "address of MQTT broker to connect to, e.g. tcp://mqtt.eclipse.org:1883. Env: BROKER").Envar("BROKER").String()
var mqttUsername = brCmd.Flag("mqttUser", "username for mqtt broker. Env: BROKER_USER").Envar("BROKER_USER").String()
var mqttPassword = brCmd.Flag("mqttPassword", "password for mqtt broker user. Env: BROKER_PW").Envar("BROKER_PW").String()
var haDiscoveryTopic = brCmd.Flag("haDiscoTopic", "Home Assistant MQTT discovery topic, defaults to 'homeassistant'").Envar("HA_DISCO_TOPIC").Default("homeassistant").String()
var brReadOnly = brCmd.Flag("ro", "Read-Only mode - don't write to MQTT (for testing").Default("false").Bool()
var mqttRootTopic = brCmd.Flag("rooTopic", "root topic, defaults to /wolf").Envar("WOLF_MQTT_ROOT_TOPIC").Default("wolf").String()
var pollInterval = brCmd.Flag("pollEvery", "poll every X seconds. Must be >10, defaults to 20").Default("20").Envar("POLL_EVERY").Int()

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0").Author("vax@kgbvax.net")
	kingpin.CommandLine.Help = "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation."
	kingpin.CommandLine.HelpFlag.Short('h')
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debug == true {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	}
	if *trace == true {
		log.SetLevel(log.TraceLevel)
		log.SetReportCaller(true)
	}

	if *pollInterval < 10 {
		log.Warn("poll interval is shorter than 10sec. Setting to 10sec to prevent excessive API load")
		*pollInterval = 10
	}

	if len(*grayLogAddr) > 0 {
		log.Info("Logging to Graylog: ", *grayLogAddr)
		hook := graylog.NewAsyncGraylogHook(*grayLogAddr, map[string]interface{}{})
		defer hook.Flush()
		log.AddHook(hook)
	}

	if wolfPw == nil {
		*wolfPw = askPw()
	}

	doTheHustle(cmd)
}

func connectWolfSmartset() (AuthToken, int, System, *runner.Task) {
	log.Debug("obtain auth token ", "user", *wolfUser)
	aTok, err := getAuthToken(*wolfUser, *wolfPw)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(ErrWolfToken) //&bail out
	}

	log.Debug("create session")
	sessId, err := createSession(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(ErrSession) //&bail out
	}

	task := runner.Go(func(shouldStop runner.S) error {
		// do setup work
		defer func() {
			// do tear-down work
		}()
		for {
			time.Sleep(60 * time.Second)

			sessionRefresh(aTok.AccessToken, sessId)

			if shouldStop() {
				break
			}
		}
		return nil // no errors
	})

	log.Debug("get system list")
	sysList, err := getSystemList(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	if len(sysList) < 1 {
		fmt.Println("System list is empty, nothing to do.")
		os.Exit(ErrSysListEmpty)
	}

	//blindly pick the first system
	system := sysList[0]

	log.Info("System ID: ", system.ID)
	log.Info("System Name: ", system.Name)
	log.Info("Gateway ID: ", system.GatewayID)
	log.Info("Gateway Software Version: ", system.GatewaySoftwareVersion)
	return aTok, sessId, system, task
}

// Ask for a user's password
func askPw() string {
	var err error
	pw, err := speakeasy.Ask("password: ")
	if err != nil {
		log.Println(err)
		os.Exit(-2)
	}
	return pw
}

func doTheHustle(cmd string) {
	var token AuthToken

	log.Debug("main cmd: ", cmd)
	switch cmd {
	case listParamCmd.FullCommand():
		{
			_, _, system, task := connectWolfSmartset()
			guiDescription, _ := getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
			printGuiParameters(guiDescription)
			task.Stop()
		}

	case brCmd.FullCommand():
		{
			var needsConnection bool = true

			var sessId int
			var system System
			var backgroundRefreshTask *runner.Task

			log.Debug("start bridge")
			lastUpdate := "2019-12-06T18:11:40.3881067Z"
			var client MQTT.Client
			if *brReadOnly == true {
				log.Info("Read-only mode, skip MQTT init")
			} else {
				log.Debug("connecting to mqtt broker at ", *mqttHost)
				client = connectMQTT(*mqttHost, *mqttUsername, *mqttPassword)
			}
			var valIdList []int64
			var params []ParameterDescriptor
			var guiDescription GuiDescription
			var err error

			for {
				if needsConnection {
					if backgroundRefreshTask != nil {
						backgroundRefreshTask.Stop()
					}

					token, sessId, system, backgroundRefreshTask = connectWolfSmartset()
					guiDescription, err = getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
					printGuiParameters(guiDescription)
					params = getPollParams(guiDescription)
					if err != nil {
						log.Error(err)
						os.Exit(ErrGuiDescription)
					}
					if !*brReadOnly {
						registerHADiscovery(params, client, *haDiscoveryTopic)
					}
					needsConnection = false

					for _, param := range params {
						valIdList = append(valIdList, param.ValueID)
					}
				}

				parameterValuesResponse, err := getParameterValues(token.AccessToken, sessId, valIdList, lastUpdate, system)
				if err != nil {
					log.Warn("failed to obtain parameters. attempting reconnect. Error= ", err)
					needsConnection = true
				} else {
					lastUpdate = parameterValuesResponse.LastAccess
					for _, valueStruct := range parameterValuesResponse.Values {
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

								//log.Debug("valueStruct response ", localTopic, "=", value)
								if !*brReadOnly {
									err = pub(client, localTopic, value)
									if err != nil {
										//log and ignore
										log.Error("faile to publish to ", localTopic, " error ", err)
									}
								}
							}
						}
						if found == false {
							log.Error("valueStruct not found in parameterDescription, valueId=", valueStruct.ValueID)
						}
					}
				}
				log.Trace("sleeping ", *pollInterval)
				time.Sleep(time.Duration(*pollInterval) * time.Second)
			}
		}
	}

}

func makeTopic(paramName string) string {
	return *mqttRootTopic + "/" + sanitizeParamName(paramName) + "/state"
}

func sanitizeParamName(paramName string) string {
	return strings.Join(strings.Fields(paramName), "_")
}

type MqttDiscoveryMsg struct {
	Name              string `json:"name"`
	StateTopic        string `json:"state_topic"`
	UnitOfMeasurement string `json:"unit_of_measurement"`
	UniqueId          string `json:"unique_id"`
	ExpireAfter       int    `json:"expire_after"`
	Qos               int    `json:"qos"`
	//SwVersion	    string `json:"sw_version"`
}

func registerHADiscovery(descriptors []ParameterDescriptor, client MQTT.Client, discoveryTopic string) {
	var wolfPrefix = "wolf-"
	discoPrefix := "homeassistant"

	for _, param := range descriptors {
		var newDisco = &MqttDiscoveryMsg{}
		newDisco.Name = param.Name
		if len(param.Unit) > 0 {
			newDisco.UnitOfMeasurement = param.Unit
		}
		newDisco.UniqueId = wolfPrefix + param.Name
		newDisco.StateTopic = makeTopic(param.Name)
		newDisco.Qos = 2
		//newDisco.SwVersion="1.0"
		newDisco.ExpireAfter = 120 //seconds
		configTopic := discoPrefix + "/sensor/" + newDisco.UniqueId + "/config"
		discoJson, err := json.Marshal(newDisco)
		if err != nil {
			//internal errer thus fatal
			log.Fatal("failed to marshal config payload ", newDisco, err)
			os.Exit(-1)
		} else {
			if !*brReadOnly {
				err = pub(client, configTopic, string(discoJson))
				if err != nil {
					//log error and ignore
					log.Error("failed to publish to ", configTopic, " error ", err)
				}
			}
		}
	}
}
