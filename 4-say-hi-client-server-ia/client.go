// client.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- LLaMA request/response types (ajusta se o teu backend usar outro formato) ---
type LLMRequest struct {
	Model        string  `json:"model"`
	Prompt       string  `json:"prompt"`
	MaxNewTokens int     `json:"max_new_tokens"`
	Temperature  float64 `json:"temperature"`
}

type LLMResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// getNameFromLLM chama o client e devolve o texto (podes melhorar extração)
func getNameFromLLM(prompt string) string {
	reqBody := LLMRequest{
		Model:        "qwen/qwen3-vl-4b",
		Prompt:       "Extract the name of the person from this prompt and return only the name without any response: " + prompt,
		MaxNewTokens: 20,
		Temperature:  0.0,
	}
	data, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://127.0.0.1:1234/v1/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Println("Erro ao chamar LLM local: %v", err)
		return "User"
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erro a ler resposta LLM: %v", err)
		return "User"
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		// tenta fallback para texto cru
		txt := strings.TrimSpace(string(body))
		if txt == "" {
			return "User"
		}
		return txt
	}

	if len(llmResp.Choices) == 0 {
		return "User"
	}
	// retorna o texto inteiro da escolha 0 - podes refinar com regex se quiseres só o nome
	return strings.TrimSpace(llmResp.Choices[0].Text)
}

// --- Estruturas ChatGPT-5 ---
type ChatGPTRequest struct {
	Model     string              `json:"model"`
	Messages  []map[string]string `json:"messages"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
}

type ChatGPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ----------------- Transport TCP minimal compatível com mcp.Client.Connect -----------------
// Este transport implementa mcp.Transport:
//
//	Connect(ctx) (mcp.Connection, error)
//
// A mcp.Connection deve implementar:
//
//	Read(ctx) (jsonrpc.Message, error)
//	Write(ctx, jsonrpc.Message) error
//	Close() error
//	SessionID() string
//
// Vamos usar as funções auxiliares do pacote jsonrpc para (de)codificar mensagens.
type tcpConnection struct {
	conn      net.Conn
	sessionID string
	r         *bufio.Reader
}

func newTCPConnection(c net.Conn) *tcpConnection {
	return &tcpConnection{
		conn:      c,
		sessionID: "session-" + c.RemoteAddr().String(),
		r:         bufio.NewReader(c),
	}
}

// Read uma mensagem jsonrpc (linha-terminada)
func (c *tcpConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	line, err := c.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return jsonrpc.DecodeMessage(line)
}

func (c *tcpConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	b, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = c.conn.Write(b)
	return err
}

func (c *tcpConnection) Close() error {
	return c.conn.Close()
}

func (c *tcpConnection) SessionID() string {
	return c.sessionID
}

// transport que o client.Connect espera
type tcpTransport struct {
	addr string
	// nota: aqui mantemos apenas a conn usada para Connect; cada Connect pode criar uma nova connection
	conn net.Conn
}

func newTCPTransport(addr string) *tcpTransport {
	return &tcpTransport{addr: addr}
}

// Connect abre a conexão TCP e devolve um mcp.Connection (o wrapper tcpConnection)
func (t *tcpTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	if t.conn == nil {
		conn, err := net.Dial("tcp", t.addr)
		if err != nil {
			return nil, err
		}
		t.conn = conn
	}
	return newTCPConnection(t.conn), nil
}

// Close fecha (opcional para o transport em si)
func (t *tcpTransport) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// --------------------------------- main ---------------------------------
func main() {
	ctx := context.Background()

	// cria o client MCP (Implementation config simples)
	client := mcp.NewClient(&mcp.Implementation{Name: "tcp-client", Version: "v1.0.0"}, nil)

	// transport para o servidor MCP (porta do servidor que tens a correr)
	transport := newTCPTransport("127.0.0.1:9000")

	// Connect -> devolve uma Session (e faz handshake/initialize internamente)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Erro client.Connect: %v", err)
	}

	defer session.Close()

	// agora podes chamar ferramentas (call_tool) diretamente na session
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Digite prompts (ex: cumprimente João)")

	for {
		fmt.Print("> ")
		prompt, _ := reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}

		// chama LLM local para obter texto/nome
		nameText := getNameFromLLM(prompt)
		// extrai nome simples (faz uma limpeza rápida)
		name := strings.TrimSpace(strings.Split(nameText, "\n")[0])
		name = strings.TrimPrefix(name, "Answer:")
		name = strings.TrimSpace(name)

		// Chama a tool 'greet' no servidor MCP via session.CallTool
		params := &mcp.CallToolParams{
			Name:      "greet",
			Arguments: map[string]any{"name": name},
		}

		res, err := session.CallTool(ctx, params)
		if err != nil {
			fmt.Println("CallTool error: %v", err)
			continue
		}
		if res.IsError {
			fmt.Println("Tool returned error: %+v", res.IsError)
			continue
		}

		// imprime o conteúdo textual (se houver)
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				fmt.Println("Resposta MCP:", tc.Text)
			}
		}
	}
}
