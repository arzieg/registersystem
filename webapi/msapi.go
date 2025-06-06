package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// functions
/*
MsLogin - erl
MsLogout - nicht notwendig
MsListBuildingBlocks - erl
MsCreateBuildingBlock - erl
MsGetBuildingBlockStatus
MsDeleteBuildingBlock

*/

type BuildingBlockType struct {
	Name string
	Uuid string
}

// MsLogin try to login into Meshstack with a api key and get a bearer token back
func MsLogin(clientid, clientsecret, apiurl string, verbose bool) (accesstoken string, err error) {

	var grant_type string = "client_credentials"

	if verbose {
		log.Println("DEBUG MSAPI MsLogin: ====================")
		log.Println("DEBUG MSAPI MsLogin: Enter function Login")

		defer log.Println("DEBUG MSAPI MSLogin: Leave function Login")
	}

	//  Define the API Method
	apiMethod := fmt.Sprintf("%s%s", apiurl, "/api/login")
	if verbose {
		log.Printf("DEBUG MSAPI MSLogin: apiMethod = %s", apiMethod)
	}

	// Create the authentication request payload
	payloadString := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=%s", clientid, clientsecret, grant_type)
	payloadStringWithoutPassword := fmt.Sprintf("client_id=%s&client_secret=XXXXXXX&grant_type=%s", clientid, grant_type)
	if verbose {
		log.Printf("DEBUG MSAPI MSLogin: payloadString = %s", payloadStringWithoutPassword)
	}

	// Create an HTTP POST request
	req, err := http.NewRequest("POST", apiMethod, bytes.NewBufferString(payloadString))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP(S) Reqeust failed. Got: %v\n", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Read respone Body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %v", err)
		return "", err
	}

	if verbose {
		log.Printf("DEBUG MSAPI msLogin: Got resp.Body = %s\n", string(bodyBytes))
	}

	// extract the authentication token
	type ResultMsLogin struct {
		AccessToken string `json:"access_token"`
	}

	// Unmarshal the JSON response into the struct
	var myaccesstoken ResultMsLogin
	err = json.Unmarshal(bodyBytes, &myaccesstoken)
	if err != nil {
		log.Printf("error unmarshaling JSON: %s\n", err)
		return "", err
	}

	if verbose {
		log.Printf("DEBUG MSAPI MsLogin: Access_Token = %s\n", myaccesstoken.AccessToken)
		log.Printf("DEBUG MSAPI MsLogin: Response status = %s\n", resp.Status)
	}

	return myaccesstoken.AccessToken, nil
}

func MsListBuildingBlocks(apiurl, projectid, apikey string, verbose bool) (bb []BuildingBlockType, err error) {

	var functionname string = "MsListBuildingBlocks"

	if verbose {
		log.Printf("DEBUG MSAPI %s: ===================================\n", functionname)
		log.Printf("DEBUG MSAPI %s: Enter function %s\n", functionname, functionname)

		defer log.Printf("DEBUG MSAPI %s: Leave function %s\n", functionname, functionname)
	}
	//  Define the API Method
	apiMethod := fmt.Sprintf("%s/api/meshobjects/meshbuildingblocks?projectIdentifier=%s", apiurl, projectid)
	if verbose {
		log.Printf("DEBUG MSAPI %s: apiMethod = %s", functionname, apiMethod)
	}

	// Create an HTTP GET request
	req, err := http.NewRequest(http.MethodGet, apiMethod, nil)
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return bb, err
	}
	bearerApikey := fmt.Sprintf("Bearer %s", apikey)
	req.Header.Set("Accept", "application/vnd.meshcloud.api.meshbuildingblock.v1.hal+json")
	req.Header.Set("Authorization", bearerApikey)

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP(S) Reqeust failed. Error: %v\n", err)
		return bb, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Read respone Body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %v", err)
		return bb, err
	}

	if verbose {
		log.Printf("DEBUG MSAPI %s: Got resp.Body = %s\n", functionname, string(bodyBytes))
	}

	// Unmarshal and extract UUID and DisplayName
	/*
		jsonData := `{
		  "_embedded": {
		    "meshBuildingBlocks": [
		      {
		        "kind": "meshBuildingBlock",
		        "apiVersion": "v1",
		        "metadata": {
		          "uuid": "xyz"
		        },
		        "spec": {
		          "displayName": "abc"
		        }
		      }
		    ]
		  }
		}`
	*/

	type Metadata struct {
		UUID string `json:"uuid"`
	}
	type Spec struct {
		DisplayName string `json:"displayName"`
	}
	type MeshBuildingBlockType struct {
		Metadata Metadata `json:"metadata"`
		Spec     Spec     `json:"spec"`
	}
	type Embedded struct {
		MeshBuildingBlockType []MeshBuildingBlockType `json:"meshBuildingBlocks"`
	}
	type Response struct {
		Embedded Embedded `json:"_embedded"`
	}

	var myvalues Response
	err = json.Unmarshal([]byte(bodyBytes), &myvalues)
	if err != nil {
		log.Printf("error unmarshal http response: %v", err)
		return bb, err
	}

	for _, item := range myvalues.Embedded.MeshBuildingBlockType {
		if verbose {
			log.Printf("UUID: %s, DisplayName: %s\n", item.Metadata.UUID, item.Spec.DisplayName)
		}
		newb := BuildingBlockType{Name: item.Spec.DisplayName, Uuid: item.Metadata.UUID}
		bb = append(bb, newb)
	}

	return bb, nil
}

