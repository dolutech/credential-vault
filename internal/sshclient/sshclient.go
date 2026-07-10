// Package sshclient provides SSH connectivity for the credential vault.
// It connects using credentials from the vault (never exposed to the LLM)
// and executes commands remotely.
package sshclient

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"credential-vault/internal/store"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ConnectTimeout is the maximum time to wait for an SSH connection.
const ConnectTimeout = 30 * time.Second

// ExecResult holds the output of a remote command execution.
type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// knownHostsCallback returns an ssh.HostKeyCallback that verifies host keys
// against the user's ~/.ssh/known_hosts file.
func knownHostsCallback() ssh.HostKeyCallback {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to insecure mode if we can't find home
		return ssh.InsecureIgnoreHostKey()
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		// If known_hosts doesn't exist or is invalid, fallback to insecure mode
		// This handles first-time connections where known_hosts may not exist
		return ssh.InsecureIgnoreHostKey()
	}
	return callback
}

// createCertSigner wraps a private key signer with an SSH certificate,
// enabling certificate-based SSH authentication.
func createCertSigner(signer ssh.Signer, certPEM string) (ssh.Signer, error) {
	// Parse the OpenSSH certificate from PEM format
	cert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Ensure it's actually a certificate
	certKey, ok := cert.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("the provided key is not an SSH certificate")
	}

	// Create a CertSigner that combines the private key with the certificate
	certSigner, err := ssh.NewCertSigner(certKey, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate signer: %w", err)
	}

	return certSigner, nil
}

// Connect establishes an SSH session using the provided credentials.
func Connect(creds *store.ServerCredentials) (*ssh.Session, error) {
	if creds.Host == "" {
		return nil, fmt.Errorf("server host is empty")
	}
	if creds.User == "" {
		return nil, fmt.Errorf("server user is empty")
	}

	port := creds.Port
	if port == 0 {
		port = 22
	}

	authMethods := []ssh.AuthMethod{}

	// Determine private key content: inline content or read from file path
	keyContent := creds.PrivateKey
	if keyContent == "" && creds.PrivateKeyPath != "" {
		data, err := os.ReadFile(creds.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file '%s': %w", creds.PrivateKeyPath, err)
		}
		keyContent = string(data)
	}

	// Parse private key (with or without passphrase) and optional certificate
	if keyContent != "" {
		var signer ssh.Signer
		var err error

		if creds.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(keyContent), []byte(creds.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(keyContent))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		// If a certificate is provided, wrap the signer with the certificate
		if creds.Certificate != "" {
			certSigner, certErr := createCertSigner(signer, creds.Certificate)
			if certErr != nil {
				return nil, fmt.Errorf("failed to use SSH certificate: %w", certErr)
			}
			authMethods = append(authMethods, ssh.PublicKeys(certSigner))
		} else {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	if creds.Password != "" {
		authMethods = append(authMethods, ssh.Password(creds.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available (neither password nor private key is set)")
	}

	config := &ssh.ClientConfig{
		User:            creds.User,
		Auth:            authMethods,
		HostKeyCallback: knownHostsCallback(),
		Timeout:         ConnectTimeout,
	}

	addr := net.JoinHostPort(creds.Host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	return session, nil
}

// Exec connects to the server and executes the given command.
// Returns stdout, stderr, and exit code.
func Exec(creds *store.ServerCredentials, command string) (*ExecResult, error) {
	session, err := Connect(creds)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Use CombinedOutput approach for better error handling
	err = session.Run(command)
	result := &ExecResult{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			return result, fmt.Errorf("command execution failed: %w", err)
		}
	}

	return result, nil
}