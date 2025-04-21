package configmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resourcetypes"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Handler implements the ResourceHandler interface for ConfigMap resources
type Handler struct {
	getClient resourcetypes.GetClientFn
	t         translations.TranslationHelperFunc
}

// NewHandler creates a new ConfigMap resource handler
func NewHandler(getClient resourcetypes.GetClientFn, t translations.TranslationHelperFunc) *Handler {
	return &Handler{
		getClient: getClient,
		t:         t,
	}
}

// RegisterTools registers all ConfigMap resource tools with the provided toolset
func (h *Handler) RegisterTools(toolset *resourcetypes.Toolset) {
	// Register read tools
	getTool, getHandler := h.Get()
	toolset.AddReadTool(getTool, getHandler)

	listTool, listHandler := h.List()
	toolset.AddReadTool(listTool, listHandler)
}

// Get creates a tool to get details of a specific configmap
func (h *Handler) Get() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_configmap",
			mcp.WithDescription(h.t("TOOL_GET_CONFIGMAP_DESCRIPTION", "Get details of a specific configmap")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("ConfigMap name"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := resourcetypes.RequiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := resourcetypes.RequiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			configmap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get configmap: %v", err)), nil
			}

			r, err := json.Marshal(configmap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// List creates a tool to list configmaps in a namespace
func (h *Handler) List() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_configmaps",
			mcp.WithDescription(h.t("TOOL_LIST_CONFIGMAPS_DESCRIPTION", "List configmaps in a namespace")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("fieldSelector",
				mcp.Description("Selector to restrict the list of returned objects by their fields"),
			),
			mcp.WithString("labelSelector",
				mcp.Description("Selector to restrict the list of returned objects by their labels"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := resourcetypes.RequiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := resourcetypes.OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := resourcetypes.OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			options := metav1.ListOptions{
				FieldSelector: fieldSelector,
				LabelSelector: labelSelector,
			}

			configmaps, err := client.CoreV1().ConfigMaps(namespace).List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list configmaps: %v", err)), nil
			}

			r, err := json.Marshal(configmaps)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
