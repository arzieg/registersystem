package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AddRemoveSystem struct {
	systemGroupName string `json:"systemGroupName"`
	serverIDs       string `json:"serverIDs"`
	add             string `json:"add"`
}

func Login(username, password, susemgr string) string {

	var sessioncookie string

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	fmt.Println("apiURL:", apiURL)

	apiMethod := fmt.Sprintf("%s%s", apiURL, "/auth/login")
	fmt.Println("apiMethod:", apiMethod)

	// Create the authentication request payload
	authPayload := AuthRequest{
		Login:    username,
		Password: password,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		fmt.Printf("Error marshalling payload: %v\n", err)
		os.Exit(1)
	}

	// Create an HTTP POST request
	req, err := http.NewRequest("POST", apiMethod, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Extract the session cookie from the response headers
	cookies := resp.Cookies()

	for _, cookie := range cookies {
		fmt.Printf("Cookie Name: %s, Cookie Value: %s\n", cookie.Name, cookie.Value)
		if cookie.Name == "pxt-session-cookie" && cookie.MaxAge == 3600 {
			sessioncookie = cookie.Value
		}
	}

	fmt.Printf("Session Cookie: %s\n", sessioncookie)
	// Print the response status
	fmt.Printf("Response status: %s\n", resp.Status)

	// Handle the response body if needed
	var responseBody bytes.Buffer
	responseBody.ReadFrom(resp.Body)
	fmt.Printf("Response body: %s\n", responseBody.String())

	return sessioncookie
}

func AddSystem(sessioncookie, susemgr, hostname, group string) (string, error) {

	fmt.Printf("Session Cookie in Addsystem %s\n", sessioncookie)
	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	fmt.Println("apiURL:", apiURL)

	apiMethodGetSystemId := fmt.Sprintf("%s%s%s", apiURL, "/system/getId?name=", hostname)
	fmt.Println("apiMethod:", apiMethodGetSystemId)

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, apiMethodGetSystemId, nil)
	if err != nil {
		fmt.Printf("Error creating request: %s\n", err)
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
		fmt.Printf("Error sending request: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %s\n", err)
		os.Exit(1)
	}

	// Print the response
	fmt.Printf("Response:\n%s\n", string(bodyBytes))

	return string(bodyBytes), nil

}

func DeleteSystem(sessioncookie, susemgr, hostname, group string) (string, error) {

	return "200", nil
}
