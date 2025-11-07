package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.SetOutput(os.Stderr)

	// Create a new MCP server
	s := server.NewMCPServer(
		"Calculator",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Add a calculator tool
	calculatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations with the numeric values returned as {operation, x, y}"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("allows you to make arithmetic operations"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add the calculator tool handler
	s.AddTool(calculatorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Using helper functions for type-safe argument access
		op, err := request.RequireString("operation")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		x, err := request.RequireFloat("x")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		y, err := request.RequireFloat("y")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var result float64
		switch op {
		case "add":
			result = x + y
		case "subtract":
			result = x - y
		case "multiply":
			result = x * y
		case "divide":
			if y == 0 {
				return mcp.NewToolResultError("cannot divide by zero"), nil
			}
			result = x / y
		}

		return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
	})

	// Add the resource handler
	resource1 := mcp.NewResource(
		"urn:example:my_resource_1",
		"MyResource1",
		mcp.WithResourceDescription("Resource 1 at MCP server"),
		mcp.WithMIMEType("application/json"),
	)

	resource2 := mcp.NewResource(
		"urn:example:my_resource_2",
		"MyResource2",
		mcp.WithResourceDescription("Resource 2 at MCP server"),
		mcp.WithMIMEType("application/json"),
	)

	// Resource handler
	handler := func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{Text: "Hello, this is a built-in resource!"},
		}, nil
	}

	// Add resources to the server
	s.AddResource(resource1, handler)
	s.AddResource(resource2, handler)

	// Start the server
	log.Println("MCP server starting...")
	if err := server.ServeStdio(s); err != nil {
		log.Println("MCP server errored")
	}
	log.Println("MCP server started")
}
