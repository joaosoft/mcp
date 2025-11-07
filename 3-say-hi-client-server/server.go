package main

import (
	"context"
	"log"
	"net"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Estruturas do input/output
type Input struct {
	Name string `json:"name"`
}
type Output struct {
	Greeting string `json:"greeting"`
}

// Tool "greet"
func SayHi(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	return nil, Output{Greeting: "Hi " + input.Name}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "greeter-tcp",
		Version: "v1.0.0",
	}, nil)

	// Adiciona tool "greet"
	mcp.AddTool(server, &mcp.Tool{
		Name:        "greet",
		Description: "Say hi",
	}, SayHi)

	// TCP listener
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Erro ao iniciar TCP: %v", err)
	}
	log.Println("Servidor MCP TCP a correr em :9000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			// Usa o conn como transporte direto para Run()
			if err := server.Run(context.Background(), c); err != nil {
				log.Printf("Erro MCP TCP: %v", err)
			}
		}(conn)
	}
}
