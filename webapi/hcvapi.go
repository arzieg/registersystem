package webapi

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
)

func GetVaultSecrets(roleID, secretID, vaultAddress, group string, verbose bool) (map[string]interface{}, error) {

	// Path to the secret
	secretPath := fmt.Sprintf("kv-clab-%s/data/suma", group)
	if verbose {
		fmt.Printf("DEBUG HCVAPI: secretPath = %s\n", secretPath)
	}
	// Initialize Vault client
	config := api.DefaultConfig()
	config.Address = vaultAddress

	// TODO: Workarround for Test
	// Customize the HTTP transport to ignore certificate errors
	// httpTransport := &http.Transport{
	// 	TLSClientConfig: &tls.Config{
	// 		InsecureSkipVerify: true, // Disables SSL certificate validation
	// 	},
	// }
	// config.HttpClient.Transport = httpTransport

	// Step 1: Initialize Vault client
	client, err := api.NewClient(config)
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
		log.Printf("DEBUG HCVAPI: Successfully authenticated! Token: %s\n", token)
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
		log.Println("DEBUG HCVAPI: Retrieved secret:")
		for key := range secretData {
			log.Printf("DEBUG HCVAPI: %s: *******\n", key)
		}
	}

	// Return the secret data
	return secretData, nil
}

func VaultLogin(roleID, secretID, vaultAddr string, verbose bool) (*api.Client, error) {
	// Create Vault client
	config := &api.Config{Address: vaultAddr}

	// Disable TLS verification (Insecure mode)
	// config.HttpClient = &http.Client{
	// 	Transport: &http.Transport{
	// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// 	},
	// }

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
		return nil, fmt.Errorf("failed to authenticate to Vault: %v", err)
	}

	// Set the token received from authentication
	client.SetToken(secret.Auth.ClientToken)
	if verbose {
		log.Println("DEBUG HCVAPI: Successful authenticate to Vault!")
	}
	return client, nil
}

// LogoutFromVault revokes the current Vault token
func VaultLogout(client *api.Client, verbose bool) error {
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

	if verbose {
		log.Println("DEBUG HCVAPI: Successful logged out from Vault!")
	}
	return nil
}

func VaultCreatePolicy(client *api.Client, group string, verbose bool) (policyName string, err error) {

	policyName = fmt.Sprintf("%s_read_policy", group)
	policyContent := fmt.Sprintf(
		`path "kv-clab-%s*" {
		capabilities = ["list", "read"]
	}
path "kv-clab-dagobah/suma" {
	capabilities = ["list", "read"]
}
path "sys/policies/acl/%s_read_policy" {
	capabilities = ["read"]
}`, group, group)

	if verbose {
		log.Printf("DEBUG HCVAPI: policyName: %s\n", policyName)
		log.Printf("DEBUG HCVAPI: policyContent:%s\n", policyContent)
	}

	_, err = client.Logical().Write(fmt.Sprintf("sys/policies/acl/%s", policyName), map[string]interface{}{
		"policy": policyContent,
	})
	if err != nil {
		return policyName, fmt.Errorf("failed to create policy: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: Policy created successfully: %s", policyName)
	}

	return policyName, nil
}

func VaultDeletePolicy(client *api.Client, group string, verbose bool) (err error) {

	policyName := fmt.Sprintf("%s_read_policy", group)

	_, err = client.Logical().Delete(fmt.Sprintf("sys/policies/acl/%s", policyName))
	if err != nil {
		return fmt.Errorf("failed to delete policy: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: Policy deleted successfully: %s", policyName)
	}

	return nil
}

