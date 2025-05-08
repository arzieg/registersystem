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
	valid := checkFlag("token", "group", "grouppassword", "sumauser", "sumapassword", "http://susemgr", "http://vault", "add")

	if !valid {
		t.Error("Expected valid flags to pass checkFlag")
	}

	invalids := []struct {
		ptoken, pgroup, pgrouppassword, psumauser, psumapassword, psusemgr, pvault, ptask string
	}{
		{"", "group", "grouppassword", "sumauser", "sumapassword", "http://susemgr", "http://vault", "add"},
		{"token", "", "grouppassword", "sumauser", "sumapassword", "http://susemgr", "http://vault", "add"},
		{"token", "group", "", "sumauser", "sumapassword", "http://susemgr", "http://vault", "add"},
		{"token", "group", "grouppassword", "", "sumapassword", "http://susemgr", "http://vault", "add"},
		{"token", "group", "grouppassword", "sumauser", "", "http://susemgr", "http://vault", "add"},
		{"token", "group", "grouppassword", "sumauser", "sumapassword", "", "http://vault", "add"},
		{"token", "group", "grouppassword", "sumauser", "sumapassword", "http://susemgr", "", "add"},
		{"token", "group", "grouppassword", "sumauser", "sumapassword", "http://susemgr", "http://vault", ""},
	}

	for i, inv := range invalids {
		if checkFlag(inv.ptoken, inv.pgroup, inv.pgrouppassword, inv.psumauser, inv.psumapassword, inv.psusemgr, inv.pvault, inv.ptask) {
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
		"-o", "token",
		"-g", "group",
		"-d", "grouppassword",
		"-u", "sumauser",
		"-p", "sumapassword",
		"-m", "http://susemgr",
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
	if token != "token" {
		t.Errorf("Expected token to be 'token', got %q", token)
	}
	if group != "group" {
		t.Errorf("Expected group to be 'group', got %q", group)
	}
	if grouppassword != "grouppassword" {
		t.Errorf("Expected grouppassword to be 'grouppassword', got %q", grouppassword)
	}
	if sumauser != "sumauser" {
		t.Errorf("Expected sumauser to be 'sumauser', got %q", sumauser)
	}
	if sumapassword != "sumapassword" {
		t.Errorf("Expected sumapassword to be 'sumapassword', got %q", sumapassword)
	}

	if susemgr != "http://susemgr" {
		t.Errorf("Expected susemgr to be 'http://susemgr', got %q", susemgr)
	}
	if vaultAddress != "http://vault" {
		t.Errorf("Expected vaultAddress to be 'http://vault', got %q", vaultAddress)
	}
	if task != "add" {
		t.Errorf("Expected task to be 'add', got %q", task)
	}
}
