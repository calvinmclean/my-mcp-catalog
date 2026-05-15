package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type contextKey string

const headersKey contextKey = "headers"

type EchoArgs struct {
	Message string `json:"message"`
}

func main() {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "ping-echo",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ping",
		Description: "Returns pong along with the container hostname, useful for verifying all requests share the same instance",
	}, pingHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes a message. Prepends $GREETING if set. Appends SECRET_KEY length if set. Includes X-User-Token presence if injected by Obot.",
	}, echoHandler)

	// Wrap the MCP handler with middleware that injects HTTP headers into context.
	// This allows tool handlers to read per-user headers injected by Obot.
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), headersKey, r.Header)
		mcpHandler.ServeHTTP(w, r.WithContext(ctx))
	}))

	addr := ":8080"
	log.Printf("MCP server listening on %s/mcp", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func pingHandler(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	host, _ := os.Hostname()
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("pong (host: %s)", host)},
		},
	}, nil, nil
}

func echoHandler(ctx context.Context, _ *mcp.CallToolRequest, args EchoArgs) (*mcp.CallToolResult, any, error) {
	result := map[string]any{
		"message":    args.Message,
		"GREETING":   os.Getenv("GREETING"),
		"SECRET_KEY": os.Getenv("SECRET_KEY"),
		"headers":    map[string][]string{},
	}

	if headers, ok := ctx.Value(headersKey).(http.Header); ok {
		result["headers"] = map[string][]string(headers)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}
