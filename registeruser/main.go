package main

/*
 registeruser:
   create user and role in hcv. create user in suse manager
*/

/*
TODO:
  Aufruf:
   registeruser -o <hcv-root-token> -g user-to-create -d password-to-create -u <sumaloginuser> -p <sumaloginpwd>
                -m <susemgr-addr> -a <hcv-vault-addr> -t <create/delete> -v <verbose>

	output: role-id, secret-id
*/

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

var (
	// commandline flags
	verbose       bool
	token         string
	group         string
	grouppassword string
	sumauser      string
	sumapassword  string
	susemgr       string
	vaultAddress  string
	task          string
)

func registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&token, "o", "", "HCV root token")
	fs.StringVar(&group, "g", "", "SUSE Manager Group")
	fs.StringVar(&grouppassword, "d", "", "SUSE Manager Group Password")
	fs.StringVar(&sumauser, "u", "", "SUSE Manager Admin-User")
	fs.StringVar(&sumapassword, "p", "", "SUSE Manager Admin-Password")
	fs.StringVar(&susemgr, "m", "", "SUSE Manager URL")
	fs.StringVar(&vaultAddress, "a", "", "Vault Address")
	fs.StringVar(&task, "t", "", "Task [add | delete]")
	fs.BoolVar(&verbose, "v", false, "Verbose output")
}

func customUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: -o [HCV token] -a [URL Vault] -m [URL SUSE Manager] -u [SUMA Adminuser] -p [SUMA Adminpassword] -g [SUMA Group] -d [SUMA Grouppassword] -t [add|delete] -v [verbose]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "The program create or delete an user und policy in HCV and create an user in the SUSE Manager .\n\nParameter:\n")

	flag.PrintDefaults()
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

func checkFlag(ptoken, pgroup, pgrouppassword, psumauser, psumapassword, psusemgr, pvault, ptask string) bool {

	if isEmpty(ptoken) {
		fmt.Fprintf(os.Stderr, "Please enter a token.\n")
		return false
	}

	if isEmpty(pgroup) {
		fmt.Fprintf(os.Stderr, "Please enter a group (user) to create.\n")
		return false
	}

	if isEmpty(pgrouppassword) {
		fmt.Fprintf(os.Stderr, "Please enter a password for the group (user).\n")
		return false
	}

	if isEmpty(psumauser) {
		fmt.Fprintf(os.Stderr, "Please enter the SUSE Manager useradmin user\n")
		return false
	}

	if isEmpty(psumapassword) {
		fmt.Fprintf(os.Stderr, "Please enter a SUSE Manager useradmin password\n")
		return false
	}

	if isEmpty(psusemgr) {
		fmt.Fprintf(os.Stderr, "Please enter the URL of the SUSE Manager.\n")
		return false
	}

	if isEmpty(pvault) {
		fmt.Fprintf(os.Stderr, "Please enter the URL of the Hashicorp Vault.\n")
		return false
	}

	if isEmpty(ptask) {
		fmt.Fprintf(os.Stderr, "Please enter a task.\n")
		return false
	}

	if !isURL(pvault) || isEmpty(pvault) {
		fmt.Fprintf(os.Stderr, "Please enter a valid URL for vault.\n")
		return false
	}

	if !isURL(psusemgr) || isEmpty(psusemgr) {
		fmt.Fprintf(os.Stderr, "Please enter a valid URL for SUSE Manager.\n")
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
		fmt.Println("DEBUG: verbose: ", verbose)
		fmt.Println("DEBUG: token:", token)
		fmt.Println("DEBUG: group:", group)
		fmt.Println("DEBUG: grouppassword:", grouppassword)
		fmt.Println("DEBUG: sumauser:", sumauser)
		fmt.Println("DEBUG: sumapassword:", sumapassword)
		fmt.Println("DEBUG: susemgr:", susemgr)
		fmt.Println("DEBUG: vaultAddress:", vaultAddress)
		fmt.Println("DEBUG: task:", task)
	}

	// no args
	if len(os.Args) == 1 {
		customUsage()
		os.Exit(1)
	}

	if !checkFlag(token, group, grouppassword, sumauser, sumapassword, susemgr, vaultAddress, task) {
		os.Exit(1)
	}

	task = getTask(task)
	if task == "error" {
		fmt.Fprintf(os.Stderr, "Please enter a valid task [add | delete].\n")
		os.Exit(1)
	}
	/*
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
	*/
	os.Exit(0)
}
