package k8s

import (
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources"
	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
)

var DefaultTools = []string{"all"}

func InitToolsets(passedToolsets []string, readOnly bool, getClient toolsets.GetClientFn, t translations.TranslationHelperFunc, enabledResourceTypes []string) (*toolsets.ToolsetGroup, error) {
	// Create a new toolset group
	tsg := toolsets.NewToolsetGroup(readOnly)

	// Create a resource registry
	registry := toolsets.NewResourceRegistry()

	// Register resources based on enabledResourceTypes
	if len(enabledResourceTypes) == 0 || contains(enabledResourceTypes, "all") {
		// Register all resources with the registry
		resources.RegisterAllResources(registry, getClient, t)
	} else {
		// Register only the specified resources
		resources.RegisterSelectedResources(registry, getClient, t, enabledResourceTypes)
	}

	// Create a toolset from the registry
	resourcesToolset := resources.CreateToolset(registry, "k8s_resources")

	// Add the toolset to the toolset group
	tsg.AddToolset(resourcesToolset)

	// Enable the requested features
	if err := tsg.EnableToolsets(passedToolsets); err != nil {
		return nil, err
	}

	return tsg, nil
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
