package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

// Patch osExit for testing
var osExit = os.Exit

var isSystemInNetwork = func(pip, pnetwork string) bool {
	// Define the IP address and the CIDR range
	ip := net.ParseIP(pip)
	pnet := fmt.Sprintf("%s/24", pnetwork)
	_, network, err := net.ParseCIDR(pnet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CIDR: %v\n", err)
		return false
	}
	return network.Contains(ip)

}

func sumaGetSystemID(sessioncookie, susemgr, hostname string, verbose bool) (id int, err error) {

	type ResultSystemGetId struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}

	type ResponseSystemGetId struct {
		Success bool                `json:"success"`
		Result  []ResultSystemGetId `json:"result"`
	}
	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemID: apiURL =  %s\n", apiURL)
	}

	/*
	 check if system is registered
	*/
	apiMethodgetSystemID := fmt.Sprintf("%s%s%s", apiURL, "/system/getId?name=", hostname)
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemID: apiMethod = %s\n", apiMethodgetSystemID)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodgetSystemID, nil)
	if err != nil {
		log.Printf("error creating request to get hostname, error: %s\n", err)
		return -1, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %s\n", err)
		return -1, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP Request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %s\n", err)
		return -1, err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemID: Got resp.Body = %s\n", string(bodyBytes))
	}

	// Unmarshal the JSON response into the struct
	var rsp ResponseSystemGetId
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		log.Printf("error unmarshaling JSON: %s\n", err)
		return -1, err
	}

	// Extract and print all fields
	var foundID int
	for _, r := range rsp.Result {
		foundID = r.Id
	}

	if foundID == 0 {
		log.Printf("host: %s not found in SUSE Manager on %s\n", hostname, susemgr)
		return -1, fmt.Errorf("host: %s not found in SUSE Manager on %s", hostname, susemgr)
	}

	return foundID, nil

}

func sumaGetSystemIP(sessioncookie, susemgr string, id int, verbose bool) (foundIP string, err error) {

	type ResultSystemGetIp struct {
		Ip   string `json:"ip"`
		Name string `json:"hostname"`
	}

	type ResponseSystemGetIp struct {
		Success bool              `json:"success"`
		Result  ResultSystemGetIp `json:"result"`
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemIP: apiURL =  %s\n", apiURL)
	}

	/*
	 check if system is registered
	*/
	apiMethodgetSystemIP := fmt.Sprintf("%s%s%d", apiURL, "/system/getNetwork?sid=", id)
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemIP: apiMethod = %s\n", apiMethodgetSystemIP)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodgetSystemIP, nil)
	if err != nil {
		log.Printf("error creating request to get IP from system, error: %s\n", err)
		return "", err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %s\n", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Request failed: HTTP %d\n", resp.StatusCode)
		return "", fmt.Errorf("HTTP Request failed: HTTP/%d", resp.StatusCode)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http response: %s\n", err)
		return "", err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI sumaGetSystemIP: Got resp.Body = %s\n", string(bodyBytes))
	}
	// Unmarshal the JSON response into the struct
	var rsp ResponseSystemGetIp
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		log.Printf("error unmarshaling JSON: %s\n", err)
		return "", err
	}

	// Extract and print all fields
	foundIP = rsp.Result.Ip

	if foundIP == "" {
		log.Printf("ID: %d not found in SUSE Manager on %s\n", id, susemgr)
		return "", fmt.Errorf("ID: %d not found in SUSE Manager on %s", id, susemgr)
	}

	if verbose {
		log.Printf("DEBUG: Found IP = %s\n", foundIP)
	}
	return foundIP, nil

}

