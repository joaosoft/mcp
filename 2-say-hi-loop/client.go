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

	// Cria cliente MCP
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-client",
		Version: "v1.0.0",
	}, nil)

	// Conecta ao servidor (executando server.go)
	transport := &mcp.CommandTransport{Command: exec.Command("go", "run", "server.go")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Digite nomes (Ctrl+C para sair):")

	for {
		fmt.Print("> ")
		name, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		name = name[:len(name)-1] // Remove \n

		if name == "" {
			continue
		}

		// Chama a tool "greet" com o nome do usuário
		params := &mcp.CallToolParams{
			Name:      "greet",
			Arguments: map[string]any{"name": name},
		}
		res, err := session.CallTool(ctx, params)
		if err != nil {
			log.Printf("CallTool failed: %v", err)
			continue
		}
		if res.IsError {
			log.Println("Tool returned an error")
			continue
		}

		for _, c := range res.Content {
			log.Printf("%s", c.(*mcp.TextContent).Text)
		}
	}
}
