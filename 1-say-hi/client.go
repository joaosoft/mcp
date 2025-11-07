package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	// Reads the user's name from the terminal
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove \n ou \r\n
	name = name[:len(name)-1]

	// Cria cliente MCP
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-client",
		Version: "v1.0.0",
	}, nil)

	// this is to connect over binary
	//transport := &mcp.CommandTransport{Command: exec.Command("myserver")}

	// Connects to the MCP server via exec.Command (will execute server.go)
	transport := &mcp.CommandTransport{Command: exec.Command("go", "run", "server.go")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Calls the "greet" tool with the user's name
	params := &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": name},
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		log.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		log.Fatal("Tool returned an error")
	}

	// Displays the server's response
	for _, c := range res.Content {
		log.Printf("Server response: %s", c.(*mcp.TextContent).Text)
	}
}