// Login try to login to SUSE Manager. Username, Password are get from Hashicorp Vault.
func SumaLogin(username, password, susemgr string, verbose bool) (sessioncookie string, err error) {

	type AuthRequest struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI SumaLogin: Enter function Login")
		log.Println("DEBUG SUMAAPI SumaLogin: ====================")
		defer log.Println("DEBUG SUMAAPI SumaLogin: Leave function Login")
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaLogin: apiURL = %s\n", apiURL)
	}

	apiMethod := fmt.Sprintf("%s%s", apiURL, "/auth/login")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaLogin: apiMethod = %s", apiMethod)
	}

	// Create the authentication request payload
	authPayload := AuthRequest{
		Login:    username,
		Password: password,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		log.Printf("error marshalling payload: %v\n", err)
		return "", err
	}

	// Create an HTTP POST request
	req, err := http.NewRequest("POST", apiMethod, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v\n", err)
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	// Extract the session cookie from the response headers
	cookies := resp.Cookies()

	for _, cookie := range cookies {
		if verbose {
			log.Printf("DEBUG SUMAAPI SumaLogin: Cookie Name: %s, Cookie Value: %s, Cookie MaxAge: %d\n", cookie.Name, cookie.Value, cookie.MaxAge)
		}
		if cookie.Name == "pxt-session-cookie" && cookie.MaxAge == 3600 {
			sessioncookie = cookie.Value
		}
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaLogin: Session Cookie = %s\n", sessioncookie)
		log.Printf("DEBUG SUMAAPI SumaLogin: Response status = %s\n", resp.Status)
	}

	// Handle the response body if needed
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		log.Printf("got error to read from respone body.\n")
		return "", err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaLogin: Response body =  %s\n", responseBody.String())
	}

	return sessioncookie, nil
}

// AddSystem add a System to a SUSE Manager SystemGroup.
func SumaAddSystem(sessioncookie, susemgr, hostname, group, network string, verbose bool) (statuscode int, err error) {

	type AddRemoveSystem struct {
		SystemGroupName string `json:"systemGroupName"`
		ServerIds       []int  `json:"serverIds"`
		Add             bool   `json:"add"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI SumaAddSystem: Enter function")
		log.Println("DEBUG SUMAAPI SumaAddSystem: ==============")
		defer log.Println("DEBUG SUMAAPI SumaAddSystem: Leave function")
	}

	foundID, err := sumaGetSystemID(sessioncookie, susemgr, hostname, verbose)
	if err != nil {
		log.Printf("could not get system id, errorcode = %v", err)
		return -1, err
	}

	if foundID == 0 {
		return -1, fmt.Errorf("did not found the system in SUSE Manager.")
	}

	foundIP, err := sumaGetSystemIP(sessioncookie, susemgr, foundID, verbose)
	if err != nil {
		log.Fatalf("Could not get IP, errorcode: %v", err)
	}

	if foundIP == "" {
		return -1, fmt.Errorf("did not found the system ID %d in SUSE Manager.", foundID)
	}

	isValid := isSystemInNetwork(foundIP, network)

	if !isValid {
		return -1, fmt.Errorf("system cannot be added. The system does not belong to the permitted network!")
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddSystem: apiURL =  %s\n", apiURL)
	}

	apiMethodAddOrRemoveSystems := fmt.Sprintf("%s%s", apiURL, "/systemgroup/addOrRemoveSystems")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddSystem: apiMethod = %s\n", apiMethodAddOrRemoveSystems)
	}

	// Create the authentication request payload
	AddRemoveSystemPayload := AddRemoveSystem{
		SystemGroupName: group,
		ServerIds:       []int{foundID},
		Add:             true,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(AddRemoveSystemPayload)
	if err != nil {
		log.Printf("Error marshalling payload: %v\n", err)
		return -1, err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddSystem: Payload =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiMethodAddOrRemoveSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return -1, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v\n", err)
		return -1, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body:", err)
		}
	}()

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddSystem: Add Node: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Request failed: HTTP %d\n", resp.StatusCode)
		return -1, err
	}

	return resp.StatusCode, nil

}

// DeleteSystem delete a System from the SUSE Manager . This implies, that it is also deleted from the SUSE Manager SystemGroup.
// To ensure, that DeleteSystem could not delete other Systems from o differen IP range, the procedure check if the IP belongs
// to the IP range we get from hashicorp vault.
func SumaDeleteSystem(sessioncookie, susemgr, hostname, network string, verbose bool) (statsucode int, err error) {

	type DeleteSystemType struct {
		ServerId    int    `json:"sid"`
		CleanupType string `json:"cleanupType"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI SumeDeleteSystem: Enter function")
		log.Println("DEBUG SUMAAPI SumeDeleteSystem: ==============")
		defer log.Println("DEBUG SUMAAPI SumeDeleteSystem: Leave function")
	}

	foundID, err := sumaGetSystemID(sessioncookie, susemgr, hostname, verbose)
	if err != nil {
		log.Printf("could not get system id, errorcode = %v\n", err)
		return -1, err
	}

	if foundID == 0 {
		return -1, fmt.Errorf("Did not find the system in SUSE Manager.")
	}

	foundIP, err := sumaGetSystemIP(sessioncookie, susemgr, foundID, verbose)
	if err != nil {
		log.Printf("Could not get IP, errorcode: %v", err)
		return -1, err
	}

	if foundIP == "" {
		return -1, fmt.Errorf("did not find the system ID %d in SUSE Manager.", foundID)
	}

	isValid := isSystemInNetwork(foundIP, network)

	if !isValid {
		return -1, fmt.Errorf("%s cannot be deleted. The system does not belong to the permitted network of the group!", hostname)
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaDeleteSystem: apiURL =  %s\n", apiURL)
	}

	apiDeleteSystems := fmt.Sprintf("%s%s", apiURL, "/system/deleteSystem")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaDeleteSystem: apiMethod = %s\n", apiDeleteSystems)
	}

	// Create the authentication request payload
	DeleteSystemPayload := DeleteSystemType{
		ServerId:    foundID,
		CleanupType: "FORCE_DELETE",
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(DeleteSystemPayload)
	if err != nil {
		log.Printf("error marshalling payload: %v\n", err)
		return -1, err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaDeleteSystem: Paylod =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiDeleteSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return -1, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v\n", err)
		return -1, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaDeleteSystem: Delete Node: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("HTTP Request failed: HTTP/%d", resp.StatusCode)
	}

	return resp.StatusCode, nil

}

