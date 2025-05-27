package main

/*
 Read vault roleID: vault read auth/approle/role/<my-approle>/role-id
 Create roleID: write -force auth/approle/role/<my-approle>
 Create secretID write -f auth/approle/role/<my-approle>/secret-id
*/

/*
TODO:
  Wenn URL nicht erreichbar, bricht das Programm mit einer Panic ab

  Error logging in with AppRole: Put "http://vault.example.com:8443/v1/auth/approle/login": dial tcp: lookup vault.example.com: no such hostAuthentication failed: no token returnedpanic: runtime error: invalid memory address or nil pointer dereference
  [signal SIGSEGV: segmentation violation code=0x1 addr=0x50 pc=0x6f5ea0]
*/

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"registersystem/webapi"
	"strings"
)

var (
	// commandline flags
	verbose      bool
	roleID       string
	secretID     string
	group        string
	hostname     string
	vaultAddress string
	task         string
)

// func init() {
// 	flag.BoolVar(&verbose, "v", false, "verbose output")
// 	flag.StringVar(&roleID, "r", "", "roleID")
// 	flag.StringVar(&secretID, "s", "", "secretID")
// 	flag.StringVar(&group, "g", "", "SUSE Manager Group")
// 	flag.StringVar(&hostname, "h", "", "Hostname")
// 	flag.StringVar(&susemgr, "m", "", "URL SUSE-Manager")
// 	flag.StringVar(&vaultAddress, "a", "", "URL vault address")
// 	flag.StringVar(&task, "t", "", "task [add | delete]")

// }

func registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&roleID, "r", "", "Role ID")
	fs.StringVar(&secretID, "s", "", "Secret ID")
	fs.StringVar(&group, "g", "", "SUSE Manager Group")
	fs.StringVar(&hostname, "h", "", "Hostname")
	fs.StringVar(&vaultAddress, "a", "", "Vault Address")
	fs.StringVar(&task, "t", "", "Task [add | delete]")
	fs.BoolVar(&verbose, "v", false, "Verbose output")
}

func customUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: -r [roleID] -s [secretID] -a [URL Vault] -h [hostname] -g [Group] -t [add|delete] -v [verbose]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "The program add a system to a SUSE Manager Systemgroup or delete a system from the SUSE Manager.\n\nParameter:\n")

	flag.PrintDefaults()
}

func isFQDN(hostname string) bool {
	// Check if hostname is an FQDN
	return strings.Contains(hostname, ".") &&
		!strings.HasSuffix(hostname, ".")
}

func isURL(line string) bool {
	u, err := url.ParseRequestURI(line)
	//fmt.Printf("Get URI: %s %v\n", u, err)
	if err != nil || u.Host == "" {
		return false
	}
	return true
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

func checkFlag(proleID, psecretID, pgroup, phostname, pvault, ptask string) bool {

	if !isFQDN(phostname) || isEmpty(phostname) {
		log.Printf("Please enter the FQDN Hostname.")
		return false
	}

	if !isURL(pvault) || isEmpty(pvault) {
		log.Printf("Please enter a valid URL for vault.")
		return false
	}

	if isEmpty(proleID) {
		log.Printf("Please enter a roleID.")
		return false
	}

	if isEmpty(psecretID) {
		log.Printf("Please enter a secretID.")
		return false
	}

	if isEmpty(pgroup) {
		log.Printf("Please enter a SUSE Manager group.")
		return false
	}

	if isEmpty(pvault) {
		log.Printf("Please enter the URL of the Hashicorp Vault.")
		return false
	}

	if isEmpty(ptask) {
		log.Printf("Please enter a task.")
		return false
	}

	return true
}

func main() {
	// Replace the default Usage with the custom one
	registerFlags(flag.CommandLine)
	flag.Usage = customUsage
	flag.Parse()

	if verbose {
		fmt.Println("DEBUG MAIN Parameter: verbose: ", verbose)
		fmt.Println("DEBUG MAIN Parameter: roleID:", roleID)
		fmt.Println("DEBUG MAIN Parameter: secretID:", secretID)
		fmt.Println("DEBUG MAIN Parameter: group:", group)
		fmt.Println("DEBUG MAIN Parameter: hostname:", hostname)
		fmt.Println("DEBUG MAIN Parameter: vaultAddress:", vaultAddress)
		fmt.Println("DEBUG MAIN Parameter: task:", task)
	}

	// no args
	if len(os.Args) == 1 {
		customUsage()
		os.Exit(1)
	}

	if !checkFlag(roleID, secretID, group, hostname, vaultAddress, task) {
		os.Exit(1)
	}

	task = getTask(task)
	if task == "error" {
		log.Fatalf("please enter a valid task [add | delete].")
	}

	client, err := webapi.VaultLogin(roleID, secretID, vaultAddress, verbose)
	if err != nil {
		log.Fatalf("error logging in to Vault: %v", err)
	}

	defer webapi.VaultLogout(client, verbose)

	suma, err := webapi.VaultGetSecrets(client, vaultAddress, "dagobah", "suma", verbose)
	if err != nil {
		log.Fatalf("error getting vault secrets: %v", err)
	}

	if suma["login"] == nil || suma["login"] == "" {
		log.Fatalf("error, suma login user not definied. Check value in vault.")
	}

	if suma["password"] == nil || suma["password"] == "" {
		log.Fatalf("error, suma password not definied. Check value in vault.")
	}

	if suma["url"] == nil || suma["url"] == "" {
		log.Fatalf("error, suma url not definied. Check value in vault.")
	}

	sumalogin := fmt.Sprintf("%s", suma["login"])
	sumapassword := fmt.Sprintf("%s", suma["password"])
	sumaurl := fmt.Sprintf("%s", suma["url"])

	secretData, err := webapi.VaultGetSecrets(client, vaultAddress, group, "config", verbose)
	if err != nil {
		log.Fatalf("error retrieving secret: %v", err)
	}

	if secretData["network"] == nil || secretData["network"] == "" {
		log.Fatalf("error, network not definied. Check value in vault.")
	}

	network := fmt.Sprintf("%s", secretData["network"])

	if verbose {
		log.Printf("DEBUG MAIN: network = %s\n", network)
	}

	sessioncookie, err := webapi.SumaLogin(sumalogin, sumapassword, sumaurl, verbose)
	if err != nil {
		log.Fatalf("could not login, errorcode: %v", err)
	}
	if verbose {
		log.Printf("DEBUG MAIN: Session Cookie %s\n", sessioncookie)
	}

	switch task {
	case "add":
		result, err := webapi.SumaAddSystem(sessioncookie, sumaurl, hostname, group, network, verbose)
		if err != nil {
			log.Fatalf("could not add System to Suma. %v", err)
		}
		if result != http.StatusOK {
			fmt.Fprintf(os.Stderr, "an error occured, got http error %d", result)
			os.Exit(1)
		} else {
			fmt.Printf("Add system %s successfully to group %s\n", hostname, group)
			if verbose {
				fmt.Printf("Got result: %d\n", result)
			}
		}
	case "delete":
		result, err := webapi.SumaDeleteSystem(sessioncookie, sumaurl, hostname, network, verbose)
		if err != nil {
			log.Fatalf("Could not delete System from Suma, errorcode: %v", err)
		}
		if result != http.StatusOK {
			log.Fatalf("an error occured, got http error %d", result)
		} else {
			log.Printf("successful delete system %s\n", hostname)
			if verbose {
				log.Printf("got result: %d\n", result)
			}
		}

	}
	os.Exit(0)
}
