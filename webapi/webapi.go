package webapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	vault "github.com/hashicorp/vault/api"
)

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ResultSystemGetId struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ResponseSystemGetId struct {
	Success bool                `json:"success"`
	Result  []ResultSystemGetId `json:"result"`
}

type AddRemoveSystem struct {
	SystemGroupName string `json:"systemGroupName"`
	ServerIds       []int  `json:"serverIds"`
	Add             bool   `json:"add"`
}

type DeleteSystemType struct {
	ServerId    int    `json:"sid"`
	CleanupType string `json:"cleanupType"`
}

func getSystemId(sessioncookie, susemgr, hostname string, verbose bool) int {

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL =  %s\n", apiURL)
	}

	/*
	 check if system is registered
	*/
	apiMethodGetSystemId := fmt.Sprintf("%s%s%s", apiURL, "/system/getId?name=", hostname)
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s\n", apiMethodGetSystemId)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodGetSystemId, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request to get hostname, error: %s\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "Error closing response body:", err)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP Request failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading http response: %s\n", err)
		os.Exit(1)
	}

	// Unmarshal the JSON response into the struct
	var rsp ResponseSystemGetId
	err = json.Unmarshal(bodyBytes, &rsp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %s\n", err)
		os.Exit(1)
	}

	// Extract and print all fields
	var foundId int
	for _, r := range rsp.Result {
		foundId = r.Id
	}

	if foundId == 0 {
		fmt.Fprintf(os.Stderr, "Host: %s not found in SUSE Manager on %s\n", hostname, susemgr)
		os.Exit(1)
	}

	return foundId

}

func Login(username, password, susemgr string, verbose bool) string {

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Enter function Login\n")
		fmt.Fprintf(os.Stderr, "DEBUG: ====================\n")
		defer fmt.Fprintf(os.Stderr, "DEBUG: Leave function Login \n")
	}

	var sessioncookie string

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiURL = %s\n", apiURL)
	}

	apiMethod := fmt.Sprintf("%s%s", apiURL, "/auth/login")
	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: apiMethod = %s", apiMethod)
	}

	// Create the authentication request payload
	authPayload := AuthRequest{
		Login:    username,
		Password: password,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling payload: %v\n", err)
		os.Exit(1)
	}

	// Create an HTTP POST request
	req, err := http.NewRequest("POST", apiMethod, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "Error closing response body:", err)
		}
	}()

	// Extract the session cookie from the response headers
	cookies := resp.Cookies()

	for _, cookie := range cookies {
		if verbose {
			fmt.Fprintf(os.Stderr, "DEBUG: Cookie Name: %s, Cookie Value: %s\n", cookie.Name, cookie.Value)
		}
		if cookie.Name == "pxt-session-cookie" && cookie.MaxAge == 3600 {
			sessioncookie = cookie.Value
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Session Cookie = %s\n", sessioncookie)
		// Print the response status
		fmt.Fprintf(os.Stderr, "DEBUG: Response status = %s\n", resp.Status)
	}

	// Handle the response body if needed
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Got error to read from respone body.\n")
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response body =  %s\n", responseBody.String())
	}

	return sessioncookie
}

func AddSystem(sessioncookie, susemgr, hostname, group string, verbose bool) int {

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Enter function AddSystem\n")
		fmt.Fprintf(os.Stderr, "DEBUG: ========================\n")
		defer fmt.Fprintf(os.Stderr, "DEBUG: Leave function AddSystem \n")
	}

	/*
	 add System to Group
	*/
	foundId := getSystemId(sessioncookie, susemgr, hostname, verbose)

	if foundId == 0 {
		fmt.Fprintf(os.Stderr, "Did not find the system in SUSE Manager.\n")
		os.Exit(1)
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
		ServerIds:       []int{foundId},
		Add:             true,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(AddRemoveSystemPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling payload: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("DEBUG: Paylod =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiMethodAddOrRemoveSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
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
		os.Exit(1)
	}

	return resp.StatusCode

}

func DeleteSystem(sessioncookie, susemgr, hostname string, verbose bool) int {

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Enter function DeleteSystem\n")
		fmt.Fprintf(os.Stderr, "DEBUG: ===========================\n")
		defer fmt.Fprintf(os.Stderr, "DEBUG: Leave function DeleteSystem \n")
	}

	/*
	 delete System
	*/

	foundId := getSystemId(sessioncookie, susemgr, hostname, verbose)

	if foundId == 0 {
		fmt.Fprintf(os.Stderr, "Did not find the system in SUSE Manager.\n")
		os.Exit(1)
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
		ServerId:    foundId,
		CleanupType: "FORCE_DELETE",
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(DeleteSystemPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling payload: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("DEBUG: Paylod =  %v\n", string(payloadBytes))
	}

	// Create an HTTP POST request
	req, err := http.NewRequest(http.MethodPost, apiDeleteSystems, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
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
		os.Exit(1)
	}

	return resp.StatusCode

}

func GetVaultSecrets(roleID, secretID, vaultAddress, group string, verbose bool) (map[string]interface{}, error) {

	// Path to the secret
	secretPath := fmt.Sprintf("kv-clab-%s/data/suma", group)
	if verbose {
		fmt.Printf("DEBUG: secretPath = %s\n", secretPath)
	}
	// Initialize Vault client
	config := vault.DefaultConfig()
	config.Address = vaultAddress

	// TODO: Workarround for Test
	// Customize the HTTP transport to ignore certificate errors
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Disables SSL certificate validation
		},
	}
	config.HttpClient.Transport = httpTransport

	// Step 1: Initialize Vault client
	client, err := vault.NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create Vault client: %v", err)
	}

	// Step 2: Authenticate using AppRole
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	// Make the login request
	secret, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error logging in with AppRole: %v", err)
	}

	// Extract the client token from the response
	if secret == nil || secret.Auth == nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: no token returned")
	}
	token := secret.Auth.ClientToken
	if verbose {
		fmt.Printf("DEBUG: Successfully authenticated! Token: %s\n", token)
	}

	// Step 3: Set the client token
	client.SetToken(token)

	// Step 4: Retrieve the secret
	secret, err = client.Logical().Read(secretPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading secret: %v", err)
	}

	// Step 5: Extract and print the secret data
	if secret == nil || secret.Data == nil {
		fmt.Fprintf(os.Stderr, "No secret found at path: %s", secretPath)
	}

	// For KV version 2, secret data is under the "data" field
	secretData, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid secret data format")
	}

	// Print the retrieved key-value pairs
	if verbose {
		fmt.Println("DEBUG: Retrieved secret:")
		for key := range secretData {
			fmt.Printf("DEBUG: %s: *******\n", key)
		}
	}

	// Return the secret data
	return secretData, nil
}
