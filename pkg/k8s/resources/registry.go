package resources

import (
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/configmap"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/deployment"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/namespace"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/node"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/pod"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources/service"
	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
)

// RegisterAllK8sResources registers all k8s resource handlers with the registry
func RegisterAllK8sResources(registry *toolsets.K8sResourceRegistry, getClient toolsets.GetClientFn, t translations.TranslationHelperFunc) {
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

// RegisterSelectedK8sResources registers only the specified resource handlers with the registry
func RegisterSelectedK8sResources(registry *toolsets.K8sResourceRegistry, getClient toolsets.GetClientFn, t translations.TranslationHelperFunc, resourceTypes []string) {
	// Map of resource types to their registration functions
	resourceMap := map[string]func(){
		"pod": func() {
			registry.Register("pod", pod.NewHandler(getClient, t))
		},
		"deployment": func() {
			registry.Register("deployment", deployment.NewHandler(getClient, t))
		},
		"service": func() {
			registry.Register("service", service.NewHandler(getClient, t))
		},
		"configmap": func() {
			registry.Register("configmap", configmap.NewHandler(getClient, t))
		},
		"namespace": func() {
			registry.Register("namespace", namespace.NewHandler(getClient, t))
		},
		"node": func() {
			registry.Register("node", node.NewHandler(getClient, t))
		},
	}

	// Register only the specified resources
	for _, resourceType := range resourceTypes {
		if registerFunc, ok := resourceMap[resourceType]; ok {
			registerFunc()
		}
	}
}

// CreateToolset creates a toolset with all registered resource handlers
func CreateToolset(registry *toolsets.K8sResourceRegistry, name string) *toolsets.Toolset {
	// Create a new toolset
	toolset := toolsets.NewToolset(name, "K8s resources related tools")

	// Register all resource handlers with the toolset
	for _, handler := range registry.GetAllHandlers() {
		handler.RegisterTools(toolset)
	}

	return toolset
}
