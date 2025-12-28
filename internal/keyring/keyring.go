package keyring

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the identifier of Bifrost in the OS keyring
	ServiceName = "bifrost"
)

type KeyType string

const (
	//KeyTypeConnection is for connection with passwords
	KeyTypeConnection KeyType = "conn"
	//KeyTypeCredential is for shared credentials
	KeyTypeCredential KeyType = "cred"
)

// Set stores a password in the OS keyring
// keyType: "conn" or "cred"
// id: the UUID of the connection or credential
// password: the password to store
func Set(keyType KeyType, keyId string, password string) error {
	key := buildKey(keyType, keyId)

	err := keyring.Set(ServiceName, key, password)
	if err != nil {
		return fmt.Errorf("failed to store password in keyring: %w", err)
	}

	return nil
}

// Get retrieves a password from the OS keyring
func Get(keyType KeyType, keyId string) (string, error) {
	key := buildKey(keyType, keyId)

	password, err := keyring.Get(ServiceName, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password from keyring: %w", err)
	}

	return password, nil
}

// Delete removes a password from the OS keyring
func Delete(keyType KeyType, keyId string) error {
	key := buildKey(keyType, keyId)

	err := keyring.Delete(ServiceName, key)
	if err != nil {
		return fmt.Errorf("failed to delete password from keyring: %w", err)
	}

	return nil
}

// buildKey creates a key in the format: "conn:{id}" or "cred:{id}"
func buildKey(keyType KeyType, keyId string) string {
	return fmt.Sprintf("%s-%s", keyType, keyId)
}

// SetConnectionPassword stores a connection password
func SetConnectionPassword(connectionID string, password string) error {
	return Set(KeyTypeConnection, connectionID, password)
}

// GetConnectionPassword retrieves a connection password
func GetConnectionPassword(connectionID string) (string, error) {
	return Get(KeyTypeConnection, connectionID)
}

// DeleteConnectionPassword removes a connection password
func DeleteConnectionPassword(connectionID string) error {
	return Delete(KeyTypeConnection, connectionID)
}

// SetCredentialPassword stores a credential password
func SetCredentialPassword(credentialID string, password string) error {
	return Set(KeyTypeCredential, credentialID, password)
}

// GetCredentialPassword retrieves a credential password
func GetCredentialPassword(credentialID string) (string, error) {
	return Get(KeyTypeCredential, credentialID)
}

// DeleteCredentialPassword removes a credential password
func DeleteCredentialPassword(credentialID string) error {
	return Delete(KeyTypeCredential, credentialID)
}
