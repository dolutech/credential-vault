// Package mcpserver implements the MCP server that exposes credential vault
// tools to opencode. Credentials are NEVER exposed to the LLM — the vault
// connects via SSH internally and returns only command output.
package mcpserver

import (
	"context"
	"fmt"
	"log"

	"credential-vault/internal/sshclient"
	"credential-vault/internal/store"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve starts the MCP server on stdio, using the provided vault.
func Serve(vault *store.Vault) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "credential-vault",
		Version: "0.3.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: true},
		},
	})

	// Tool: list_servers
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_servers",
			Description: "Lists server names and descriptions registered in the vault. No credentials are exposed.",
		},
		handleListServers(vault),
	)

	// Tool: get_connection_info
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_connection_info",
			Description: "Returns safe connection info (host, port, user, description) for a server. Does NOT return the password or private key.",
		},
		handleGetConnectionInfo(vault),
	)

	// Tool: deploy
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "deploy",
			Description: "Connects via SSH to the specified server and executes a deploy command. Credentials are read from the vault and never exposed to the LLM.",
		},
		handleDeploy(vault),
	)

	// Tool: ssh_exec
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "ssh_exec",
			Description: "Executes an arbitrary command via SSH on the specified server. Credentials are read from the vault internally and never exposed.",
		},
		handleSSHExec(vault),
	)

	log.Println("credential-vault MCP server started on stdio")
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

// --- list_servers ---

type listServersInput struct{}

type serverEntry struct {
	Name        string `json:"name"         jsonschema:"the name of the server"`
	Description string `json:"description"  jsonschema:"the description of the server"`
}

type listServersOutput struct {
	Servers []serverEntry `json:"servers" jsonschema:"list of servers (names and descriptions only, no credentials)"`
}

func handleListServers(vault *store.Vault) func(context.Context, *mcp.CallToolRequest, listServersInput) (*mcp.CallToolResult, listServersOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input listServersInput) (*mcp.CallToolResult, listServersOutput, error) {
		servers, err := vault.ListServersWithInfo()
		if err != nil {
			return nil, listServersOutput{}, err
		}

		entries := make([]serverEntry, 0, len(servers))
		for _, s := range servers {
			entries = append(entries, serverEntry{
				Name:        s.Name,
				Description: s.Description,
			})
		}

		return nil, listServersOutput{Servers: entries}, nil
	}
}

// --- get_connection_info ---

type connectionInfoInput struct {
	ServerName string `json:"server_name" jsonschema:"the name of the server to look up"`
}

type connectionInfoOutput struct {
	Name        string `json:"name"         jsonschema:"the server name"`
	Host        string `json:"host"         jsonschema:"the server host/IP"`
	Port        int    `json:"port"         jsonschema:"the SSH port"`
	User        string `json:"user"         jsonschema:"the SSH user"`
	Description string `json:"description"  jsonschema:"the server description"`
}

func handleGetConnectionInfo(vault *store.Vault) func(context.Context, *mcp.CallToolRequest, connectionInfoInput) (*mcp.CallToolResult, connectionInfoOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input connectionInfoInput) (*mcp.CallToolResult, connectionInfoOutput, error) {
		if input.ServerName == "" {
			return nil, connectionInfoOutput{}, fmt.Errorf("server_name é obrigatório")
		}

		creds, err := vault.GetServer(input.ServerName)
		if err != nil {
			return nil, connectionInfoOutput{}, err
		}

		port := creds.Port
		if port == 0 {
			port = 22
		}

		info := connectionInfoOutput{
			Name:        input.ServerName,
			Host:        creds.Host,
			Port:        port,
			User:        creds.User,
			Description: creds.Description,
		}

		// NOTE: We intentionally do NOT return Password or PrivateKey.
		return nil, info, nil
	}
}

// --- deploy ---

type deployInput struct {
	ServerName string `json:"server_name" jsonschema:"the name of the server to deploy to"`
	Command    string `json:"command"      jsonschema:"the deploy command to execute over SSH"`
}

type deployOutput struct {
	ServerName string `json:"server_name"   jsonschema:"the server that was deployed to"`
	Command   string `json:"command"        jsonschema:"the command that was executed"`
	Stdout    string `json:"stdout"         jsonschema:"standard output of the command"`
	Stderr    string `json:"stderr"         jsonschema:"standard error of the command"`
	ExitCode  int    `json:"exit_code"      jsonschema:"exit code of the command"`
}

func handleDeploy(vault *store.Vault) func(context.Context, *mcp.CallToolRequest, deployInput) (*mcp.CallToolResult, deployOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input deployInput) (*mcp.CallToolResult, deployOutput, error) {
		if input.ServerName == "" {
			return nil, deployOutput{}, fmt.Errorf("server_name é obrigatório")
		}
		if input.Command == "" {
			return nil, deployOutput{}, fmt.Errorf("command é obrigatório")
		}

		creds, err := vault.GetServer(input.ServerName)
		if err != nil {
			return nil, deployOutput{}, err
		}

		result, err := sshclient.Exec(creds, input.Command)
		if err != nil {
			return nil, deployOutput{}, fmt.Errorf("falha ao executar deploy em '%s': %w", input.ServerName, err)
		}

		return nil, deployOutput{
			ServerName: input.ServerName,
			Command:   input.Command,
			Stdout:    result.Stdout,
			Stderr:    result.Stderr,
			ExitCode:  result.ExitCode,
		}, nil
	}
}

// --- ssh_exec ---

type sshExecInput struct {
	ServerName string `json:"server_name" jsonschema:"the name of the server"`
	Command    string `json:"command"      jsonschema:"the command to execute via SSH"`
}

type sshExecOutput struct {
	ServerName string `json:"server_name"   jsonschema:"the server name"`
	Stdout    string `json:"stdout"         jsonschema:"standard output of the command"`
	Stderr    string `json:"stderr"         jsonschema:"standard error of the command"`
	ExitCode  int    `json:"exit_code"      jsonschema:"exit code of the command"`
}

func handleSSHExec(vault *store.Vault) func(context.Context, *mcp.CallToolRequest, sshExecInput) (*mcp.CallToolResult, sshExecOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input sshExecInput) (*mcp.CallToolResult, sshExecOutput, error) {
		if input.ServerName == "" {
			return nil, sshExecOutput{}, fmt.Errorf("server_name é obrigatório")
		}
		if input.Command == "" {
			return nil, sshExecOutput{}, fmt.Errorf("command é obrigatório")
		}

		creds, err := vault.GetServer(input.ServerName)
		if err != nil {
			return nil, sshExecOutput{}, err
		}

		result, err := sshclient.Exec(creds, input.Command)
		if err != nil {
			return nil, sshExecOutput{}, fmt.Errorf("falha ao executar comando em '%s': %w", input.ServerName, err)
		}

		return nil, sshExecOutput{
			ServerName: input.ServerName,
			Stdout:    result.Stdout,
			Stderr:    result.Stderr,
			ExitCode:  result.ExitCode,
		}, nil
	}
}