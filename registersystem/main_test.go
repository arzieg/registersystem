package main

import (
	"flag"
	"os"
	"testing"
)

// Test isFQDN
func TestIsFQDN(t *testing.T) {
	tests := []struct {
		hostname string
		want     bool
	}{
		{"example.com", true},
		{"localhost", false},
		{"example.", false},
		{"", false},
		{"sub.domain.com", true},
	}

	for _, tt := range tests {
		got := isFQDN(tt.hostname)
		if got != tt.want {
			t.Errorf("isFQDN(%q) = %v; want %v", tt.hostname, got, tt.want)
		}
	}
}

// Test isURL
func TestIsURL(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"http://vault.example.com", true},
		{"https://vault.google.com", true},
		{"https://vault.myexample.org:8443", true},
		{"https://susemgr", true},
		{"", false},
		{"vault.com", false},
		{"/my/url", false},
	}

	for _, tt := range tests {
		got := isURL(tt.line)
		if got != tt.want {
			t.Errorf("isURL(%q) = %v; want %v", tt.line, got, tt.want)
		}
	}
}

// Test isEmpty
func TestIsEmpty(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"", true},
		{"a line", false},
	}

	for _, tt := range tests {
		got := isEmpty(tt.line)
		if got != tt.want {
			t.Errorf("isEmpty(%q) = %v; want %v", tt.line, got, tt.want)
		}
	}
}

// Test getTask
func TestGetTask(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"add", "add"},
		{"a", "add"},
		{"delete", "delete"},
		{"d", "delete"},
		{"firefox", "error"},
	}

	for _, tt := range tests {
		got := getTask(tt.line)
		if got != tt.want {
			t.Errorf("getTask(%q) = %v; want %v", tt.line, got, tt.want)
		}
	}
}

// Test checkFlag
func TestCheckFlag(t *testing.T) {
	valid := checkFlag("role", "secret", "group", "host.example.com", "http://vault", "add")
	if !valid {
		t.Error("Expected valid flags to pass checkFlag")
	}

	invalids := []struct {
		proleID, psecretID, pgroup, phostname, pvault, ptask string
	}{
		{"", "secret", "group", "host.example.com", "http://vault", "add"},
		{"role", "", "group", "host.example.com", "http://vault", "add"},
		{"role", "secret", "", "host.example.com", "http://vault", "add"},
		{"role", "secret", "group", "", "http://vault", "add"},
		{"role", "secret", "group", "host.example.com", "", "add"},
		{"role", "secret", "group", "host.example.com", "http://vault", ""},
		{"role", "secret", "group", "notfqdn", "http://vault", "add"},
	}

	for i, inv := range invalids {
		if checkFlag(inv.proleID, inv.psecretID, inv.pgroup, inv.phostname, inv.pvault, inv.ptask) {
			t.Errorf("Expected checkFlag to fail for invalid input set #%d: %+v", i, inv)
		}
	}
}

// Test flag parsing and global variable assignment
func TestFlagParsing(t *testing.T) {
	// Save original os.Args and reset after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Save original global variables and restore after test
	origRoleID := roleID
	origSecretID := secretID
	origGroup := group
	origHostname := hostname
	origVaultAddress := vaultAddress
	origTask := task
	origVerbose := verbose
	defer func() {
		roleID = origRoleID
		secretID = origSecretID
		group = origGroup
		hostname = origHostname
		vaultAddress = origVaultAddress
		task = origTask
		verbose = origVerbose
	}()

	os.Args = []string{
		"cmd",
		"-r", "roleid",
		"-s", "secretid",
		"-g", "group",
		"-h", "host.example.com",
		"-a", "http://vault",
		"-t", "add",
		"-v",
	}

	// Reset flags for test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	registerFlags(flag.CommandLine)

	flag.Parse()

	if !verbose {
		t.Error("Expected verbose to be true")
	}
	if roleID != "roleid" {
		t.Errorf("Expected roleID to be 'roleid', got %q", roleID)
	}
	if secretID != "secretid" {
		t.Errorf("Expected secretID to be 'secretid', got %q", secretID)
	}
	if group != "group" {
		t.Errorf("Expected group to be 'group', got %q", group)
	}
	if hostname != "host.example.com" {
		t.Errorf("Expected hostname to be 'host.example.com', got %q", hostname)
	}
	if vaultAddress != "http://vault" {
		t.Errorf("Expected vaultAddress to be 'http://vault', got %q", vaultAddress)
	}
	if task != "add" {
		t.Errorf("Expected task to be 'add', got %q", task)
	}
}
