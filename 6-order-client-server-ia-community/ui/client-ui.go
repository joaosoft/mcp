package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ==================== Structs ====================

type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Example     json.RawMessage `json:"example,omitempty"`
}

type ResourceSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType"`
}

type ChatMessage struct {
	Message string `json:"message"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

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

// ==================== UI Template ====================

var uiTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>MCP Go UI</title>
<style>
body { font-family: sans-serif; background: #1e1e1e; color: #eee; margin: 0; padding: 0; }
header { background: #282828; padding: 10px; text-align: center; font-size: 24px; }
nav { display: flex; background: #333; }
nav button { flex: 1; padding: 10px; background: #444; border: none; color: #fff; cursor: pointer; }
nav button.active { background: #4a90e2; }
section { padding: 10px; }
.tool, .resource { border: 1px solid #444; padding: 10px; margin: 10px 0; border-radius: 5px; background: #2a2a2a; }
.chat-box { display: flex; flex-direction: column; height: 80vh; }
.chat-messages { flex: 1; overflow-y: auto; background: #111; padding: 10px; border-radius: 5px; margin-bottom: 10px; }
.chat-message { margin: 5px 0; padding: 8px; border-radius: 5px; max-width: 80%; }
.user { background: #4a90e2; align-self: flex-end; }
.bot { background: #333; align-self: flex-start; }
.chat-input { display: flex; }
.chat-input input { flex: 1; padding: 10px; border-radius: 5px; border: 1px solid #555; background: #1e1e1e; color: #eee; }
.chat-input button { margin-left: 5px; padding: 10px 15px; background: #4a90e2; border: none; border-radius: 5px; color: #fff; cursor: pointer; }
</style>
</head>
<body>
<header>MCP Go UI</header>
<nav>
	<button class="tab-btn active" data-tab="tools">Tools</button>
	<button class="tab-btn" data-tab="resources">Resources</button>
	<button class="tab-btn" data-tab="chat">Chat</button>
</nav>
<section id="content">
	<div id="tools" class="tab active-tab"></div>
	<div id="resources" class="tab" style="display:none;"></div>
	<div id="chat" class="tab" style="display:none;">
		<div class="chat-box">
			<div id="chatMessages" class="chat-messages"></div>
			<div class="chat-input">
				<input type="text" id="chatInput" placeholder="Type your message..." />
				<button onclick="sendChat()">Send</button>
			</div>
		</div>
	</div>
</section>

<script>
// ==================== Tabs ====================
document.querySelectorAll('.tab-btn').forEach(btn => {
	btn.addEventListener('click', () => {
		document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
		document.querySelectorAll('.tab').forEach(t => t.style.display = 'none');
		btn.classList.add('active');
		document.getElementById(btn.dataset.tab).style.display = 'block';
	});
});

// ==================== Chat ====================
const chatInput = document.getElementById('chatInput');
chatInput.addEventListener('keypress', e => {
	if (e.key === 'Enter') sendChat();
});

async function sendChat() {
	const input = document.getElementById('chatInput');
	const message = input.value.trim();
	if (!message) return;

	addMessage('user', message);
	input.value = '';

	try {
		const res = await fetch('/chat', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ message })
		});
		const data = await res.json();
		if (data.error) addMessage('bot', '❌ ' + data.error);
		else addMessage('bot', data.response || '(no answer)');
	} catch(err) {
		addMessage('bot', '❌ Network error');
	}
}

function addMessage(role, text) {
	const container = document.getElementById('chatMessages');
	const div = document.createElement('div');
	div.className = 'chat-message ' + role;
	div.innerText = text;
	container.appendChild(div);
	setTimeout(() => container.scrollTop = container.scrollHeight, 50);
}

// ==================== Tools ====================
async function loadTools(auto = false) {
	const res = await fetch('/tools');
	const tools = await res.json();
	const container = document.getElementById('tools');
	if (!auto) container.innerHTML = '';

	tools.forEach(t => {
		if (auto && document.getElementById("tool_" + t.name)) return;
		const toolDiv = document.createElement('div');
		toolDiv.className = 'tool';
		toolDiv.id = "tool_" + t.name;

		const h3 = document.createElement('h3'); h3.innerText = t.name; toolDiv.appendChild(h3);
		const p = document.createElement('p'); p.innerText = t.description; toolDiv.appendChild(p);

		container.appendChild(toolDiv);
	});
}
setInterval(() => loadTools(true), 5000);

// ==================== Resources ====================
async function loadResources(auto = false) {
	const res = await fetch('/resources');
	const resources = await res.json();
	const container = document.getElementById('resources');
	if (!auto) container.innerHTML = '';

	resources.forEach(r => {
		const id = "resource_" + r.name.replace(/\s+/g, '_');
		if (auto && document.getElementById(id)) return;
		const div = document.createElement('div');
		div.className = 'resource'; div.id = id;

		const h3 = document.createElement('h3'); h3.innerText = r.name + " (" + (r.mimeType || "unknown") + ")"; div.appendChild(h3);
		const p = document.createElement('p'); p.innerText = r.description || "No description."; div.appendChild(p);

		container.appendChild(div);
	});
}
setInterval(() => loadResources(true), 5000);

loadTools();
loadResources();
</script>
</body>
</html>
`

// ==================== Helpers ====================

// respondError centralizes HTTP error response
func respondError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// getDynamicToolList returns available tools as a formatted string
func getDynamicToolList(ctx context.Context, mcpClient *client.Client) string {
	res, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return "- calculate: arithmetic tool (default)\n"
	}
	list := ""
	for _, t := range res.Tools {
		list += fmt.Sprintf("- %s: %s with input schema %s \n", t.Name, t.Description, t.InputSchema)
	}
	return list
}

