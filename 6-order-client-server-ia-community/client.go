package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the STDIO client and start the server as a subprocess
	c, err := client.NewStdioMCPClient(
		"go",               // command
		os.Environ(),       // environment
		"run", "server.go", // arguments
	)
	if err != nil {
		log.Fatalf("Error creating MCP stdio client: %v", err)
	}
	defer c.Close()

	// Initialize the client
	initReq := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "Go MCP StdIO Client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	fmt.Println("Initializing MCP client via STDIO...")
	serverInfo, err := c.Initialize(ctx, initReq)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Optional: ping/health check
	if err = c.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	fmt.Println("Server is alive and responding")

	fmt.Printf("Connected to server: %s (version %s)\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)

	// List available tools
	toolsRes, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Error listing tools: %v", err)
	}
	fmt.Println("Available tools:")
	for _, t := range toolsRes.Tools {
		fmt.Printf("- %s: %s\n", t.Name, t.Description)
	}

	// Call the "calculate" tool
	callReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calculate",
			Arguments: map[string]any{
				"operation": "multiply",
				"x":         6,
				"y":         7,
			},
		},
	}

	// Use a short context for the call
	callCtx, callCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer callCancel()

	callRes, err := c.CallTool(callCtx, callReq)
	if err != nil {
		log.Fatalf("Error executing tool: %v", err)
	}

	if len(callRes.Content) > 0 {
		switch v := callRes.Content[0].(type) {
		case mcp.TextContent:
			fmt.Println("Calculation result:", v.Text)
		default:
			fmt.Println("Calculation result: <unknown type>")
		}
	} else {
		fmt.Println("Calculation result: <empty>")
	}
}
