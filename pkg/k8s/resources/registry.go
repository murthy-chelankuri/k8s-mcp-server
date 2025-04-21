package resources

import (
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/configmap"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/deployment"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/namespace"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/node"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/pod"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/service"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resourcetypes"
	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
)

// RegisterAllResources registers all resource handlers with the registry
func RegisterAllResources(registry *resourcetypes.ResourceRegistry, getClient resourcetypes.GetClientFn, t translations.TranslationHelperFunc) {
	// Register Pod resource handler
	registry.Register("pod", pod.NewHandler(getClient, t))

	// Register Deployment resource handler
	registry.Register("deployment", deployment.NewHandler(getClient, t))

	// Register Service resource handler
	registry.Register("service", service.NewHandler(getClient, t))

	// Register ConfigMap resource handler
	registry.Register("configmap", configmap.NewHandler(getClient, t))

	// Register Namespace resource handler
	registry.Register("namespace", namespace.NewHandler(getClient, t))

	// Register Node resource handler
	registry.Register("node", node.NewHandler(getClient, t))
}

// CreateToolset creates a toolset with all registered resource handlers
func CreateToolset(registry *resourcetypes.ResourceRegistry, name string) *toolsets.Toolset {
	// Create a new toolset
	toolset := resourcetypes.NewToolset(name)

	// Register all resource handlers with the toolset
	for _, handler := range registry.GetAllHandlers() {
		handler.RegisterTools(toolset)
	}

	return ConvertToToolsetsToolset(toolset)
}

// ConvertToToolsetsToolset converts our Toolset to a toolsets.Toolset
func ConvertToToolsetsToolset(toolset *resourcetypes.Toolset) *toolsets.Toolset {
	ts := toolsets.NewToolset(toolset.Name, "K8s resources related tools")

	// Add read tools
	for _, tool := range toolset.ReadTools {
		ts.AddReadTools(toolsets.NewServerTool(tool.Tool, tool.Handler))
	}

	// Add write tools
	for _, tool := range toolset.WriteTools {
		ts.AddWriteTools(toolsets.NewServerTool(tool.Tool, tool.Handler))
	}

	return ts
}
