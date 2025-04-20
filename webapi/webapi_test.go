package webapi

import (
	"encoding/json"
	_ "encoding/json"
	"fmt"
	"net/http"
	_ "net/http"
	"net/http/httptest"
	_ "net/http/httptest"
	"os"
	"testing"
)

// Patch os.Exit for testing
var osExit = os.Exit

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
		json.NewEncoder(w).Encode(response)
	})
	return httptest.NewServer(handler)
}

func TestgetSystemID(t *testing.T) {
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

func TestgetSystemIP(t *testing.T) {
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
