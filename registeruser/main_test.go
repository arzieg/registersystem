package main

import (
	"flag"
	"os"
	"testing"
)

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
			t.Errorf("isEmpty(%q) = %v; want %v", tt.line, got, tt.want)
		}
	}
}

func TestCheckFlag(t *testing.T) {
	valid := checkFlag("role", "secret", "group", "grouppassword", "127.0.0.0", "http://vault", "add")

	if !valid {
		t.Error("Expected valid flags to pass checkFlag")
	}

	invalids := []struct {
		proleID, psecretID, pgroup, pgrouppassword, pnetwork, pvault, ptask string
	}{
		{"", "secret", "group", "grouppassword", "127.0.0.0", "http://vault", "add"},
		{"role", "", "group", "grouppassword", "127.0.0.0", "http://vault", "add"},
		{"role", "secret", "", "grouppassword", "127.0.0.0", "http://vault", "add"},
		{"role", "secret", "group", "", "127.0.0.0", "http://vault", "add"},
		{"role", "secret", "group", "grouppassword", "", "http://vault", "add"},
		{"role", "secret", "group", "grouppassword", "127.0.0.0", "", "add"},
		{"role", "secret", "group", "grouppassword", "127.0.0.0", "http://vault", ""},
	}

	for i, inv := range invalids {
		if checkFlag(inv.proleID, inv.psecretID, inv.pgroup, inv.pgrouppassword, inv.pnetwork, inv.pvault, inv.ptask) {
			t.Errorf("Expected checkFlag to fail for invalid input set #%d: %+v", i, inv)
		}
	}
}

func TestFlagParsing(t *testing.T) {
	// Save original os.Args and reset after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{
		"cmd",
		"-r", "roleid",
		"-s", "secretid",
		"-g", "group",
		"-d", "grouppassword",
		"-n", "127.0.0.0",
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
	if grouppassword != "grouppassword" {
		t.Errorf("Expected grouppassword to be 'grouppassword', got %q", grouppassword)
	}
	if network != "127.0.0.0" {
		t.Errorf("Expected network to be '127.0.0.0', got %q", network)
	}

	if vaultAddress != "http://vault" {
		t.Errorf("Expected vaultAddress to be 'http://vault', got %q", vaultAddress)
	}
	if task != "add" {
		t.Errorf("Expected task to be 'add', got %q", task)
	}
}
