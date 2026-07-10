// Package cli implements the command-line interface for the credential vault.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"credential-vault/internal/mcpserver"
	"credential-vault/internal/store"
)

// Version is the current version of credential-vault.
const Version = "0.3.0"

// Shared bufio reader for all CLI input — avoids multiple buffered readers
// competing for stdin data.
var stdinReader = bufio.NewReader(os.Stdin)

// passwordReader reads a password from the terminal without echoing it.
// Uses golang.org/x/term for cross-platform hidden input (Linux, macOS, Windows).
func passwordReader(prompt string) string {
	fmt.Fprint(os.Stderr, prompt) // Use stderr so it doesn't mix with stdout pipes
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return ""
		}
		return string(pw)
	}
	// Fallback for non-terminal environments (e.g., piped input)
	pw, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(pw)
}

// getVaultPath returns the vault path from VAULT_PATH env var or the default.
func getVaultPath() (string, error) {
	if path := os.Getenv("VAULT_PATH"); path != "" {
		return path, nil
	}
	return store.DefaultVaultPath()
}

// getVaultPassword returns the vault password from VAULT_PASSWORD env var
// or prompts the user interactively.
func getVaultPassword() string {
	if pw := os.Getenv("VAULT_PASSWORD"); pw != "" {
		return pw
	}
	return passwordReader("Master password: ")
}

// getVault creates a Vault instance using env vars or interactive prompts.
func getVault() (*store.Vault, error) {
	path, err := getVaultPath()
	if err != nil {
		return nil, err
	}
	pw := getVaultPassword()
	if pw == "" {
		return nil, fmt.Errorf("master password cannot be empty")
	}
	return store.New(path, pw), nil
}

// promptString asks the user for a string value with a default.
func promptString(prompt, def string) string {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	val, _ := stdinReader.ReadString('\n')
	val = strings.TrimSpace(val)
	if val == "" {
		return def
	}
	return val
}

// promptInt asks the user for an integer value with a default.
func promptInt(prompt string, def int) int {
	defStr := strconv.Itoa(def)
	valStr := promptString(prompt, defStr)
	if valStr == "" {
		return def
	}
	n, err := strconv.Atoi(valStr)
	if err != nil || n <= 0 || n > 65535 {
		return def
	}
	return n
}

