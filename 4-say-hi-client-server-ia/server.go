package main

import (
	"bufio"
	"context"
	"log"
	"net"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Tool: SayHi ---
type Input struct {
	Name string `json:"name"`
}

type Output struct {
	Greeting string `json:"greeting"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	return nil, Output{Greeting: "Hi " + input.Name}, nil
}

// --- TCPConnection wrapper ---
func NewTCPConnection(conn net.Conn) *TCPConnection {
	return &TCPConnection{conn: conn}
}

type TCPConnection struct {
	conn      net.Conn
	sessionID string
}

func (c *TCPConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	reader := bufio.NewReader(c.conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Decodifica diretamente a linha JSON em uma jsonrpc.Message
	msg, err := jsonrpc.DecodeMessage(line)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (c *TCPConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}

	// Adiciona o \n para garantir que a linha termine corretamente
	data = append(data, '\n')

	_, err = c.conn.Write(data)
	return err
}

func (c *TCPConnection) Close() error {
	return c.conn.Close()
}

func (c *TCPConnection) SessionID() string {
	return c.sessionID
}

// --- TCPTransport wrapper ---
type TCPTransport struct {
	conn net.Conn
}

func NewTCPTransport(conn net.Conn) *TCPTransport {
	return &TCPTransport{conn: conn}
}

func (t *TCPTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	return NewTCPConnection(t.conn), nil
}

// --- Main ---
func main() {
	// Cria o server MCP
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)

	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Erro ao abrir TCP: %v", err)
	}
	defer listener.Close()

	log.Println("Servidor MCP TCP rodando na porta 9000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			transport := NewTCPTransport(c)
			if err := server.Run(context.Background(), transport); err != nil {
				log.Printf("Erro MCP TCP: %v", err)
			}
		}(conn)
	}
}
