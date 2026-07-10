package integtest

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPServerIntegration tests the full MCP server lifecycle:
// 1. Connects to credential-vault serve as a client (like opencode would)
// 2. Lists tools and verifies 4 tools are registered
// 3. Calls list_servers and verifies servers are returned without credentials
// 4. Calls get_connection_info and verifies password is NOT leaked
func TestMCPServerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Build path to the credential-vault binary (built before running tests)
	binaryPath := "/home/lucascatao/Documentos/Projetos/Tools Dolutech/credential-vault"

	cmd := exec.Command(binaryPath, "serve")
	cmd.Env = []string{
		"VAULT_PATH=/tmp/opencode/test-vault.json",
		"VAULT_PASSWORD=test-secret-pass",
		"HOME=" + "/tmp/opencode",
	}

	transport := &mcp.CommandTransport{Command: cmd}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.3.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	// Test 1: List tools — expect 4 tools
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(tools.Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(tools.Tools))
	}
	expectedTools := map[string]bool{
		"list_servers":         false,
		"get_connection_info":  false,
		"deploy":               false,
		"ssh_exec":             false,
	}
	for _, tool := range tools.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}
	for name, found := range expectedTools {
		if !found {
			t.Errorf("tool '%s' not found", name)
		}
	}
	t.Logf("✓ 4 tools registered correctly")

	// Test 2: Call list_servers
	listResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_servers",
	})
	if err != nil {
		t.Fatalf("list_servers call failed: %v", err)
	}
	listOutput := contentToText(listResult.Content)
	if !strings.Contains(listOutput, "Server PROD 1") {
		t.Errorf("list_servers output missing 'Server PROD 1': %s", listOutput)
	}
	// Verify no credentials leaked
	if strings.Contains(listOutput, "my-secret-password") {
		t.Fatal("SECURITY: password leaked in list_servers output!")
	}
	t.Logf("✓ list_servers works, no credentials leaked")

	// Test 3: Call get_connection_info
	infoResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_connection_info",
		Arguments: map[string]any{
			"server_name": "Server PROD 1",
		},
	})
	if err != nil {
		t.Fatalf("get_connection_info call failed: %v", err)
	}
	infoOutput := contentToText(infoResult.Content)
	// Should contain host, port, user
	if !strings.Contains(infoOutput, "192.168.1.10") {
		t.Errorf("get_connection_info missing host: %s", infoOutput)
	}
	if !strings.Contains(infoOutput, "root") {
		t.Errorf("get_connection_info missing user: %s", infoOutput)
	}
	// MUST NOT contain password or private key
	if strings.Contains(infoOutput, "my-secret-password") {
		t.Fatal("SECURITY: password leaked in get_connection_info output!")
	}
	t.Logf("✓ get_connection_info works, host/user shown but NO password leaked")
}

func contentToText(content []mcp.Content) string {
	var sb strings.Builder
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}