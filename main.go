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
	"fmt"
	"github.com/One-com/gonelog/log"
	"github.com/One-com/gonelog/syslog"
	"github.com/bgentry/speakeasy"
	"github.com/mattn/go-isatty"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"time"
)

import _ "github.com/motemen/go-loghttp/global"

var app = kingpin.New("wolfmqttbridge", "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation.")
var debug = app.Flag("debug", "Enable debug mode").Short('d').Bool()
var wolf_user = app.Flag("user", "username at wolf-smartset.com").Envar("WOLF_USER").Required().String()
var wolf_pw = app.Flag("password", "Password for wolf-smartset.com.").Envar("WOLF_PW").Required().String()
var listParamCmd = app.Command("list", "list parameters available in gateway")
var brCmd = app.Command("br", "start bridge")

func main() {
	log.SetPrintLevel(syslog.LOG_INFO, true)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("0.1-SNAPSHOT").Author("vax@kgbvax.net")
	kingpin.CommandLine.Help = "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation."
	kingpin.CommandLine.HelpFlag.Short('h')
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debug == true {
		log.SetLevel(syslog.LOG_DEBUG)
	}

	log.SetFlags(log.Llevel | log.Lcolor | log.Lname | log.LUTC)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.DEBUG("Auto Colouring on")
		log.AutoColoring()
	}

	if wolf_pw == nil {
		*wolf_pw = askPw()
	}

	log.DEBUG("obtain auth token ", "user", *wolf_user)
	aTok, err := getAuthToken(*wolf_user, *wolf_pw)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	log.DEBUG("create session")
	sessId, err := createSession(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	go sessionRefresh(aTok.AccessToken, sessId)

	log.DEBUG("get system list")
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

	log.INFO("System", "ID", system.ID)
	log.INFO("System", "Name", system.Name)
	log.INFO("Gateway", "ID", system.GatewayID)
	log.INFO("Gateway", "Software Version", system.GatewaySoftwareVersion)

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
	log.DEBUG("main", "cmd", cmd)
	switch cmd {
	case listParamCmd.FullCommand():
		{
			guiDescription, _ := getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
			printGuiParameters(guiDescription)
		}

	case brCmd.FullCommand():
		{
			log.DEBUG("start bridge")
			lastUpdate := "2019-11-22" //some date in the past
			guiDescription, _ := getGUIDescriptionForGateway(token.AccessToken, system.GatewayID, system.ID)
			params := getPollParams(guiDescription)

			for {
				var valIdList []int64
				for _, param := range params {
					valIdList = append(valIdList, param.ValueID)
				}

				paramterValuesResponse := getParameterValues(token.AccessToken, sessId, valIdList, lastUpdate, system)
				lastUpdate = paramterValuesResponse.LastAccess
				for _, value := range paramterValuesResponse.Values {
					found := false
					for _, param := range params { //join with parameter meta
						if param.ValueID == value.ValueID {
							log.DEBUG("value response ", "name", param.Name, "value", value.Value)
							found = true
						}
					}
					if found == false {
						log.ERROR("value not found in parameterDescription", "valueID", value.ValueID)
					}

				}
				time.Sleep(20 * time.Second)

			}
		}
	}

}
