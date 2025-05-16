package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Helper to patch osExit for tests
func patchOsExit(t *testing.T) (called *bool, code *int) {
	called = new(bool)
	code = new(int)
	osExit = func(c int) {
		*called = true
		*code = c
		panic("osExit called")
	}
	t.Cleanup(func() { osExit = os.Exit })
	return
}

func TestIsSystemInNetwork(t *testing.T) {
	tests := []struct {
		ip      string
		network string
		want    bool
	}{
		{"192.168.1.10", "192.168.1.0", true},
		{"192.168.2.10", "192.168.1.0", false},
		{"invalid", "192.168.1.0", false},
	}
	for _, tt := range tests {
		got := isSystemInNetwork(tt.ip, tt.network)
		if got != tt.want {
			t.Errorf("isSystemInNetwork(%q, %q) = %v, want %v", tt.ip, tt.network, got, tt.want)
		}
	}
}

func TestSumaGetSystemID_Success(t *testing.T) {
	// Patch HTTP client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": 42, "name": "testhost"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Patch sumaGetSystemID to use test server
	id, err := sumaGetSystemID("cookie", server.URL, "testhost", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected id 42, got %d", id)
	}
}

func TestSumaGetSystemID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"result":  []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	id, err := sumaGetSystemID("cookie", server.URL, "missinghost", false)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got %v", err)
	}
	if id != -1 {
		t.Errorf("expected id -1, got %d", id)
	}
}

func TestSumaGetSystemIP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"ip":       "10.0.0.1",
				"hostname": "testhost",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ip, err := sumaGetSystemIP("cookie", server.URL, 42, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "10.0.0.1" {
		t.Errorf("expected ip 10.0.0.1, got %s", ip)
	}
}

func TestSumaGetSystemIP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"ip":       "",
				"hostname": "testhost",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ip, err := sumaGetSystemIP("cookie", server.URL, 42, false)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got %v", err)
	}
	if ip != "" {
		t.Errorf("expected empty ip, got %s", ip)
	}
}

func TestSumaLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "pxt-session-cookie",
			Value:  "session123",
			MaxAge: 3600,
		})
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cookie, err := SumaLogin("user", "pass", server.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cookie != "session123" {
		t.Errorf("expected session123, got %s", cookie)
	}
}

func TestSumaAddSystem_InvalidNetwork(t *testing.T) {
	// Patch sumaGetSystemID and sumaGetSystemIP to return valid values
	oldGetSystemID := sumaGetSystemID
	oldGetSystemIP := sumaGetSystemIP
	sumaGetSystemID = func(sessioncookie, susemgr, hostname string, verbose bool) (int, error) {
		return 42, nil
	}
	sumaGetSystemIP = func(sessioncookie, susemgr string, id int, verbose bool) (string, error) {
		return "10.0.0.1", nil
	}
	defer func() {
		sumaGetSystemID = oldGetSystemID
		sumaGetSystemIP = oldGetSystemIP
	}()

	status, err := SumaAddSystem("cookie", "http://dummy", "host", "group", "192.168.1.0", false)
	if err == nil || !strings.Contains(err.Error(), "does not belong to the permitted network") {
		t.Errorf("expected network error, got %v", err)
	}
	if status != -1 {
		t.Errorf("expected status -1, got %d", status)
	}
}

func TestSumaDeleteSystem_InvalidNetwork(t *testing.T) {
	oldGetSystemID := sumaGetSystemID
	oldGetSystemIP := sumaGetSystemIP
	sumaGetSystemID = func(sessioncookie, susemgr, hostname string, verbose bool) (int, error) {
		return 42, nil
	}
	sumaGetSystemIP = func(sessioncookie, susemgr string, id int, verbose bool) (string, error) {
		return "10.0.0.1", nil
	}
	defer func() {
		sumaGetSystemID = oldGetSystemID
		sumaGetSystemIP = oldGetSystemIP
	}()

	status, err := SumaDeleteSystem("cookie", "http://dummy", "host", "192.168.1.0", false)
	if err == nil || !strings.Contains(err.Error(), "does not belong to the permitted network") {
		t.Errorf("expected network error, got %v", err)
	}
	if status != -1 {
		t.Errorf("expected status -1, got %d", status)
	}
}

func TestSumaAddUser_Success(t *testing.T) {
	// Mock SUSE Manager API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a successful user addition response
		resp := map[string]interface{}{
			"success": true,
			"result":  "User added successfully",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	status, err := SumaAddUser("cookie", server.URL, "testuser", "testpass", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != 0 {
		t.Errorf("expected status 0, got %d", status)
	}
}

func TestSumaAddUser_Failure(t *testing.T) {
	// Mock SUSE Manager API server with failure response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": false,
			"error":   "User already exists",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	status, err := SumaAddUser("cookie", server.URL, "existinguser", "testpass", false)
	if err == nil || !strings.Contains(err.Error(), "User already exists") {
		t.Errorf("expected error about existing user, got %v", err)
	}
	if status != -1 {
		t.Errorf("expected status -1, got %d", status)
	}
}

func TestSumaRemoveUser_Success(t *testing.T) {
	// Mock SUSE Manager API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a successful user removal response
		resp := map[string]interface{}{
			"success": true,
			"result":  "User removed successfully",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	err := SumaRemoveUser("cookie", server.URL, "testuser", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestSumaRemoveUser_Failure(t *testing.T) {
	// Mock SUSE Manager API server with failure response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": false,
			"error":   "User does not exist",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	err := SumaRemoveUser("cookie", server.URL, "nonexistentuser", false)
	if err == nil || !strings.Contains(err.Error(), "User does not exist") {
		t.Errorf("expected error about non-existent user, got %v", err)
	}

}

// More tests can be added for SumaAddUser, SumaRemoveUser, sumaRemoveSystemGroup, sumaCheckSystemGroup, etc.
// For brevity, only core exported functions and key error paths are covered here.
