package main

/*
 Read vault roleID: vault read auth/approle/role/<my-approle>/role-id
 Create roleID: write -force auth/approle/role/<my-approle>
 Create secretID write -f auth/approle/role/<my-approle>/secret-id
*/

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"registersystem/webapi"
)

var (
	// commandline flags
	verbose      bool
	roleID       string
	secretID     string
	group        string
	hostname     string
	susemgr      string
	vaultAddress string
	task         string
)

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.StringVar(&roleID, "r", "", "roleID")
	flag.StringVar(&secretID, "s", "", "secretID")
	flag.StringVar(&group, "g", "", "SUSE Manager Group")
	flag.StringVar(&hostname, "h", "", "Hostname")
	flag.StringVar(&susemgr, "m", "", "URL SUSE-Manager")
	flag.StringVar(&vaultAddress, "a", "", "URL vault address")
	flag.StringVar(&task, "t", "", "task [add | delete]")

}

func customUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: -r [roleID] -s [secretID] -a [URL Vault] -m [URL SUSE Manager] -h [hostname] -g [Group] -t [add|delete] -v [verbose]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "The program add a system to a SUSE Manager Systemgroup or delete a system from the SUSE Manager.\n\nParameter:\n")

	flag.PrintDefaults()
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

func checkFlag(proleID, psecretID, pgroup, phostname, psusemgr, pvault, ptask string) bool {

	if !isFQDN(phostname) || isEmpty(phostname) {
		fmt.Fprintf(os.Stderr, "Please enter the FQDN Hostname.")
		return false
	}

	if isEmpty(proleID) {
		fmt.Fprintf(os.Stderr, "Please enter a roleID.")
		return false
	}

	if isEmpty(psecretID) {
		fmt.Fprintf(os.Stderr, "Please enter a secretID.")
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

	if isEmpty(pvault) {
		fmt.Fprintf(os.Stderr, "Please enter the URL of the Hashicorp Vault.")
		return false
	}

	if isEmpty(ptask) {
		fmt.Fprintf(os.Stderr, "Please enter a task.")
		return false
	}

	return true
}

func main() {
	// Replace the default Usage with the custom one
	flag.Usage = customUsage
	flag.Parse()

	if verbose {
		fmt.Println("DEBUG: verbose: ", verbose)
		fmt.Println("DEBUG: roleID:", roleID)
		fmt.Println("DEBUG: secretID:", secretID)
		fmt.Println("DEBUG: group:", group)
		fmt.Println("DEBUG: hostname:", hostname)
		fmt.Println("DEBUG: susemgr:", susemgr)
		fmt.Println("DEBUG: vaultAddress:", vaultAddress)
		fmt.Println("DEBUG: task:", task)
	}

	// no args
	if len(os.Args) == 1 {
		customUsage()
		os.Exit(1)
	}

	if !checkFlag(roleID, secretID, group, hostname, susemgr, vaultAddress, task) {
		os.Exit(1)
	}

	task = getTask(task)
	if task == "error" {
		fmt.Fprintf(os.Stderr, "Please enter a valid task [add | delete].\n")
		os.Exit(1)
	}

	secretData, err := webapi.GetVaultSecrets(roleID, secretID, vaultAddress, group, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving secret: %v", err)
	}

	user, ok := secretData["login"].(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Value for 'Login' is not a string\n")
	}
	password, ok := secretData["password"].(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Value for 'Password' is not a string\n")
	}
	network, ok := secretData["network"].(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Value for 'Network' is not a string\n")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: network = %s\n", network)
	}

	sessioncookie := webapi.Login(user, password, susemgr, verbose)
	if verbose {
		_, err := fmt.Fprintf(os.Stdout, "DEBUG: Session Cookie %s\n", sessioncookie)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error writing to stdout:", err)
		}

	}

	switch task {
	case "add":
		result := webapi.AddSystem(sessioncookie, susemgr, hostname, group, network, verbose)
		if result != http.StatusOK {
			fmt.Fprintf(os.Stderr, "An error occured, got http error %d", result)
			os.Exit(1)
		} else {
			fmt.Printf("Successful add system %s to group %s\n", hostname, group)
			fmt.Printf("Got result: %d\n", result)
		}
	case "delete":
		result := webapi.DeleteSystem(sessioncookie, susemgr, hostname, network, verbose)
		if result != http.StatusOK {
			fmt.Fprintf(os.Stderr, "An error occured, got http error %d", result)
			os.Exit(1)
		} else {
			fmt.Printf("Successful delete system %s\n", hostname)
			fmt.Printf("Got result: %d\n", result)
		}

	}

	os.Exit(0)
}
