package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SessionStr struct {
	SessionID int `json:"SessionId"`
}


func createSession(bearerToken string) (int, error) {
	url := "https://www.wolf-smartset.com/portal/api/portal/CreateSession"

	payload := strings.NewReader("{\n    \"Timestamp\": \"2019-11-04 21:53:50\"\n}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s",bearerToken))
	req.Header.Add("User-Agent", "undef")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Host", "www.wolf-smartset.com")
 	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("cache-control", "no-cache")

	res, err := http.DefaultClient.Do(req)
	if (err!=nil) {
		return 0,err
	}

	var sessId int
/*	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)  //&bail out
	}

	fmt.Println("x",string(body))
	fmt.Println("x",res.Header) */

	_,err=fmt.Fscanf(res.Body,"%d",&sessId)

	defer res.Body.Close()

	return sessId,err
}

func sessionRefresh( bearerToken string,sessionid int) {
	sess:=SessionStr{sessionid}
	url := "https://www.wolf-smartset.com/portal/api/portal/UpdateSession"
	fmt.Printf("SessionId is %v\n",sessionid)

	payload,_ := json.Marshal(sess)
	for    {
		time.Sleep(6 * time.Second)
		fmt.Println("refreshing session")

		payLoadReader:=bytes.NewReader(payload)
		req, err := http.NewRequest("POST", url, payLoadReader)
		if (err!=nil) {
			fmt.Println(err.Error())
		}
		setStdHeader(req,bearerToken,"application/json")

		res, _ := http.DefaultClient.Do(req)

		if (err!=nil) {
			fmt.Println(err.Error())
		} else {
			res.Body.Close()
		}
	}
 }

func setStdHeader(request *http.Request, bearerToken string, contentType string) {
	request.Header.Add("Content-Type", contentType)
	request.Header.Add("cache-control", "no-cache")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s",bearerToken))
	request.Header.Add("Host", "www.wolf-smartset.com")
	request.Header.Add("Connection","keep-alive")
	request.Header.Add("User-Agent","WolfMQTTBridge/1.0")
	request.Header.Add("Accept", "*/*")
	request.Header.Add("X-Pect","The Spanish Inquisition")
	//request.Header.Add()
}