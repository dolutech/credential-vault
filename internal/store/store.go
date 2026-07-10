// Package store manages the encrypted credential vault file.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"credential-vault/internal/crypto"
)

// VaultVersion is the current vault file format version.
const VaultVersion = 1

// ServerCredentials holds the connection details for a single server.
type ServerCredentials struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
	Description string `json:"description,omitempty"`
}

// VaultData is the in-memory structure that gets encrypted.
type VaultData struct {
	Servers map[string]ServerCredentials `json:"servers"`
}

// VaultFile is the on-disk format: metadata + encrypted payload.
type VaultFile struct {
	Version    int            `json:"version"`
	KDF        crypto.KDFParams `json:"kdf"`
	Ciphertext string         `json:"ciphertext"` // base64 of AES-256-GCM ciphertext
}

// Vault provides read/write access to the encrypted credential store.
type Vault struct {
	path     string
	password string
}

// New creates a new Vault instance pointing to the given file path.
// The password is used to derive the encryption key.
func New(path, password string) *Vault {
	return &Vault{
		path:     path,
		password: password,
	}
}

// DefaultVaultPath returns the default vault location for the current OS:
//   Linux:   $XDG_CONFIG_HOME/credential-vault/vault.json (default: ~/.config/credential-vault/vault.json)
//   macOS:   ~/Library/Application Support/credential-vault/vault.json
//   Windows: %AppData%\credential-vault\vault.json
func DefaultVaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "credential-vault", "vault.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "credential-vault", "vault.json"), nil
	default:
		// Linux, BSD, and other Unix-like systems: respect XDG_CONFIG_HOME
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgConfig, "credential-vault", "vault.json"), nil
	}
}

// Exists checks whether the vault file already exists on disk.
func (v *Vault) Exists() bool {
	_, err := os.Stat(v.path)
	return err == nil
}

// Init creates a new empty vault file with the given master password.
// Fails if the vault already exists.
func (v *Vault) Init() error {
	if v.Exists() {
		return errors.New("vault already exists — use a different path or delete the existing vault")
	}

	if err := os.MkdirAll(filepath.Dir(v.path), 0700); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	return v.save(&VaultData{Servers: make(map[string]ServerCredentials)})
}

// load reads and decrypts the vault file.
func (v *Vault) load() (*VaultData, error) {
	if v.password == "" {
		return nil, errors.New("vault password not set (VAULT_PASSWORD env var is empty)")
	}

	raw, err := os.ReadFile(v.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	var file VaultFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("failed to parse vault file: %w", err)
	}

	if file.Version != VaultVersion {
		return nil, fmt.Errorf("unsupported vault version: %d (expected %d)", file.Version, VaultVersion)
	}

	key, err := crypto.DeriveKey(v.password, file.KDF)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	plaintext, err := crypto.Decrypt(key, file.Ciphertext)
	if err != nil {
		return nil, err
	}

	var data VaultData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted vault data: %w", err)
	}

	if data.Servers == nil {
		data.Servers = make(map[string]ServerCredentials)
	}

	return &data, nil
}

// save encrypts and writes the vault data to disk.
func (v *Vault) save(data *VaultData) error {
	if v.password == "" {
		return errors.New("vault password not set (VAULT_PASSWORD env var is empty)")
	}

	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal vault data: %w", err)
	}

	params := crypto.DefaultKDFParams()
	if params.Salt == "" {
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		params.Salt = salt
	}

	key, err := crypto.DeriveKey(v.password, params)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	ciphertext, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	file := VaultFile{
		Version:    VaultVersion,
		KDF:        params,
		Ciphertext: ciphertext,
	}

	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(v.path), 0700); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	if err := os.WriteFile(v.path, raw, 0600); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	return nil
}

// AddServer adds or updates a server in the vault.
func (v *Vault) AddServer(name string, creds ServerCredentials) error {
	data, err := v.load()
	if err != nil {
		return err
	}
	data.Servers[name] = creds
	return v.save(data)
}

// GetServer retrieves a server's credentials from the vault.
func (v *Vault) GetServer(name string) (*ServerCredentials, error) {
	data, err := v.load()
	if err != nil {
		return nil, err
	}
	creds, ok := data.Servers[name]
	if !ok {
		return nil, fmt.Errorf("server '%s' not found in vault", name)
	}
	return &creds, nil
}

// DeleteServer removes a server from the vault.
func (v *Vault) DeleteServer(name string) error {
	data, err := v.load()
	if err != nil {
		return err
	}
	if _, ok := data.Servers[name]; !ok {
		return fmt.Errorf("server '%s' not found in vault", name)
	}
	delete(data.Servers, name)
	return v.save(data)
}

// ListServers returns the names of all servers in the vault (sorted).
// Does NOT return credentials.
func (v *Vault) ListServers() ([]string, error) {
	data, err := v.load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(data.Servers))
	for name := range data.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// ListServersWithInfo returns server names with their non-sensitive info
// (description only — no host/user/password).
func (v *Vault) ListServersWithInfo() ([]ServerInfo, error) {
	data, err := v.load()
	if err != nil {
		return nil, err
	}
	result := make([]ServerInfo, 0, len(data.Servers))
	for name, creds := range data.Servers {
		result = append(result, ServerInfo{
			Name:        name,
			Description: creds.Description,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// ServerInfo holds non-sensitive server information safe to expose to the LLM.
type ServerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ConnectionInfo holds connection details WITHOUT the password or private key.
// This is safe to return to the LLM so it knows where it's connecting.
type ConnectionInfo struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Description string `json:"description"`
}