package main

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CheckOrderStatusInput struct {
	IdOrder string `json:"idOrder"`
}

type CheckOrderStatusOutput struct {
	Status string `json:"status"`
}

func CheckOrderStatus(ctx context.Context, req *mcp.CallToolRequest, input CheckOrderStatusInput) (*mcp.CallToolResult, CheckOrderStatusOutput, error) {
	fmt.Printf("calling the method for getting the order %s status", input.IdOrder)
	status := "new-" + input.IdOrder
	return nil, CheckOrderStatusOutput{Status: status}, nil
}

type GetOrderInput struct {
	IdOrder string `json:"idOrder"`
}

type GetOrderOutput struct {
	Order string `json:"order"`
}

func GetOrder(ctx context.Context, req *mcp.CallToolRequest, input GetOrderInput) (*mcp.CallToolResult, GetOrderOutput, error) {
	fmt.Printf("calling the method for getting the order %s", input.IdOrder)
	return nil, GetOrderOutput{Order: fmt.Sprintf("order %s", input.IdOrder)}, nil
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
	server := mcp.NewServer(&mcp.Implementation{Name: "order", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "orderStatus", Description: "check the order status by id"}, CheckOrderStatus)
	mcp.AddTool(server, &mcp.Tool{Name: "getOrder", Description: "get the order by id"}, GetOrder)

	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Printf("Erro ao abrir TCP: %v", err)
	}
	defer listener.Close()

	fmt.Println("Servidor MCP TCP rodando na porta 9000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			transport := NewTCPTransport(c)
			if err := server.Run(context.Background(), transport); err != nil {
				fmt.Printf("Erro MCP TCP: %v", err)
			}
		}(conn)
	}
}
