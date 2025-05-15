package main

/*
 registeruser:
   create user and role in hcv. create user in suse manager
*/

/*
  call
   registeruser -r <roleID> -s <secretID> -a <hcv-vault-addr>
   		-d <password-to-create-in-suma> -g <suma-group-to-create>
		-n <network configured for dev environment>
		-t <create/delete entry>
		-v <verbose>

	output: role-id, secret-id

	TODO:
		delete entries

*/

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"registersystem/webapi"
	"strings"
)

var (
	// commandline flags
	verbose       bool
	roleID        string
	secretID      string
	group         string
	grouppassword string
	network       string
	vaultAddress  string
	task          string

	grouproleID   string // roleID of the created User
	groupsecretID string // secretID of the created User
)

const kvprefix string = "kv-clab-"

func registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&roleID, "r", "", "HCV roleID")
	fs.StringVar(&secretID, "s", "", "HCV secretID")
	fs.StringVar(&group, "g", "", "SUSE Manager Group")
	fs.StringVar(&grouppassword, "d", "", "SUSE Manager Group Password")
	fs.StringVar(&network, "n", "", "Network of the Testenvironment f.i. 172.1.22.0")
	fs.StringVar(&vaultAddress, "a", "", "Vault Address")
	fs.StringVar(&task, "t", "", "Task [add | delete]")
	fs.BoolVar(&verbose, "v", false, "Verbose output")
}

func customUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s: -r [roleID] -s [secretID] -a [URL Vault] -g [SUMA Group] -d [SUMA Grouppassword] -n [Network] -t [add|delete] -v [verbose]\n\n", os.Args[0])
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

func isIP(line string) bool {
	i := net.ParseIP(line)
	return i != nil
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

func checkFlag(proleID, psecretID, pgroup, pgrouppassword, pnetwork, pvault, ptask string) bool {

	if isEmpty(proleID) {
		log.Println("Please enter a roleID.")
		return false
	}

	if isEmpty(psecretID) {
		log.Println("Please enter a secretID.")
		return false
	}

	if isEmpty(pgroup) {
		log.Println("Please enter a group (user) to create.")
		return false
	}

	if isEmpty(pgrouppassword) {
		log.Println("Please enter a password for the group (user) in SUSE Manager.")
		return false
	}

	if isEmpty(pvault) {
		log.Println("Please enter the URL of the Hashicorp Vault.")
		return false
	}

	if isEmpty(ptask) {
		log.Println("Please enter a task.")
		return false
	}

	if !isURL(pvault) || isEmpty(pvault) {
		log.Println("Please enter a valid URL for vault.")
		return false
	}

	if !isIP(pnetwork) || isEmpty(pnetwork) {
		log.Println("Please enter a valid IP for the network.")
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
		log.Println("DEBUG MAIN: verbose: ", verbose)
		log.Println("DEBUG MAIN: roleID:", roleID)
		log.Println("DEBUG MAIN: secretID:", secretID)
		log.Println("DEBUG MAIN: group:", group)
		log.Println("DEBUG MAIN: grouppassword:", grouppassword)
		log.Println("DEBUG MAIN: network:", network)
		log.Println("DEBUG MAIN: vaultAddress:", vaultAddress)
		log.Println("DEBUG MAIN: task:", task)
	}

	// no args
	if len(os.Args) == 1 {
		customUsage()
		os.Exit(1)
	}

	if !checkFlag(roleID, secretID, group, grouppassword, network, vaultAddress, task) {
		os.Exit(1)
	}

	task = getTask(task)
	if task == "error" {
		log.Fatalf("please enter a valid task [add | delete].\n")
	}

	client, err := webapi.VaultLogin(roleID, secretID, vaultAddress, verbose)
	if err != nil {
		log.Fatalf("error logging in to Vault: %v", err)
	}

	defer webapi.VaultLogout(client, verbose)

	suma, err := webapi.GetVaultSecrets(client, vaultAddress, "dagobah", verbose)
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

	/* TODO
	Add user to suma
	*/

	switch task {
	case "add":
		{
			policyName, err := webapi.VaultCreatePolicy(client, group, verbose)
			if err != nil {
				log.Fatalf("error create policy: %v", err)
			}

			if verbose {
				log.Printf("DEBUG MAIN: policyName = %s\n", policyName)
				log.Printf("DEBUG MAIN: client =  %v\n", client)
			}

			grouproleID, groupsecretID, err = webapi.VaultCreateRole(client, group, policyName, verbose)
			if err != nil {
				log.Fatalf("error create role: %v", err)
			}

			// enable KV
			path := fmt.Sprintf("%s%s", kvprefix, group)
			err = webapi.VaultEnableKVv2(client, path, verbose)
			if err != nil {
				log.Fatalf("error enabling kv, got: %v ", err)
			}

			// write AppRole Output to KV
			path = fmt.Sprintf("%s%s/data/approle_output", kvprefix, group)
			err = webapi.VaultUpdateSecret(client, path, "role_id", roleID, verbose)
			if err != nil {
				log.Fatalf("error writing secret to vault: %v", err)
			}

			err = webapi.VaultUpdateSecret(client, path, "secret_id", secretID, verbose)
			if err != nil {
				log.Fatalf("error writing secret to vault: %v", err)
			}

			// write Network to KV
			path = fmt.Sprintf("%s%s/data/config", kvprefix, group)
			err = webapi.VaultUpdateSecret(client, path, "network", network, verbose)
			if err != nil {
				log.Fatalf("error writing secret to vault: %v", err)
			}

			fmt.Fprintf(os.Stdout, "API Login-Information for User: %s\nroleID=%s\nsecretID=%s\n", group, grouproleID, groupsecretID)

		}
	case "delete":
		{
			err := webapi.VaultDeletePolicy(client, group, verbose)
			if err != nil {
				log.Fatalf("error deleting policy: %v", err)
			}

			// disable KV
			path := fmt.Sprintf("%s%s", kvprefix, group)
			err = webapi.VaultDisableKVv2(client, path, verbose)
			if err != nil {
				log.Fatalf("error disable kv, got: %v ", err)
			}

			fmt.Fprintf(os.Stdout, "Successful remove policy and kv-vault\n")
		}
	}
	/* logout */
	// err = webapi.VaultLogout(client, verbose)
	// if err != nil {
	// 	log.Fatalf("Error logout from Vault: %v", err)
	// }

	os.Exit(0)
}
