package node

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

// Handler implements the ResourceHandler interface for Node resources
type Handler struct {
	getClient resourcetypes.GetClientFn
	t         translations.TranslationHelperFunc
}

// NewHandler creates a new Node resource handler
func NewHandler(getClient resourcetypes.GetClientFn, t translations.TranslationHelperFunc) *Handler {
	return &Handler{
		getClient: getClient,
		t:         t,
	}
}

// RegisterTools registers all Node resource tools with the provided toolset
func (h *Handler) RegisterTools(toolset *resourcetypes.Toolset) {
	// Register read tools
	listTool, listHandler := h.List()
	toolset.AddReadTool(listTool, listHandler)
}

// List creates a tool to list all nodes
func (h *Handler) List() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_nodes",
			mcp.WithDescription(h.t("TOOL_LIST_NODES_DESCRIPTION", "List all nodes in the cluster")),
			mcp.WithString("fieldSelector",
				mcp.Description("Selector to restrict the list of returned objects by their fields"),
			),
			mcp.WithString("labelSelector",
				mcp.Description("Selector to restrict the list of returned objects by their labels"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

			nodes, err := client.CoreV1().Nodes().List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list nodes: %v", err)), nil
			}

			r, err := json.Marshal(nodes)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
