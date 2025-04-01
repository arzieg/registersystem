package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	// commandline flags
	verbose  bool
	user     string
	password string
	group    string
	hostname string
	susemgr  string
	task     string
)

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.StringVar(&user, "u", "", "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&group, "g", "", "SUSE Manager Group")
	flag.StringVar(&hostname, "h", "", "Hostname")
	flag.StringVar(&susemgr, "s", "", "URL SUSE-Manager")
	flag.StringVar(&task, "t", "", "task [add | delete]")

}

func isFQDN(hostname string) bool {
	// Check if hostname is an FQDN
	return strings.Contains(hostname, ".") &&
		!strings.HasSuffix(hostname, ".")
}

func isEmpty(line string) bool {
	return (line == "")
}

func getTask(line string) string {
	switch strings.ToLower(line) {
	case "add", "a":
		return "add"
	case "delete", "d":
		return "delete"
	default:
		return "error"
	}
}

func checkFlag(puser, ppassword, pgroup, phostname, psusemgr, ptask string) bool {

	if !isFQDN(phostname) || isEmpty(phostname) {
		fmt.Fprintf(os.Stderr, "Please enter the FQDN Hostname.")
		return false
	}

	if isEmpty(puser) {
		fmt.Fprintf(os.Stderr, "Please enter a username.")
		return false
	}

	if isEmpty(ppassword) {
		fmt.Fprintf(os.Stderr, "Please enter a password.")
		return false
	}

	if isEmpty(pgroup) {
		fmt.Fprintf(os.Stderr, "Please enter a SUSE Manager group.")
		return false
	}

	if isEmpty(psusemgr) {
		fmt.Fprintf(os.Stderr, "Please enter the URL of the SUSE Manager.")
		return false
	}

	if isEmpty(ptask) {
		fmt.Fprintf(os.Stderr, "Please enter a task.")
		return false
	}

	return true
}

func main() {
	flag.Parse()

	if !checkFlag(user, password, group, hostname, susemgr, task) {
		os.Exit(1)
	}

	task = getTask(task)
	if task == "error" {
		fmt.Fprintf(os.Stderr, "Please enter a valid task [add | delete].\n")
		os.Exit(1)
	}

	fmt.Println("verbose:", verbose)
	fmt.Println("user:", user)
	fmt.Println("password:", password)
	fmt.Println("group:", group)
	fmt.Println("hostname:", hostname)
	fmt.Println("susemgr:", susemgr)
	fmt.Println("task:", task)

	// Define the API endpoint
	apiURL := fmt.Sprintf("%s%s", susemgr, "/rhn/manager/api")
	fmt.Println("apiURL:", apiURL)

	apiMethod := fmt.Sprintf("%s%s", apiURL, "/auth/login")
	fmt.Println("apiMethod:", apiMethod)

	// JSON payload
	payload := fmt.Sprintf(`{"login": "%s", "password": "%s"}`, user, password)
	fmt.Println("payload:", payload)

	bodyReader := bytes.NewReader([]byte(payload))

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, apiMethod, bodyReader)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Set the required header
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client
	client := &http.Client{}

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making the request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// Print the output
	fmt.Printf("HTTP Response Code: %d\n", resp.StatusCode)
	fmt.Printf("HTTP Response Body: %s\n", string(body))

	if resp.StatusCode == 200 {
		// Retrieve the pxt-session-cookie
		cookieCounter := 0
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "pxt-session-cookie" {
				cookieCounter++
				if cookieCounter == 2 { // Check if it's the second cookie
					fmt.Printf("Second pxt-session-cookie: %s\n", cookie.Value)
					return
				}
			}
		}

		fmt.Println("Second pxt-session-cookie not found in the response.")
	}
}
