package k8s

import (
	"context"

	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"k8s.io/client-go/kubernetes"
)

type GetClientFn func(context.Context) (kubernetes.Interface, error)

var DefaultTools = []string{"all"}

func InitToolsets(passedToolsets []string, readOnly bool, getClient GetClientFn, t translations.TranslationHelperFunc) (*toolsets.ToolsetGroup, error) {
	// Create a new toolset group
	tsg := toolsets.NewToolsetGroup(readOnly)

	// Define all available features with their default state (disabled)
	// Create toolsets
	resources := toolsets.NewToolset("k8s_resources", "K8s resources related tools").
		AddReadTools(
			toolsets.NewServerTool(GetPod(getClient, t)),
			toolsets.NewServerTool(ListPods(getClient, t)),
			toolsets.NewServerTool(GetPodLogs(getClient, t)),
			toolsets.NewServerTool(GetDeployment(getClient, t)),
			toolsets.NewServerTool(ListDeployments(getClient, t)),
			toolsets.NewServerTool(ScaleDeployment(getClient, t)),
			toolsets.NewServerTool(GetService(getClient, t)),
			toolsets.NewServerTool(ListServices(getClient, t)),
			toolsets.NewServerTool(GetConfigMap(getClient, t)),
			toolsets.NewServerTool(ListConfigMaps(getClient, t)),
			toolsets.NewServerTool(ListNamespaces(getClient, t)),
			toolsets.NewServerTool(ListNodes(getClient, t)),
		).
		AddWriteTools(
			toolsets.NewServerTool(DeletePod(getClient, t)),
			toolsets.NewServerTool(ScaleDeployment(getClient, t)),
		)

	// tsg.AddToolset(pullRequests)
	tsg.AddToolset(resources)

	// Enable the requested features
	if err := tsg.EnableToolsets(passedToolsets); err != nil {
		return nil, err
	}

	return tsg, nil
}