func sumaCheckUser(sessioncookie, group, susemgrurl string, verbose bool) (exists bool) {

	type responseUserListUsers struct {
		Success bool `json:"success"`
		Result  []struct {
			Login string `json:"login"`
		} `json:"result"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI sumaCheckUser: Enter function sumaCheckUser")
		log.Println("DEBUG SUMAAPI sumaCheckUser: ============================")
		defer log.Println("DEBUG SUMAAPI sumaCheckUser: Leave function sumaCheckUser")
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgrurl, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaCheckUser: apiURL =  %s\n", apiURL)
	}

	apiUserListUsers := fmt.Sprintf("%s%s", apiURL, "/user/listUsers")
	if verbose {
		log.Printf("DEBUG SUMAAPI sumaCheckUser: apiMethod = %s\n", apiUserListUsers)
	}

	req, err := http.NewRequest(http.MethodGet, apiUserListUsers, nil)
	if err != nil {
		log.Printf("error creating request to get user list, error: %s\n", err)
		osExit(1)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %s\n", err)
		osExit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("http request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading http, got response: %s\n", err)
		osExit(1)
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI sumaCheckUser: Got resp.Body = %s\n", string(bodyBytes))
	}

	// Unmarshal the JSON response into the struct
	var rsp responseUserListUsers
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		log.Printf("error unmarshaling JSON: %s\n", err)
		osExit(1)
	}

	for _, user := range rsp.Result {
		if verbose {
			log.Printf("DEBUG SUMAAPI sumaCheckUser: User in SUMA: %s\n", user.Login)
		}
		if user.Login == group {
			return true
		}
	}

	return false
}

func SumaAddUser(sessioncookie, group, grouppassword, susemgrurl string, verbose bool) (statuscode int, err error) {

	type AddUser struct {
		Login     string `json:"login"`
		Password  string `json:"password"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI SumaAddUser: Enter function")
		log.Println("DEBUG SUMAAPI SumaAddUser: ==============")
		defer log.Println("DEBUG SUMAAPI SumaAddUser: Leave function")
	}

	//check if user exists
	ok := sumaCheckUser(sessioncookie, group, susemgrurl, verbose)

	if ok {
		log.Fatalf("user %s already exists in SUMA.\n", group)
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgrurl, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddUser: apiURL =  %s\n", apiURL)
	}

	apiUserCreate := fmt.Sprintf("%s%s", apiURL, "/user/create")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddUser: apiMethod = %s\n", apiUserCreate)
	}

	// Create the authentication request payload
	AddUserPayload := AddUser{
		Login:     group,
		Password:  grouppassword,
		FirstName: group,
		LastName:  group,
		Email:     "root@localhost",
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(AddUserPayload)
	if err != nil {
		log.Printf("error marshalling payload: %v\n", err)
		return 1, err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddUser: Payload =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiUserCreate, bytes.NewBuffer(payloadBytes))

	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return 1, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v\n", err)
		return 1, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddUser: Add User: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Request failed: HTTP %d\n", resp.StatusCode)
		return 1, err
	}

	return resp.StatusCode, nil
}

