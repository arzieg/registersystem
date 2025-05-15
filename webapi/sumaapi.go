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

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ResultSystemGetId struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ResultSystemGetIp struct {
	Ip   string `json:"ip"`
	Name string `json:"hostname"`
}

type ResponseSystemGetId struct {
	Success bool                `json:"success"`
	Result  []ResultSystemGetId `json:"result"`
}

type ResponseSystemGetIp struct {
	Success bool              `json:"success"`
	Result  ResultSystemGetIp `json:"result"`
}

type ResponseGetApiCallList struct {
	Name        string `json:"name"`
	Parameters  string `json:"parameters"`
	Exceptions  string `json:"string"`
	ReturnValue string `json:"return"`
}

type responseUserListUsers struct {
	Success bool `json:"success"`
	Result  []struct {
		Login string `json:"login"`
	} `json:"result"`
}

type AddRemoveSystem struct {
	SystemGroupName string `json:"systemGroupName"`
	ServerIds       []int  `json:"serverIds"`
	Add             bool   `json:"add"`
}

type AddUser struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type RemoveUser struct {
	Login string `json:"login"`
}

type DeleteSystemType struct {
	ServerId    int    `json:"sid"`
	CleanupType string `json:"cleanupType"`
}

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

var getSystemID = func(sessioncookie, susemgr, hostname string, verbose bool) int {

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL =  %s\n", apiURL)
	}

	/*
	 check if system is registered
	*/
	apiMethodgetSystemID := fmt.Sprintf("%s%s%s", apiURL, "/system/getId?name=", hostname)
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s\n", apiMethodgetSystemID)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodgetSystemID, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request to get hostname, error: %s\n", err)
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
	var rsp ResponseSystemGetId
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %s\n", err)
		osExit(1)
	}

	// Extract and print all fields
	var foundID int
	for _, r := range rsp.Result {
		foundID = r.Id
	}

	if foundID == 0 {
		fmt.Fprintf(os.Stderr, "Host: %s not found in SUSE Manager on %s\n", hostname, susemgr)
		osExit(1)
	}

	return foundID

}

var getSystemIP = func(sessioncookie, susemgr string, id int, verbose bool) string {

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL =  %s\n", apiURL)
	}

	/*
	 check if system is registered
	*/
	apiMethodgetSystemIP := fmt.Sprintf("%s%s%d", apiURL, "/system/getNetwork?sid=", id)
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s\n", apiMethodgetSystemIP)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodgetSystemIP, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request to get IP from system, error: %s\n", err)
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
	var rsp ResponseSystemGetIp
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %s\n", err)
		osExit(1)
	}

	// Extract and print all fields
	foundIP := rsp.Result.Ip

	if foundIP == "" {
		fmt.Fprintf(os.Stderr, "ID: %d not found in SUSE Manager on %s\n", id, susemgr)
		osExit(1)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Found IP = %s\n", foundIP)

	return foundIP

}

// Login try to login to SUSE Manager. Username, Password are get from Hashicorp Vault.
func Login(username, password, susemgr string, verbose bool) string {

	if verbose {
		log.Println("DEBUG SUMAAPI: Enter function Login")
		log.Println("DEBUG SUMAAPI: ====================")
		defer log.Println("DEBUG SUMAAPI: Leave function Login")
	}

	var sessioncookie string

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		log.Printf("DEBUG SUMAAPI: apiURL = %s\n", apiURL)
	}

	apiMethod := fmt.Sprintf("%s%s", apiURL, "/auth/login")
	if verbose {
		log.Printf("DEBUG SUMAAPI: apiMethod = %s", apiMethod)
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
		osExit(1)
	}

	// Create an HTTP POST request
	req, err := http.NewRequest("POST", apiMethod, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("error creating request: %v\n", err)
		osExit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending request: %v\n", err)
		osExit(1)
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
			log.Printf("DEBUG SUMAAPI: Cookie Name: %s, Cookie Value: %s, Cookie MaxAge: %d\n", cookie.Name, cookie.Value, cookie.MaxAge)
		}
		if cookie.Name == "pxt-session-cookie" && cookie.MaxAge == 3600 {
			sessioncookie = cookie.Value
		}
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI: Session Cookie = %s\n", sessioncookie)
		// Print the response status
		log.Printf("DEBUG SUMAAPI: Response status = %s\n", resp.Status)
	}

	// Handle the response body if needed
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error to read from respone body.\n")
		osExit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response body =  %s\n", responseBody.String())
	}

	return sessioncookie
}

