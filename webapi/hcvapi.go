package webapi

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hashicorp/vault/api"
	vault "github.com/hashicorp/vault/api"
)

func GetVaultSecrets(roleID, secretID, vaultAddress, group string, verbose bool) (map[string]interface{}, error) {

	// Path to the secret
	secretPath := fmt.Sprintf("kv-clab-%s/data/suma", group)
	if verbose {
		fmt.Printf("DEBUG: secretPath = %s\n", secretPath)
	}
	// Initialize Vault client
	config := vault.DefaultConfig()
	config.Address = vaultAddress

	// TODO: Workarround for Test
	// Customize the HTTP transport to ignore certificate errors
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Disables SSL certificate validation
		},
	}
	config.HttpClient.Transport = httpTransport

	// Step 1: Initialize Vault client
	client, err := vault.NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create Vault client: %v", err)
	}

	// Step 2: Authenticate using AppRole
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	// Make the login request
	secret, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error logging in with AppRole: %v", err)
	}

	// Extract the client token from the response
	if secret == nil || secret.Auth == nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: no token returned")
	}
	token := secret.Auth.ClientToken
	if verbose {
		fmt.Printf("DEBUG: Successfully authenticated! Token: %s\n", token)
	}

	// Step 3: Set the client token
	client.SetToken(token)

	// Step 4: Retrieve the secret
	secret, err = client.Logical().Read(secretPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading secret: %v", err)
	}

	// Step 5: Extract and print the secret data
	if secret == nil || secret.Data == nil {
		fmt.Fprintf(os.Stderr, "No secret found at path: %s", secretPath)
	}

	// For KV version 2, secret data is under the "data" field
	secretData, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "Invalid secret data format")
	}

	// Print the retrieved key-value pairs
	if verbose {
		fmt.Println("DEBUG: Retrieved secret:")
		for key := range secretData {
			fmt.Printf("DEBUG: %s: *******\n", key)
		}
	}

	// Return the secret data
	return secretData, nil
}

func VaultLogin(roleID, secretID, vaultAddr string) (*api.Client, error) {
	// Create Vault client
	config := &api.Config{Address: vaultAddr}

	// Disable TLS verification (Insecure mode)
	config.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %v", err)
	}

	// Prepare the AppRole login payload
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	// Authenticate using AppRole
	secret, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Vault: %v", err)
	}

	// Set the token received from authentication
	client.SetToken(secret.Auth.ClientToken)
	log.Println("Successfully authenticated with Vault!")

	return client, nil
}

// LogoutFromVault revokes the current Vault token
func VaultLogout(client *api.Client) error {
	// Get the token to revoke
	token := client.Token()
	if token == "" {
		return fmt.Errorf("no token found, already logged out?")
	}

	// Revoke the token
	_, err := client.Logical().Write("auth/token/revoke-self", nil)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %v", err)
	}

	log.Println("Successfully logged out from Vault!")
	return nil
}