// callLLM sends a prompt to LLM and returns parsed JSON
func callLLM(systemPrompt, userPrompt string) (map[string]any, error) {
	reqBody := LLMRequest{
		Model: "qwen/qwen3-vl-4b",
		Messages: MessageList{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxNewTokens: 50,
		Temperature:  0.0,
		N:            1,
	}
	data, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://127.0.0.1:1234/v1/chat/completions", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return nil, err
	}
	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from LLM")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(llmResp.Choices[0].Message.Content), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %v", err)
	}
	return parsed, nil
}

// ==================== Main ====================

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mcpClient, err := client.NewStdioMCPClient("go", os.Environ(), "run", "server.go")
	if err != nil {
		log.Fatalf("Error creating MCP client: %v", err)
	}
	defer mcpClient.Close()

	initReq := mcp.InitializeRequest{Params: mcp.InitializeParams{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo:      mcp.Implementation{Name: "Go MCP UI", Version: "1.0"},
	}}
	serverInfo, err := mcpClient.Initialize(ctx, initReq)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	if err = mcpClient.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}

	fmt.Printf("Connected to MCP server: %s (%s)\n", serverInfo.ServerInfo.Name, serverInfo.ServerInfo.Version)

	// ==================== HTTP Handlers ====================
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("ui").Parse(uiTemplate))
		_ = tmpl.Execute(w, nil)
	})

	http.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		res, err := mcpClient.ListTools(r.Context(), mcp.ListToolsRequest{})
		if err != nil {
			respondError(w, err)
			return
		}
		var tools []ToolSchema
		for _, t := range res.Tools {
			raw, _ := json.Marshal(t.InputSchema)
			tools = append(tools, ToolSchema{Name: t.Name, Description: t.Description, InputSchema: raw})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tools)
	})

	http.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		res, err := mcpClient.ListResources(r.Context(), mcp.ListResourcesRequest{})
		if err != nil {
			respondError(w, err)
			return
		}
		var list []ResourceSchema
		for _, r := range res.Resources {
			list = append(list, ResourceSchema{Name: r.Name, Description: r.Description, MIMEType: r.MIMEType})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST", http.StatusMethodNotAllowed)
			return
		}
		var msg ChatMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}

		systemPrompt := fmt.Sprintf(`
You are an assistant. Decide if the user's request requires a tool.
Available tools:
%s
If found, return JSON: {"tool":"<tool_name>","arguments":{...}}
If not, return JSON: {"message":"<plain text reply>"}
Return ONLY JSON.`, getDynamicToolList(r.Context(), mcpClient))

		parsed, err := callLLM(systemPrompt, msg.Message)
		if err != nil {
			respondError(w, fmt.Errorf("LLM error: %v", err))
			return
		}
		fmt.Println("LLM parsed:", parsed)

		tool, _ := parsed["tool"].(string)
		arguments, _ := parsed["arguments"].(map[string]any)
		message, _ := parsed["message"].(string)

		reply := ""
		if message == "" && tool != "" {
			// validate if tool exists
			res, _ := mcpClient.ListTools(r.Context(), mcp.ListToolsRequest{})
			found := false
			for _, t := range res.Tools {
				if t.Name == tool {
					found = true
					break
				}
			}
			if !found {
				respondError(w, fmt.Errorf("tool %s not found on server", tool))
				return
			}

			callReq := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool, Arguments: arguments}}
			callToolResponse, err := mcpClient.CallTool(r.Context(), callReq)
			if err != nil {
				respondError(w, err)
				return
			}

			if len(callToolResponse.Content) > 0 {
				switch v := callToolResponse.Content[0].(type) {
				case mcp.TextContent:
					reply = v.Text
				default:
					reply = fmt.Sprintf("unknown content type: %T", v)
				}
			} else {
				reply = "no answer"
			}
		} else if message != "" {
			reply = message
		} else {
			reply = "no answer"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"response": reply})
	})

	fmt.Println("🚀 MCP Go UI running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
