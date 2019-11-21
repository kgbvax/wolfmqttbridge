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
	"github.com/One-com/gonelog/log"
	"github.com/One-com/gonelog/syslog"
	"github.com/bgentry/speakeasy"
	"github.com/mattn/go-isatty"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

//import _ "github.com/motemen/go-loghttp/global"

var debug = kingpin.Flag("debug", "Enable debug mode").Short('d').Bool()
var wolf_user = kingpin.Flag("user", "username at wolf-smartset.com").Envar("WOLF_USER").Required().String()
var wolf_pw = kingpin.Flag("password", "Password for wolf-smartset.com.").String()
var listParamCmd   = kingpin.Command("list", "list parameters available in gateway")
var brCmd = kingpin.Command("br", "start bridge")
var brParams = brCmd.Arg("parameterId","List of parameterId, see 'list' command").Int32List()

type GetParameterValuesReq struct {
	BundleID     int       `json:"BundleId"`
	IsSubBundle  bool      `json:"IsSubBundle"`
	ValueIDList  []int64   `json:"ValueIdList"`
	GatewayID    int       `json:"GatewayId"`
	SystemID     int       `json:"SystemId"`
	LastAccess   time.Time `json:"LastAccess"`
	GuiIDChanged bool      `json:"GuiIdChanged"`
	SessionID    int       `json:"SessionId"`
}


type AuthToken struct {
	AccessToken                 string `json:"access_token"`
	ExpiresIn                   int    `json:"expires_in"`
	TokenType                   string `json:"token_type"`
	RefreshToken                string `json:"refresh_token"`
	Scope                       string `json:"scope"`
	CultureInfoCode             string `json:"CultureInfoCode"`
	IsPasswordReset             bool   `json:"IsPasswordReset"`
	IsProfessional              bool   `json:"IsProfessional"`
	IsProfessionalPasswordReset bool   `json:"IsProfessionalPasswordReset"`
}

type SystemList []struct {
	ID                     int           `json:"Id"`
	GatewayID              int           `json:"GatewayId"`
	IsForeignSystem        bool          `json:"IsForeignSystem"`
	AccessLevel            int           `json:"AccessLevel"`
	GatewayUsername        string        `json:"GatewayUsername"`
	Name                   string        `json:"Name"`
	SystemShares           []interface{} `json:"SystemShares"`
	GatewaySoftwareVersion string        `json:"GatewaySoftwareVersion"`
}

type SystemStateRequest struct {
	SessionID  int `json:"SessionId"`
	SystemList []struct {
		SystemID  int `json:"SystemId"`
		GatewayID int `json:"GatewayId"`
	} `json:"SystemList"`
}

func getAuthToken(username string, password string) (AuthToken, error) {
	url := "https://www.wolf-smartset.com/portal/connect/token2"
	data := AuthToken{}

	payload := strings.NewReader(fmt.Sprintf("grant_type=password&username=%s&password=%s&scope=all",username,password))
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cache-control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return data,err
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err=json.Unmarshal([]byte(body), &data)

	return data,err
}


func getSystemList(bearerToken string) (SystemList, error) {
	url := "https://www.wolf-smartset.com/portal/api/portal/GetSystemList?_=1574201847834"

	req, _ := http.NewRequest("GET", url, nil)
	setStdHeader(req,bearerToken,"")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	data := SystemList{}
	err:=json.Unmarshal([]byte(body), &data)
	return data,err
}

func getGUIDescriptionForGateway(bearerToken string, gatewayId int, systemId int) (GuiDescription, error) {
	url := fmt.Sprintf("https://www.wolf-smartset.com/portal/api/portal/GetGuiDescriptionForGateway?GatewayId=%d&SystemId=%d",gatewayId,systemId)

	req, _ := http.NewRequest("GET", url, nil)

	setStdHeader(req,bearerToken,"")

	res, _ := http.DefaultClient.Do(req) //@TODO error handling

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)  //@TODO error handling



	data:=GuiDescription{}
	err := json.Unmarshal([]byte(body), &data)
	return data,err
}

func main() {
	log.SetPrintLevel(syslog.LOG_INFO, true)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("0.1-SNAPSHOT").Author("vax@kgbvax.net")
	kingpin.CommandLine.Help = "Wolf Smartset MQTT Bridge, see github.com/kgbvax/wolfmqttbridge for documentation."
	kingpin.CommandLine.HelpFlag.Short('h')

	cmd := kingpin.Parse()

	if *debug == true {
		log.SetLevel(syslog.LOG_DEBUG)
 	}

	doTheHustle(cmd)

	log.SetFlags(log.Llevel | log.Lcolor | log.Lname | log.LUTC)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.DEBUG("Auto Colouring on")
		log.AutoColoring()
	}

	aTok,err:=getAuthToken("kgbvax","matrod-gecjY0-gyhzys")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)  //&bail out
	}
	sessId, err :=createSession(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}
	go sessionRefresh(aTok.AccessToken,sessId)

	sysList,err := getSystemList(aTok.AccessToken)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1) //&bail out
	}

	if (len(sysList)<1) {
		fmt.Println("System list is empty, nothing to do.")
		os.Exit(0)
	}

	sl:=sysList[0]
	fmt.Printf("ID: %v\n",sl.ID)
	fmt.Printf("GatewayID: %v\n",sl.GatewayID)
	fmt.Printf("Gateway Software Version: %s\n",sl.GatewaySoftwareVersion)
	fmt.Printf("Name: %s\n",sl.Name)

	guiDescription,_:=getGUIDescriptionForGateway(aTok.AccessToken,sl.GatewayID,sl.ID)
	printGuiParameters(guiDescription)

	time.Sleep(60*time.Second)


}

// Ask for a user's password if it's not given
func askPw(user string, pw string) string {
	if pw == "" {
		var err error
		pw, err = speakeasy.Ask(user + "'s password: ")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return pw
}

func doTheHustle(s string) {


}