func SumaRemoveUser(sessioncookie, group, susemgrurl string, verbose bool) (err error) {

	type RemoveUser struct {
		Login string `json:"login"`
	}

	if verbose {
		log.Println("DEBUG SUMAAPI SumaRemoveUser: Enter function")
		log.Println("DEBUG SUMAAPI SumaRemoveUser: ==============")
		defer log.Println("DEBUG SUMAAPI SumaRemoveUser: Leave function")
		log.Printf("DEBUG SUMAAPI SumaRemoveUser: sessioncookie: %s\n", sessioncookie)
	}

	//check if user exists
	ok := sumaCheckUser(sessioncookie, group, susemgrurl, verbose)

	if !ok {
		log.Printf("user %s already removed in SUMA.\n", group)
		return nil
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgrurl, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaRemoveUser: apiURL =  %s\n", apiURL)
	}

	apiUserRemove := fmt.Sprintf("%s%s", apiURL, "/user/delete")
	if verbose {
		log.Printf("DEBUG SUMAAPI SumaRemoveUser: apiMethod = %s\n", apiUserRemove)
	}

	// Create the authentication request payload
	RemoveUserPayload := RemoveUser{
		Login: group,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(RemoveUserPayload)
	if err != nil {
		log.Printf("error marshalling payload: %v\n", err)
		return err
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaRemoveUser: Payload =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiUserRemove, bytes.NewBuffer(payloadBytes))

	if err != nil {
		log.Printf("error creating request: %v\n", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	if verbose {
		log.Printf("DEBUG SUMAAPI: SumaRemoveUser: %v\n", resp)
	}

	if err != nil {
		log.Printf("error sending request: %v\n", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("removing user %s failed, got http error %d", group, resp.StatusCode)
	}

	return nil
}

func GetApiList(sessioncookie, susemgr string, verbose bool) {
	type ResponseGetApiCallList struct {
		Name        string `json:"name"`
		Parameters  string `json:"parameters"`
		Exceptions  string `json:"string"`
		ReturnValue string `json:"return"`
	}

	log.Printf("DEBUG SUMAAPI GetApiList: sessioncookie =  %s\n", sessioncookie)

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI GetApiList: apiURL =  %s\n", apiURL)
	}

	apiApiCallList := fmt.Sprintf("%s%s", apiURL, "/api/getApiCallList")
	if verbose {
		log.Printf("DEBUG SUMAAPI GetApiList: apiMethod = %s\n", apiApiCallList)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiApiCallList, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request, error: %s\n", err)
		osExit(1)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "pxt-session-cookie",
		Value: sessioncookie,
	})

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %s\n", err)
		osExit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "Error closing response body:", err)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP Request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading http response: %s\n", err)
		osExit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Got resp.Body = %s\n", string(bodyBytes))
	}
	// Unmarshal the JSON response into the struct
	var rsp ResponseGetApiCallList
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %s\n", err)
		osExit(1)
	}

	fmt.Printf("%v", rsp)
}
