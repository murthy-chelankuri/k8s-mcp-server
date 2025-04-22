package toolsets

import (
	"context"
	"fmt"

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
func NewToolset(name string, description string) *Toolset {
	return &Toolset{
		Name:        name,
		Description: description,
		Enabled:     false,
		readOnly:    false,
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

// ToolsetGroup represents a group of toolsets
type ToolsetGroup struct {
	Toolsets     map[string]*Toolset
	everythingOn bool
	readOnly     bool
}

// NewToolsetGroup creates a new toolset group
func NewToolsetGroup(readOnly bool) *ToolsetGroup {
	return &ToolsetGroup{
		Toolsets:     make(map[string]*Toolset),
		everythingOn: false,
		readOnly:     readOnly,
	}
}

// AddToolset adds a toolset to the group
func (tg *ToolsetGroup) AddToolset(ts *Toolset) {
	if tg.readOnly {
		ts.SetReadOnly()
	}
	tg.Toolsets[ts.Name] = ts
}

// IsEnabled checks if a toolset is enabled
func (tg *ToolsetGroup) IsEnabled(name string) bool {
	// If everythingOn is true, all features are enabled
	if tg.everythingOn {
		return true
	}

	feature, exists := tg.Toolsets[name]
	if !exists {
		return false
	}
	return feature.Enabled
}

// EnableToolsets enables multiple toolsets by name
func (tg *ToolsetGroup) EnableToolsets(names []string) error {
	// Special case for "all"
	for _, name := range names {
		if name == "all" {
			tg.everythingOn = true
			break
		}
		err := tg.EnableToolset(name)
		if err != nil {
			return err
		}
	}
	// Do this after to ensure all toolsets are enabled if "all" is present anywhere in list
	if tg.everythingOn {
		for name := range tg.Toolsets {
			err := tg.EnableToolset(name)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

// EnableToolset enables a single toolset by name
func (tg *ToolsetGroup) EnableToolset(name string) error {
	toolset, exists := tg.Toolsets[name]
	if !exists {
		return fmt.Errorf("toolset %s does not exist", name)
	}
	toolset.Enabled = true
	tg.Toolsets[name] = toolset
	return nil
}

// RegisterTools registers all enabled toolsets with the server
func (tg *ToolsetGroup) RegisterTools(s *server.MCPServer) {
	for _, toolset := range tg.Toolsets {
		toolset.RegisterTools(s)
	}
}

// ResourceHandler defines the interface for all Kubernetes resource handlers
type ResourceHandler interface {
	// RegisterTools registers all tools for this resource with the provided toolset
	RegisterTools(toolset *Toolset)
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
