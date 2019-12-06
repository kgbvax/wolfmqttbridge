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
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	refreshSessionURL  = "https://www.wolf-smartset.com/portal/api/portal/UpdateSession"
	authenticateURL    = "https://www.wolf-smartset.com/portal/connect/token2"
	parameterValuesURL = "https://www.wolf-smartset.com/portal/api/portal/GetParameterValues"
	createSessionURL   = "https://www.wolf-smartset.com/portal/api/portal/CreateSession"
)

func getParameterValues(bearerToken string, sessionId int, valueIDList []int64, lastUpdate string, sys System) (ParameterValuesResponse, error) {
	reqPayload := ParameterValuesRequest{1000, false,
		valueIDList,
		sys.GatewayID, sys.ID,
		"2019-11-22T19:35:06.7715496Z", false, sessionId}
	response := ParameterValuesResponse{}
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		log.Error("error marshalloing request: ", err)
		return response, err
	}

	log.Trace("about to request parameterValues from ", parameterValuesURL)
	log.Trace("request: ", string(payload))

	req, err := http.NewRequest("POST", parameterValuesURL, bytes.NewReader(payload))
	if err != nil {
		log.Error("error creating request ", err)
		return response, err
	}
	setStdHeader(req, bearerToken, "application/json")
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Error("parmeterValues request failed ", err)
		if res != nil {
			res.Body.Close()
		}
		return response, err
	}
	defer res.Body.Close()

	log.Trace("status ", res.Status)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error("error reading response ", err)
		return response, err
	}
	log.Trace("response ", string(body))

	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Error("error unmarshalling response ", err)
	}

	if res.StatusCode != 200 {
		log.Warn("recieved status ", res.Status)
		log.Debug("response ", string(body))

	}
	return response, err
}

type SystemStateRequest struct {
	SessionID  int `json:"SessionId"`
	SystemList []struct {
		SystemID  int `json:"SystemId"`
		GatewayID int `json:"GatewayId"`
	} `json:"SystemList"`
}

func getAuthToken(username string, password string) (AuthToken, error) {
	data := AuthToken{}

	payload := strings.NewReader(fmt.Sprintf("grant_type=password&username=%s&password=%s&scope=all", username, password))
	req, _ := http.NewRequest("POST", authenticateURL, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cache-control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}

	if err != nil {
		log.Error(err)
		return data, err
	}
	if res.StatusCode != 200 {
		log.Fatalf("attempt to get token failed, code=%v\n", res.Status)
		os.Exit(-1)
	}

	body, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal([]byte(body), &data)

	return data, err
}

func getSystemList(bearerToken string) (SystemList, error) {
	url := "https://www.wolf-smartset.com/portal/api/portal/GetSystemList"
	data := SystemList{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err)
		return data, err
	}
	setStdHeader(req, bearerToken, "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		return data, err
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		log.Error(err)
		return data, err
	}
	return data, err
}

func getGUIDescriptionForGateway(bearerToken string, gatewayId int, systemId int) (GuiDescription, error) {
	url := fmt.Sprintf("https://www.wolf-smartset.com/portal/api/portal/GetGuiDescriptionForGateway?GatewayId=%d&SystemId=%d", gatewayId, systemId)
	data := GuiDescription{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err)
		return data, err
	}

	setStdHeader(req, bearerToken, "")
	log.Trace("fetch GuiDescription.. ")

	res, err := http.DefaultClient.Do(req)
	log.Trace("done fetch GuiDescription")
	if err != nil {
		log.Error(err)
		return data, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body) //@TODO error handling
	if err != nil {
		log.Error(err)
		return data, err
	}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		log.Error(err)
		return data, err
	}

	return data, err
}

func createSession(bearerToken string) (int, error) {

	payload := strings.NewReader("{\n    \"Timestamp\": \"2019-11-04 21:53:50\"\n}")

	req, _ := http.NewRequest("POST", createSessionURL, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Add("User-Agent", "github.com/kgbvax/wolfmqttbridge 1")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Host", "www.wolf-smartset.com")
	req.Header.Add("Connection", "keep-alive")

	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()

	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		log.Fatal("attempt to establish session failed, code: ", res.Status)
	}

	var sessId int
	_, err = fmt.Fscanf(res.Body, "%d", &sessId)

	return sessId, err
}

func sessionRefresh(bearerToken string, sessionid int) {
	sess := SessionStr{sessionid}

	payload, err := json.Marshal(sess)
	if err != nil {
		log.Error("failed to marshal session")
	}
	log.Debug("refreshing session")

	payLoadReader := bytes.NewReader(payload)
	log.Trace("request ", string(payload))
	req, err := http.NewRequest("POST", refreshSessionURL, payLoadReader)
	if err != nil {
		log.Error(err.Error())
	}

	setStdHeader(req, bearerToken, "application/json")
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Error(err)
	} else {
		if res != nil {
			res.Body.Close()
		}
	}

	if log.IsLevelEnabled(log.TraceLevel) {
		body, _ := ioutil.ReadAll(res.Body)
		log.Trace("response ", string(body))
	}

	if res.StatusCode != 200 {
		log.Warn("irregular refresh status ", res.Status)
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
