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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func getParameterValues(bearerToken string, sessionId int, valueIDList []int64, lastUpdate string, sys System) ParameterValuesResponse {
	reqPayload := ParameterValuesRequest{1000, false,
		valueIDList,
		sys.GatewayID, sys.ID,
		"2019-11-22T19:35:06.7715496Z", false, sessionId}
	url := "https://www.wolf-smartset.com/portal/api/portal/GetParameterValues"
	payload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	setStdHeader(req, bearerToken, "application/json")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	response := ParameterValuesResponse{}

	json.Unmarshal([]byte(body), &response)
	return response
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

	payload := strings.NewReader(fmt.Sprintf("grant_type=password&username=%s&password=%s&scope=all", username, password))
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cache-control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return data, err
	}
	if res.StatusCode != 200 {
		log.Fatalf("attempt to get token failed, code=%v\n", res.Status)
		os.Exit(-1)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal([]byte(body), &data)

	return data, err
}

func getSystemList(bearerToken string) (SystemList, error) {
	url := "https://www.wolf-smartset.com/portal/api/portal/GetSystemList"

	req, _ := http.NewRequest("GET", url, nil)
	setStdHeader(req, bearerToken, "")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	data := SystemList{}
	err := json.Unmarshal([]byte(body), &data)
	return data, err
}

func getGUIDescriptionForGateway(bearerToken string, gatewayId int, systemId int) (GuiDescription, error) {
	url := fmt.Sprintf("https://www.wolf-smartset.com/portal/api/portal/GetGuiDescriptionForGateway?GatewayId=%d&SystemId=%d", gatewayId, systemId)

	req, _ := http.NewRequest("GET", url, nil)

	setStdHeader(req, bearerToken, "")
	res, _ := http.DefaultClient.Do(req) //@TODO error handling
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body) //@TODO error handling

	data := GuiDescription{}
	err := json.Unmarshal([]byte(body), &data)
	return data, err
}

func createSession(bearerToken string) (int, error) {
	url := "https://www.wolf-smartset.com/portal/api/portal/CreateSession"

	payload := strings.NewReader("{\n    \"Timestamp\": \"2019-11-04 21:53:50\"\n}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Add("User-Agent", "undef")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Host", "www.wolf-smartset.com")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("cache-control", "no-cache")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		log.Fatalf("attempt to establish session failed, code=%v\n", res.Status)
	}

	var sessId int

	_, err = fmt.Fscanf(res.Body, "%d", &sessId)

	defer res.Body.Close()

	return sessId, err
}

func sessionRefresh(bearerToken string, sessionid int) {
	sess := SessionStr{sessionid}
	url := "https://www.wolf-smartset.com/portal/api/portal/UpdateSession"
	fmt.Printf("SessionId is %v\n", sessionid)

	payload, _ := json.Marshal(sess)
	for {
		time.Sleep(60 * time.Second)
		fmt.Println("refreshing session")

		payLoadReader := bytes.NewReader(payload)
		req, err := http.NewRequest("POST", url, payLoadReader)
		if err != nil {
			fmt.Println(err.Error())
		}
		setStdHeader(req, bearerToken, "application/json")

		res, _ := http.DefaultClient.Do(req)

		if err != nil {
			fmt.Println(err.Error())
		} else {
			res.Body.Close()
		}
	}
}

func setStdHeader(request *http.Request, bearerToken string, contentType string) {
	request.Header.Add("Content-Type", contentType)
	request.Header.Add("cache-control", "no-cache")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	request.Header.Add("Host", "www.wolf-smartset.com")
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("User-Agent", "WolfMQTTBridge/1.0")
	request.Header.Add("Accept", "*/*")
	request.Header.Add("X-Pect", "The Spanish Inquisition")
}

type SessionStr struct {
	SessionID int `json:"SessionId"`
}

type ParameterValuesRequest struct {
	BundleID     int     `json:"BundleId"`
	IsSubBundle  bool    `json:"IsSubBundle"`
	ValueIDList  []int64 `json:"ValueIdList"`
	GatewayID    int     `json:"GatewayId"`
	SystemID     int     `json:"SystemId"`
	LastAccess   string  `json:"LastAccess"`
	GuiIDChanged bool    `json:"GuiIdChanged"`
	SessionID    int     `json:"SessionId"`
}

type ParameterValuesResponse struct {
	LastAccess string `json:"LastAccess"`
	Values     []struct {
		ValueID int64  `json:"ValueId"`
		Value   string `json:"Value"`
		State   int    `json:"State"`
	} `json:"Values"`
	IsNewJobCreated bool `json:"IsNewJobCreated"`
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

type System struct {
	ID                     int           `json:"Id"`
	GatewayID              int           `json:"GatewayId"`
	IsForeignSystem        bool          `json:"IsForeignSystem"`
	AccessLevel            int           `json:"AccessLevel"`
	GatewayUsername        string        `json:"GatewayUsername"`
	Name                   string        `json:"Name"`
	SystemShares           []interface{} `json:"SystemShares"`
	GatewaySoftwareVersion string        `json:"GatewaySoftwareVersion"`
}
type SystemList []System
