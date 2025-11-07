package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type LLMRequest struct {
	Model        string      `json:"model"`
	Messages     MessageList `json:"messages"`
	MaxNewTokens int         `json:"max_new_tokens"`
	Temperature  float64     `json:"temperature"`
	N            int         `json:"n"`
}

type MessageList []Message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// calculateParsePromptWithLLM envia o prompt para a LLM e retorna operation, x, y
func calculateParsePromptWithLLM(prompt string) (operation string, x, y float64, err error) {
	reqBody := LLMRequest{
		Model: "qwen/qwen3-vl-4b",
		Messages: MessageList{
			{
				Role:    "system",
				Content: "You are a strict parser. Return JSON filling the following with {operation, x, y} only!",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxNewTokens: 20,
		Temperature:  0.0,
		N:            1,
	}
	data, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://127.0.0.1:1234/v1/chat/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var llmResp LLMResponse
	if err = json.Unmarshal(body, &llmResp); err != nil {
		return "", 0, 0, err
	}

	if len(llmResp.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("no choices returned from LLM")
	}

	fmt.Println("LLM prompt:", prompt)
	fmt.Println("LLM choices:", llmResp.Choices[0].Message.Content)

	// Esperamos que o LLM retorne JSON: {"operation":"multiply","x":6,"y":7}
	var params struct {
		Operation string  `json:"operation"`
		X         float64 `json:"x"`
		Y         float64 `json:"y"`
	}

	if err = json.Unmarshal([]byte(llmResp.Choices[0].Message.Content), &params); err != nil {
		return "", 0, 0, fmt.Errorf("invalid JSON from LLM: %v", err)
	}

	return params.Operation, params.X, params.Y, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cria o cliente STDIO MCP
	c, err := client.NewStdioMCPClient(
		"go",               // comando
		os.Environ(),       // environment
		"run", "server.go", // argumentos
	)
	if err != nil {
		log.Fatalf("Error creating MCP stdio client: %v", err)
	}
	defer c.Close()

	// Inicializa o cliente
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

	// Ping opcional
	if err = c.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	fmt.Println("Server is alive and responding")
	fmt.Printf("Connected to server: %s (version %s)\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)

	// Lista ferramentas
	toolsRes, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Error listing tools: %v", err)
	}
	fmt.Println("Available tools:")
	for _, t := range toolsRes.Tools {
		fmt.Printf("- %s: %s\n", t.Name, t.Description)
	}

	// Loop de prompts
	for {
		fmt.Print("\nEnter calculation (e.g., 'Multiply 6 by 7'): ")
		reader := bufio.NewReader(os.Stdin)
		prompt, _ := reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}

		// Chama a LLM para parse do prompt
		op, x, y, err := calculateParsePromptWithLLM(prompt)
		fmt.Println("LLM parsed: operation:", op, "x:", x, "y:", y)
		if err != nil {
			fmt.Println("LLM parse error:", err)
			continue
		}

		// Chama tool MCP "calculate"
		callReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "calculate",
				Arguments: map[string]any{
					"operation": op,
					"x":         x,
					"y":         y,
				},
			},
		}

		callCtx, callCancel := context.WithTimeout(context.Background(), 10*time.Second)
		callRes, err := c.CallTool(callCtx, callReq)
		callCancel()
		if err != nil {
			fmt.Println("CallTool error:", err)
			continue
		}

		if len(callRes.Content) > 0 {
			if txt, ok := callRes.Content[0].(mcp.TextContent); ok {
				fmt.Println("Calculation result:", txt.Text)
			} else {
				fmt.Println("Calculation result: <unknown type>")
			}
		} else {
			fmt.Println("Calculation result: <empty>")
		}
	}
}
