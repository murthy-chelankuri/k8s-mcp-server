package resourcetypes

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/client-go/kubernetes"
)

// GetClientFn is a function type that returns a Kubernetes client interface
type GetClientFn func(context.Context) (kubernetes.Interface, error)

// ResourceHandler defines the interface for all Kubernetes resource handlers
type ResourceHandler interface {
	// RegisterTools registers all tools for this resource with the provided toolset
	RegisterTools(toolset *Toolset)
}

// Toolset is a wrapper around the toolsets.Toolset to provide resource-specific registration methods
type Toolset struct {
	Name       string
	ReadTools  []ToolWithHandler
	WriteTools []ToolWithHandler
}

// ToolWithHandler represents a tool and its handler function
type ToolWithHandler struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

// NewToolset creates a new toolset with the given name
func NewToolset(name string) *Toolset {
	return &Toolset{
		Name:       name,
		ReadTools:  []ToolWithHandler{},
		WriteTools: []ToolWithHandler{},
	}
}

// AddReadTool adds a read tool to the toolset
func (t *Toolset) AddReadTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	t.ReadTools = append(t.ReadTools, ToolWithHandler{Tool: tool, Handler: handler})
}

// AddWriteTool adds a write tool to the toolset
func (t *Toolset) AddWriteTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	t.WriteTools = append(t.WriteTools, ToolWithHandler{Tool: tool, Handler: handler})
}

// ResourceRegistry is a registry for all resource handlers
type ResourceRegistry struct {
	handlers map[string]ResourceHandler
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		handlers: make(map[string]ResourceHandler),
	}
}

// Register registers a resource handler with the registry
func (r *ResourceRegistry) Register(name string, handler ResourceHandler) {
	r.handlers[name] = handler
}

// GetHandler returns a resource handler by name
func (r *ResourceRegistry) GetHandler(name string) (ResourceHandler, bool) {
	handler, ok := r.handlers[name]
	return handler, ok
}

// GetAllHandlers returns all registered resource handlers
func (r *ResourceRegistry) GetAllHandlers() map[string]ResourceHandler {
	return r.handlers
}
