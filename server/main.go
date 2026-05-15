package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type contextKey string

const userTokenKey contextKey = "X-User-Token"

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
		token := r.Header.Get("X-User-Token")
		ctx := context.WithValue(r.Context(), userTokenKey, token)
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
	msg := args.Message

	if greeting := os.Getenv("GREETING"); greeting != "" {
		msg = greeting + ": " + msg
	}

	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		msg += fmt.Sprintf(" [SECRET_KEY set, len=%d]", len(secretKey))
	}

	if token, _ := ctx.Value(userTokenKey).(string); token != "" {
		msg += fmt.Sprintf(" [X-User-Token present, len=%d]", len(token))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}
