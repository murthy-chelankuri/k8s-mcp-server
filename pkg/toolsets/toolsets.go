package toolsets

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/client-go/kubernetes"
)

// GetClientFn is a function type that returns a Kubernetes client interface
type GetClientFn func(context.Context) (kubernetes.Interface, error)

// NewServerTool creates a new ServerTool with the given tool and handler
func NewServerTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	return server.ServerTool{Tool: tool, Handler: handler}
}

// Toolset represents a group of related tools
type Toolset struct {
	Name        string
	Description string
	Enabled     bool
	readOnly    bool
	writeTools  []server.ServerTool
	readTools   []server.ServerTool
}

// NewToolset creates a new toolset with the given name and description
func NewToolset(name string, description string, readOnly bool) *Toolset {
	return &Toolset{
		Name:        name,
		Description: description,
		Enabled:     true,
		readOnly:    readOnly,
	}
}

// GetActiveTools returns all active tools for this toolset
func (t *Toolset) GetActiveTools() []server.ServerTool {
	if t.Enabled {
		if t.readOnly {
			return t.readTools
		}
		return append(t.readTools, t.writeTools...)
	}
	return nil
}

// GetAvailableTools returns all available tools for this toolset
func (t *Toolset) GetAvailableTools() []server.ServerTool {
	if t.readOnly {
		return t.readTools
	}
	return append(t.readTools, t.writeTools...)
}

// RegisterTools registers all tools with the server
func (t *Toolset) RegisterTools(s *server.MCPServer) {
	if !t.Enabled {
		return
	}
	for _, tool := range t.readTools {
		s.AddTool(tool.Tool, tool.Handler)
	}
	if !t.readOnly {
		for _, tool := range t.writeTools {
			s.AddTool(tool.Tool, tool.Handler)
		}
	}
}

// SetReadOnly sets the toolset to read-only mode
func (t *Toolset) SetReadOnly() {
	// Set the toolset to read-only
	t.readOnly = true
}

// AddReadTool adds a mcp tool and handler func to the toolset
func (t *Toolset) AddReadTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	t.readTools = append(t.readTools, NewServerTool(tool, handler))
}

// AddWriteTool adds a write tool to the toolset
func (t *Toolset) AddWriteTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	if !t.readOnly {
		t.writeTools = append(t.writeTools, NewServerTool(tool, handler))
	}
}

// K8sResourceHandler defines the interface for all Kubernetes resource handlers
type K8sResourceHandler interface {
	// RegisterTools registers all tools for a k8s resource with the provided toolset
	RegisterTools(toolset *Toolset)

	// RegisterResources registers all mcp resources for a k8s resource with the provided resource set
	// RegisterResources(resourceSet *ResourceSet)
}

// K8sResourceRegistry is a registry for all resource handlers
type K8sResourceRegistry struct {
	handlers map[string]K8sResourceHandler
}

// NewK8sResourceRegistry creates a new resource registry
func NewK8sResourceRegistry() *K8sResourceRegistry {
	return &K8sResourceRegistry{
		handlers: make(map[string]K8sResourceHandler),
	}
}

// Register registers a resource handler with the registry
func (r *K8sResourceRegistry) Register(name string, handler K8sResourceHandler) {
	r.handlers[name] = handler
}

// GetHandler returns a resource handler by name
func (r *K8sResourceRegistry) GetHandler(name string) (K8sResourceHandler, bool) {
	handler, ok := r.handlers[name]
	return handler, ok
}

// GetAllHandlers returns all registered resource handlers
func (r *K8sResourceRegistry) GetAllHandlers() map[string]K8sResourceHandler {
	return r.handlers
}