// AddSystem add a System to a SUSE Manager SystemGroup.
func AddSystem(sessioncookie, susemgr, hostname, group, network string, verbose bool) int {

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Enter function AddSystem\n")
		fmt.Fprintf(os.Stderr, "DEBUG: ========================\n")
		defer fmt.Fprintf(os.Stderr, "DEBUG: Leave function AddSystem \n")
	}

	/*
	 add System to Group
	*/
	foundID := getSystemID(sessioncookie, susemgr, hostname, verbose)

	if foundID == 0 {
		fmt.Fprintf(os.Stderr, "Did not find the system in SUSE Manager.\n")
		osExit(1)
	}

	foundIP := getSystemIP(sessioncookie, susemgr, foundID, verbose)

	if foundIP == "" {
		fmt.Fprintf(os.Stderr, "Did not find the system ID %d in SUSE Manager.\n", foundID)
		osExit(1)
	}

	isValid := isSystemInNetwork(foundIP, network)

	if !isValid {
		fmt.Fprintf(os.Stderr, "System cannot be added. The system does not belong to the permitted network!\n")
		osExit(1)
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL =  %s\n", apiURL)
	}

	apiMethodAddOrRemoveSystems := fmt.Sprintf("%s%s", apiURL, "/systemgroup/addOrRemoveSystems")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s\n", apiMethodAddOrRemoveSystems)
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
		fmt.Fprintf(os.Stderr, "Error marshalling payload: %v\n", err)
		osExit(1)
	}

	if verbose {
		fmt.Printf("DEBUG: Paylod =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiMethodAddOrRemoveSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		osExit(1)
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
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		osExit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "Error closing response body:", err)
		}
	}()

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Add Node: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP Request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	return resp.StatusCode

}

// DeleteSystem delete a System from the SUSE Manager . This implies, that it is also deleted from the SUSE Manager SystemGroup.
// To ensure, that DeleteSystem could not delete other Systems from o differen IP range, the procedure check if the IP belongs
// to the IP range we get from hashicorp vault.
func DeleteSystem(sessioncookie, susemgr, hostname, network string, verbose bool) int {

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Enter function DeleteSystem\n")
		fmt.Fprintf(os.Stderr, "DEBUG: ===========================\n")
		defer fmt.Fprintf(os.Stderr, "DEBUG: Leave function DeleteSystem \n")
	}

	/*
	 delete System
	*/

	foundID := getSystemID(sessioncookie, susemgr, hostname, verbose)

	if foundID == 0 {
		fmt.Fprintf(os.Stderr, "Did not find the system in SUSE Manager.\n")
		osExit(1)
	}

	foundIP := getSystemIP(sessioncookie, susemgr, foundID, verbose)

	if foundIP == "" {
		fmt.Fprintf(os.Stderr, "Did not find the system ID %d in SUSE Manager.\n", foundID)
		osExit(1)
	}

	isValid := isSystemInNetwork(foundIP, network)

	if !isValid {
		fmt.Fprintf(os.Stderr, "%s cannot be deleted. The system does not belong to the permitted network of the group!\n", hostname)
		osExit(1)
	}

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL =  %s\n", apiURL)
	}

	apiDeleteSystems := fmt.Sprintf("%s%s", apiURL, "/system/deleteSystem")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s\n", apiDeleteSystems)
	}

	// Create the authentication request payload
	DeleteSystemPayload := DeleteSystemType{
		ServerId:    foundID,
		CleanupType: "FORCE_DELETE",
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(DeleteSystemPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling payload: %v\n", err)
		osExit(1)
	}

	if verbose {
		fmt.Printf("DEBUG: Paylod =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiDeleteSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		osExit(1)
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
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		osExit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "Error closing response body:", err)
		}
	}()

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Delete Node: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP Request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	return resp.StatusCode

}

func sumaCheckUser(sessioncookie, group, susemgrurl string, verbose bool) (exists bool) {

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

func SumaAddUser(sessioncookie, group, grouppassword, susemgrurl string, verbose bool) int {

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
		osExit(1)
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaAddUser: Payload =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiUserCreate, bytes.NewBuffer(payloadBytes))

	if err != nil {
		log.Printf("error creating request: %v\n", err)
		osExit(1)
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
		osExit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	if verbose {
		log.Printf("DEBUG SUMAAPI: Add User: %v\n", resp)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP Request failed: HTTP %d\n", resp.StatusCode)
		osExit(1)
	}

	return resp.StatusCode
}

func SumaRemoveUser(sessioncookie, group, susemgrurl string, verbose bool) (err error) {

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
		log.Fatalf("error marshalling payload: %v\n", err)
	}

	if verbose {
		log.Printf("DEBUG SUMAAPI SumaRemoveUser: Payload =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiUserRemove, bytes.NewBuffer(payloadBytes))

	if err != nil {
		log.Fatalf("error creating request: %v\n", err)
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

	// Extract and print all fields
	// foundIP := rsp.Result.Ip

	// if foundIP == "" {
	// 	fmt.Fprintf(os.Stderr, "ID: %d not found in SUSE Manager on %s\n", id, susemgr)
	// 	osExit(1)
	// }

	// fmt.Fprintf(os.Stderr, "DEBUG: Found IP = %s\n", foundIP)

}
