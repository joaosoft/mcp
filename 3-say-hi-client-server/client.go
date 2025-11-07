package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// Estrutura do request MCP mínimo
type CallToolRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	JsonRPC string                 `json:"jsonrpc"`
}

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("Erro ao conectar TCP: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Lê nome do usuário
	fmt.Print("Enter your name: ")
	var name string
	fmt.Scanln(&name)

	// Cria request JSON-RPC mínimo
	req := CallToolRequest{
		ID:      "1",
		Method:  "call_tool",
		Params:  map[string]interface{}{"tool_name": "greet", "arguments": map[string]interface{}{"name": name}},
		JsonRPC: "2.0",
	}

	data, _ := json.Marshal(req)
	data = append(data, '\n') // cada request em linha separada
	_, err = conn.Write(data)
	if err != nil {
		log.Fatalf("Erro ao escrever no TCP: %v", err)
	}

	// Lê resposta
	resp, _ := reader.ReadString('\n')
	fmt.Println("Server response:", resp)
}
