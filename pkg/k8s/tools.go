package k8s

import (
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources"
	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
)

var DefaultTools = []string{"all"}

func InitToolset(readOnly bool, getClient toolsets.GetClientFn, t translations.TranslationHelperFunc, enabledResourceTypes []string) (*toolsets.Toolset, error) {

	// Create a resource registry
	registry := toolsets.NewK8sResourceRegistry()

	// Register resources based on enabledResourceTypes
	if len(enabledResourceTypes) == 0 || contains(enabledResourceTypes, "all") {
		// Register all k8s resources with the registry
		resources.RegisterAllK8sResources(registry, getClient, t)
	} else {
		// Register only the specified k8s resources
		resources.RegisterSelectedK8sResources(registry, getClient, t, enabledResourceTypes)
	}

	// Create a toolset from the registry
	k8sToolset := resources.CreateToolset(registry, "k8s_resources")

	return k8sToolset, nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
