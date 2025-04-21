package k8s

import (
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resources"
	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resourcetypes"
	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
)

// GetClientFn is a function type that returns a Kubernetes client interface
// type GetClientFn = resourcetypes.GetClientFn

var DefaultTools = []string{"all"}

func InitToolsets(passedToolsets []string, readOnly bool, getClient resourcetypes.GetClientFn, t translations.TranslationHelperFunc) (*toolsets.ToolsetGroup, error) {
	// Create a new toolset group
	tsg := toolsets.NewToolsetGroup(readOnly)

	// Create a resource registry
	registry := resourcetypes.NewResourceRegistry()

	// Register all resources with the registry
	resources.RegisterAllResources(registry, getClient, t)

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
