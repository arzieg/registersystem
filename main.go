package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"systemmanager/webapi"
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

	sessioncookie := webapi.Login(user, password, susemgr)
	fmt.Fprintf(os.Stdout, "\nSession Cookie %s\n", sessioncookie)

}
