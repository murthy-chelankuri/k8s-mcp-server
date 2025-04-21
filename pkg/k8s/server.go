package k8s

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new K8s MCP Server with the specified K8s clientset and logger.
func NewServer(version string, opts ...server.ServerOption) *server.MCPServer {
	// Add default options
	defaultOpts := []server.ServerOption{
		server.WithToolCapabilities(true),
		// server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	}
	opts = append(defaultOpts, opts...)

	// Create a new MCP server
	s := server.NewMCPServer(
		"k8s-mcp-server",
		version,
		opts...,
	)
	return s
}