// readMultiLine reads multi-line input until EOF.
// Uses the shared stdinReader so it doesn't compete with other readers.
func readMultiLine(prompt string) string {
	fmt.Fprintf(os.Stderr, "%s (Ctrl+D to finish):\n", prompt)
	data, err := io.ReadAll(stdinReader)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Run executes the CLI command based on the arguments.
func Run(args []string) error {
	if len(args) < 1 {
		return printUsage()
	}

	command := args[0]

	switch command {
	case "init":
		return cmdInit()
	case "add":
		return cmdAdd(args[1:])
	case "list":
		return cmdList()
	case "delete":
		return cmdDelete(args[1:])
	case "serve":
		return cmdServe()
	case "help", "-h", "--help":
		return printUsage()
	case "--version", "-v":
		fmt.Fprintf(os.Stdout, "credential-vault v%s\n", Version)
		return nil
	default:
		return fmt.Errorf("unknown command: %s\n%s", command, usageString())
	}
}

func cmdInit() error {
	path, err := getVaultPath()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Initializing vault at: %s\n", path)

	pw := getVaultPassword()
	if pw == "" {
		return fmt.Errorf("master password cannot be empty")
	}

	// Confirm password
	fmt.Fprint(os.Stderr, "Confirm master password: ")
	confirm, _ := stdinReader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if pw != confirm {
		return fmt.Errorf("passwords do not match")
	}

	vault := store.New(path, pw)
	if err := vault.Init(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nVault created successfully at: %s\n", path)
	fmt.Fprintln(os.Stderr, "Use the 'add' command to register servers.")
	return nil
}

func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: credential-vault add <server-name>")
	}
	name := args[0]

	vault, err := getVault()
	if err != nil {
		return err
	}
	if !vault.Exists() {
		return fmt.Errorf("vault not found — run 'credential-vault init' first")
	}

	fmt.Fprintf(os.Stderr, "\nAdding server: %s\n\n", name)

	host := promptString("Host/IP", "")
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	port := promptInt("SSH port", 22)
	user := promptString("User", "root")
	if user == "" {
		user = "root"
	}

	// --- Authentication method selection ---
	fmt.Fprintln(os.Stderr, "\nAuthentication method:")
	fmt.Fprintln(os.Stderr, "  1 - Password")
	fmt.Fprintln(os.Stderr, "  2 - Private key (paste content)")
	fmt.Fprintln(os.Stderr, "  3 - Private key file (path on disk)")
	fmt.Fprintln(os.Stderr, "  4 - Private key + SSH certificate")
	fmt.Fprint(os.Stderr, "Choose [1-4]: ")

	authChoice, _ := stdinReader.ReadString('\n')
	authChoice = strings.TrimSpace(authChoice)
	if authChoice == "" {
		authChoice = "1" // default to password
	}

	var pw string
	var privateKey string
	var privateKeyPath string
	var passphrase string
	var certificate string

	switch authChoice {
	case "1":
		// Password
		fmt.Fprint(os.Stderr, "Password: ")
		pwLine, _ := stdinReader.ReadString('\n')
		pw = strings.TrimSpace(pwLine)
		if pw == "" {
			return fmt.Errorf("password cannot be empty")
		}

	case "2":
		// Private key (paste content)
		privateKey = readMultiLine("Paste private key content")
		if privateKey == "" {
			return fmt.Errorf("private key cannot be empty")
		}
		fmt.Fprint(os.Stderr, "Passphrase for private key (leave empty if none): ")
		ph, _ := stdinReader.ReadString('\n')
		passphrase = strings.TrimSpace(ph)

	case "3":
		// Private key file path
		fmt.Fprint(os.Stderr, "Path to private key file: ")
		keyPath, _ := stdinReader.ReadString('\n')
		keyPath = strings.TrimSpace(keyPath)
		if keyPath == "" {
			return fmt.Errorf("key file path cannot be empty")
		}
		privateKeyPath = keyPath
		fmt.Fprint(os.Stderr, "Passphrase for private key (leave empty if none): ")
		ph, _ := stdinReader.ReadString('\n')
		passphrase = strings.TrimSpace(ph)

	case "4":
		// Private key + SSH certificate
		privateKey = readMultiLine("Paste private key content")
		if privateKey == "" {
			return fmt.Errorf("private key cannot be empty")
		}
		fmt.Fprint(os.Stderr, "Passphrase for private key (leave empty if none): ")
		ph, _ := stdinReader.ReadString('\n')
		passphrase = strings.TrimSpace(ph)
		certificate = readMultiLine("Paste SSH certificate content")
		if certificate == "" {
			return fmt.Errorf("certificate cannot be empty")
		}

	default:
		return fmt.Errorf("invalid choice: %s (use 1-4)", authChoice)
	}

	description := promptString("Description (optional)", "")

	creds := store.ServerCredentials{
		Host:           host,
		Port:           port,
		User:           user,
		Password:       pw,
		PrivateKey:     privateKey,
		PrivateKeyPath: privateKeyPath,
		Passphrase:     passphrase,
		Certificate:    certificate,
		Description:    description,
	}

	if err := vault.AddServer(name, creds); err != nil {
		return fmt.Errorf("failed to save server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nServer '%s' added successfully!\n", name)
	return nil
}

func cmdList() error {
	vault, err := getVault()
	if err != nil {
		return err
	}
	if !vault.Exists() {
		return fmt.Errorf("vault not found — run 'credential-vault init' first")
	}

	servers, err := vault.ListServersWithInfo()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		fmt.Fprintln(os.Stderr, "No servers registered.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Registered servers (%d):\n\n", len(servers))
	for _, s := range servers {
		desc := s.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(os.Stdout, "  - %s: %s\n", s.Name, desc)
	}
	return nil
}

func cmdDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: credential-vault delete <server-name>")
	}
	name := args[0]

	vault, err := getVault()
	if err != nil {
		return err
	}
	if !vault.Exists() {
		return fmt.Errorf("vault not found")
	}

	if err := vault.DeleteServer(name); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Server '%s' removed successfully.\n", name)
	return nil
}

func cmdServe() error {
	path, err := getVaultPath()
	if err != nil {
		return err
	}
	pw := os.Getenv("VAULT_PASSWORD")
	if pw == "" {
		return fmt.Errorf("VAULT_PASSWORD not set — the MCP server requires the password via environment variable")
	}

	vault := store.New(path, pw)
	if !vault.Exists() {
		return fmt.Errorf("vault not found at %s — run 'credential-vault init' first", path)
	}

	return mcpserver.Serve(vault)
}

func usageString() string {
	return `Credential Vault

Usage:
  credential-vault <command> [args]

Commands:
  init                  Initialize a new vault (set master password)
  add <name>            Add or update a server (interactive mode)
  list                  List registered servers (never shows credentials)
  delete <name>         Remove a server from the vault
  serve                 Start the MCP server (stdio) for AI assistants
  help                  Show this help
  --version, -v         Show version

Environment Variables:
  VAULT_PASSWORD        Master password to decrypt the vault (required for 'serve')
  VAULT_PATH            Vault file path (default: OS-specific, see docs)

Examples:
  credential-vault init
  credential-vault add "Server PROD 1"
  credential-vault list
  VAULT_PASSWORD=my-password credential-vault serve`
}

func printUsage() error {
	fmt.Fprintln(os.Stderr, usageString())
	return nil
}