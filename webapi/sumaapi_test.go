package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Patch os.Exit for testing
//var osExit = os.Exit

// Save original functions to restore after test
var (
	origGetSystemID       = getSystemID
	origGetSystemIP       = getSystemIP
	origIsSystemInNetwork = isSystemInNetwork
)

func TestIsSystemInNetwork(t *testing.T) {
	tests := []struct {
		ip      string
		network string
		want    bool
	}{
		{"192.168.1.10", "192.168.1.0", true},
		{"192.168.2.10", "192.168.1.0", false},
		{"10.0.0.5", "10.0.0.0", true},
		{"10.0.1.5", "10.0.0.0", false},
		{"192.168.1.10", "invalid", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s in %s", tt.ip, tt.network), func(t *testing.T) {
			got := isSystemInNetwork(tt.ip, tt.network)
			if got != tt.want {
				t.Errorf("isSystemInNetwork(%q, %q) = %v, want %v", tt.ip, tt.network, got, tt.want)
			}
		})
	}
}

func setupTestServer(t *testing.T, path string, response interface{}) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	})
	return httptest.NewServer(handler)
}

func TestGetSystemID(t *testing.T) {
	resp := ResponseSystemGetId{
		Success: true,
		Result:  []ResultSystemGetId{{Id: 42, Name: "testhost"}},
	}
	server := setupTestServer(t, "/rhn/manager/api/system/getId", resp)
	defer server.Close()

	oldExit := osExit
	defer func() { osExit = oldExit }()
	osExit = func(code int) { panic(fmt.Sprintf("os.Exit(%d)", code)) }

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getSystemID panicked: %v", r)
		}
	}()

	id := getSystemID("dummy-cookie", server.URL, "testhost", false)
	if id != 42 {
		t.Errorf("Expected id 42, got %d", id)
	}
}

func TestGetSystemIP(t *testing.T) {
	resp := ResponseSystemGetIp{
		Success: true,
		Result:  ResultSystemGetIp{Ip: "192.168.1.10", Name: "testhost"},
	}
	server := setupTestServer(t, "/rhn/manager/api/system/getNetwork", resp)
	defer server.Close()

	oldExit := osExit
	defer func() { osExit = oldExit }()
	osExit = func(code int) { panic(fmt.Sprintf("os.Exit(%d)", code)) }

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getSystemIP panicked: %v", r)
		}
	}()

	ip := getSystemIP("dummy-cookie", server.URL, 42, false)
	if ip != "192.168.1.10" {
		t.Errorf("Expected ip 192.168.1.10, got %s", ip)
	}
}

func TestLogin(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/rhn/manager/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "pxt-session-cookie",
			Value:  "test-session",
			MaxAge: 3600,
		})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	oldExit := osExit
	defer func() { osExit = oldExit }()
	osExit = func(code int) { panic(fmt.Sprintf("os.Exit(%d)", code)) }

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Login panicked: %v", r)
		}
	}()

	cookie := Login("user", "pass", server.URL, false)
	if cookie != "test-session" {
		t.Errorf("Expected session cookie 'test-session', got %s", cookie)
	}
}

func TestAddSystem_Success(t *testing.T) {
	// Mock getSystemID to return a fixed ID
	getSystemID = func(sessioncookie, susemgr, hostname string, verbose bool) int {
		return 42
	}

	// Mock getSystemIP to return a fixed IP
	getSystemIP = func(sessioncookie, susemgr string, id int, verbose bool) string {
		return "192.168.1.100"
	}

	// Mock isSystemInNetwork to always return true
	isSystemInNetwork = func(pip, pnetwork string) bool {
		return true
	}

	// Restore original functions
	defer func() {
		getSystemID = origGetSystemID
		getSystemIP = origGetSystemIP
		isSystemInNetwork = origIsSystemInNetwork
	}()

	// Mock SUSE Manager API endpoint for addOrRemoveSystems
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rhn/manager/api/systemgroup/addOrRemoveSystems" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"success": true}`)
			return
		}
		t.Errorf("Unexpected URL: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Silence stderr to avoid cluttering test output
	oldStderr := os.Stderr
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	status := AddSystem("dummy-cookie", server.URL, "testhost", "testgroup", "192.168.1.0", false)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	if status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

}

func TestAddSystem_SystemNotInNetwork(t *testing.T) {
	getSystemID = func(sessioncookie, susemgr, hostname string, verbose bool) int {
		return 42
	}
	getSystemIP = func(sessioncookie, susemgr string, id int, verbose bool) string {
		return "10.0.0.1"
	}
	isSystemInNetwork = func(pip, pnetwork string) bool {
		return false
	}

	// Silence stderr and capture os.Exit
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected os.Exit to be called")
		}
		w.Close()
		os.Stderr = oldStderr
		getSystemID = origGetSystemID
		getSystemIP = origGetSystemIP
		isSystemInNetwork = origIsSystemInNetwork
	}()

	// Override os.Exit to panic for test
	//oldExit := osExit
	osExit = func(code int) { panic("os.Exit called") }
	defer func() { osExit = os.Exit }()

	AddSystem("dummy-cookie", "http://dummy", "testhost", "testgroup", "192.168.1.0", false)
}

// Patch os.Exit in AddSystem and helpers
func init() {
	// Patch all os.Exit calls in this package to use osExit
	// This requires replacing all os.Exit(1) with osExit(1) in the code under test.
}