func VaultCreateRole(client *api.Client, group, policyName string, verbose bool) (roleID, secretID string, err error) {

	roleData := map[string]interface{}{
		"policies":      []string{policyName},
		"token_ttl":     3600,
		"token_max_ttl": 14400,
	}

	// Write the role to Vault
	rolePath := fmt.Sprintf("auth/approle/role/%s", group)
	_, err = client.Logical().Write(rolePath, roleData)
	if err != nil {
		return roleID, secretID, fmt.Errorf("failed to create role: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: AppRole created successfully: %s", group)
	}

	// Retrieve role ID for authentication
	roleIDPath := fmt.Sprintf("auth/approle/role/%s/role-id", group)
	roleIDSecretResponse, err := client.Logical().Read(roleIDPath)

	if err != nil {
		return roleID, secretID, fmt.Errorf("failed to retrieve role ID: %v", err)
	}

	roleID, ok := roleIDSecretResponse.Data["role_id"].(string)

	if !ok {
		return roleID, secretID, fmt.Errorf("failed to retrieve role ID: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: Got roleID: %s\n", roleID)
	}

	// get secretID
	secretIDPath := fmt.Sprintf("auth/approle/role/%s/secret-id", group)
	secretIDResponse, err := client.Logical().Write(secretIDPath, map[string]interface{}{})

	if err != nil {
		return roleID, secretID, fmt.Errorf("failed to generate secret ID: %v", err)
	}

	secretID, ok = secretIDResponse.Data["secret_id"].(string)

	if !ok {
		return roleID, secretID, fmt.Errorf("unexpected response format for secret ID")
	}

	if verbose {
		log.Println("DEBUG HCVAPI: Got secretID: #########")
	}

	return roleID, secretID, nil
}

func VaultEnableKVv2(client *api.Client, path string, verbose bool) (err error) {

	mountConfig := map[string]interface{}{
		"type": "kv",
		"options": map[string]interface{}{
			"version": "2",
		},
	}

	// Vault API path for enabling secrets engine
	enablePath := fmt.Sprintf("/sys/mounts/%s", path)

	// Check if the KV secrets engine is already enabled
	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list Vault mounts: %v", err)
	}

	// Vault paths always end with "/"
	mountPath := path + "/"

	if _, exists := mounts[mountPath]; exists {
		if verbose {
			log.Printf("DEBUG HCVAPI: KV v2 is already enabled at: %s\n", path)
		}
		return nil
	}

	// Write request to Vault
	_, err = client.Logical().Write(enablePath, mountConfig)
	if err != nil {
		return fmt.Errorf("failed to enable KV v2: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: KV v2 successfully enabled at:%s\n", path)
	}
	return nil
}

func VaultDisableKVv2(client *api.Client, path string, verbose bool) (err error) {

	// Vault API path for disabling secrets engine
	disablePath := fmt.Sprintf("/sys/mounts/%s", path)

	// Check if the KV secrets engine is already enabled
	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list Vault mounts: %v", err)
	}

	// Vault paths always end with "/"
	mountPath := path + "/"

	if _, exists := mounts[mountPath]; !exists {
		if verbose {
			log.Printf("DEBUG HCVAPI: KV v2 is already disabled: %s\n", path)
		}
		return nil
	}

	// Write request to Vault
	_, err = client.Logical().Delete(disablePath)
	if err != nil {
		return fmt.Errorf("failed to disable KV v2: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: KV v2 successful disable path:%s\n", path)
	}
	return nil
}

func VaultUpdateSecret(client *api.Client, path, key, value string, verbose bool) error {
	// Read existing secrets
	secret, err := client.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("failed to read existing secrets: %v", err)
	}

	// Initialize the data structure if there are no existing secrets
	var existingData map[string]interface{}
	if secret != nil && secret.Data != nil {
		existingData, _ = secret.Data["data"].(map[string]interface{})
	} else {
		existingData = make(map[string]interface{})
	}

	// Insert key-value
	existingData[key] = value

	// Write the updated secrets back to Vault
	updatedSecret := map[string]interface{}{
		"data": existingData, // KV v2 requires the data field
	}

	_, err = client.Logical().Write(path, updatedSecret)
	if err != nil {
		return fmt.Errorf("failed to write updated secrets: %v", err)
	}

	if verbose {
		log.Printf("DEBUG HCVAPI: Successful update secret on %s", path)
	}

	return nil
}