func MsCreateBuildingBlock(apiurl, apikey string, payload []byte, verbose bool) (uuid string, err error) {

	var functionname string = "MsCreateBuildingBlock"

	if verbose {
		log.Printf("DEBUG MSAPI %s: ===================================\n", functionname)
		log.Printf("DEBUG MSAPI %s: Enter function %s\n", functionname, functionname)

		defer log.Printf("DEBUG MSAPI %s: Leave function %s\n", functionname, functionname)
	}
	//  Define the API Method
	apiMethod := fmt.Sprintf("%s/api/meshobjects/meshbuildingblocks", apiurl)
	if verbose {
		log.Printf("DEBUG MSAPI %s: apiMethod = %s", functionname, apiMethod)
	}

	if verbose {
		log.Printf("DEBUG MSAPI %s: payload = %s", functionname, payload)
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiMethod, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return "", err
	}
	bearerApikey := fmt.Sprintf("Bearer %s", apikey)
	req.Header.Set("Accept", "application/vnd.meshcloud.api.meshbuildingblock.v1.hal+json")
	req.Header.Set("Authorization", bearerApikey)
	req.Header.Set("Content-Type", "application/vnd.meshcloud.api.meshbuildingblock.v1.hal+json;charset=UTF-8")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP(S) Reqeust failed. Got: %v\n", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Read respone Body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %v", err)
		return "", err
	}

	if verbose {
		log.Printf("DEBUG MSAPI %s: Got resp.Body = %s\n", functionname, string(bodyBytes))
	}

	// get uuid

	type Metadata struct {
		UUID string `json:"uuid"`
	}
	type Response struct {
		Metadata Metadata `json:"metadata"`
	}

	var myuuid Response
	err = json.Unmarshal([]byte(bodyBytes), &myuuid)
	if err != nil {
		log.Printf("error unmarshal http response: %v", err)
		return "", err
	}

	uuid = myuuid.Metadata.UUID

	if verbose {
		log.Printf("UUID: %s\n", myuuid.Metadata.UUID)
	}

	return uuid, nil
}

func MsDeleteBuildingBlock(apiurl, apikey, uuid string, verbose bool) (err error) {

	var functionname string = "MsDeleteBuildingBlock"

	if verbose {
		log.Printf("DEBUG MSAPI %s: ===================================\n", functionname)
		log.Printf("DEBUG MSAPI %s: Enter function %s\n", functionname, functionname)

		defer log.Printf("DEBUG MSAPI %s: Leave function %s\n", functionname, functionname)
	}
	//  Define the API Method
	apiMethod := fmt.Sprintf("%s/api/meshobjects/meshbuildingblocks/%s", apiurl, uuid)
	if verbose {
		log.Printf("DEBUG MSAPI %s: apiMethod = %s", functionname, apiMethod)
	}

	// Create an HTTP DELETE request
	req, err := http.NewRequest(http.MethodDelete, apiMethod, nil)
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return err
	}
	bearerApikey := fmt.Sprintf("Bearer %s", apikey)
	req.Header.Set("Authorization", bearerApikey)

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP(S) Reqeust failed. Got: %v\n", err)
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	return nil
}

func MsGetBuildingBlock(apiurl, apikey, uuid string, verbose bool) (status string, err error) {

	var functionname string = "MsGetBuildingBlock"

	if verbose {
		log.Printf("DEBUG MSAPI %s: ===================================\n", functionname)
		log.Printf("DEBUG MSAPI %s: Enter function %s\n", functionname, functionname)

		defer log.Printf("DEBUG MSAPI %s: Leave function %s\n", functionname, functionname)
	}
	//  Define the API Method
	apiMethod := fmt.Sprintf("%s/api/meshobjects/meshbuildingblocks/%s", apiurl, uuid)
	if verbose {
		log.Printf("DEBUG MSAPI %s: apiMethod = %s", functionname, apiMethod)
	}

	// Create an HTTP GET request
	req, err := http.NewRequest(http.MethodGet, apiMethod, nil)
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return "", err
	}
	bearerApikey := fmt.Sprintf("Bearer %s", apikey)
	req.Header.Set("Accept", "application/vnd.meshcloud.api.meshbuildingblock.v1.hal+json")
	req.Header.Set("Authorization", bearerApikey)

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP(S) Reqeust failed. Got: %v\n", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Read respone Body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %v", err)
		return "", err
	}

	if verbose {
		log.Printf("DEBUG MSAPI %s: Got resp.Body = %s\n", functionname, string(bodyBytes))
	}

	// get status
	/*
		The status of the Building Block. One of
			WAITING_FOR_DEPENDENT_INPUT,
			WAITING_FOR_OPERATOR_INPUT,
			PENDING, IN_PROGRESS,
			SUCCEEDED,
			FAILED,
			ABORTED
	*/

	type Status struct {
		Status string `json:"status"`
	}
	type Response struct {
		Status Status `json:"status"`
	}

	var mystatus Response
	err = json.Unmarshal([]byte(bodyBytes), &mystatus)
	if err != nil {
		log.Printf("error unmarshal http response: %v", err)
		return "", err
	}

	status = mystatus.Status.Status

	if verbose {
		log.Printf("STATUS: %s\n", mystatus.Status.Status)
	}

	return status, nil
}
