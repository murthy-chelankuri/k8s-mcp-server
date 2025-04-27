package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Handler implements the K8sResourceHandler interface for Deployment resources
type Handler struct {
	getClient toolsets.GetClientFn
	t         translations.TranslationHelperFunc
}

// NewHandler creates a new Deployment resource handler
func NewHandler(getClient toolsets.GetClientFn, t translations.TranslationHelperFunc) *Handler {
	return &Handler{
		getClient: getClient,
		t:         t,
	}
}

// RegisterTools registers all Deployment resource tools with the provided toolset
func (h *Handler) RegisterTools(toolset *toolsets.Toolset) {
	// Register read tools
	getTool, getHandler := h.Get()
	toolset.AddReadTool(getTool, getHandler)

	listTool, listHandler := h.List()
	toolset.AddReadTool(listTool, listHandler)

	// Register write tools
	scaleTool, scaleHandler := h.Scale()
	toolset.AddWriteTool(scaleTool, scaleHandler)
}

// Get creates a tool to get details of a specific deployment
func (h *Handler) Get() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_deployment",
			mcp.WithDescription(h.t("TOOL_GET_DEPLOYMENT_DESCRIPTION", "Get details of a specific deployment")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Deployment name"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := toolsets.RequiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := toolsets.RequiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get deployment: %v", err)), nil
			}

			r, err := json.Marshal(deployment)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// List creates a tool to list deployments in a namespace
func (h *Handler) List() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_deployments",
			mcp.WithDescription(h.t("TOOL_LIST_DEPLOYMENTS_DESCRIPTION", "List deployments in a namespace")),
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
			namespace, err := toolsets.RequiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := toolsets.OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := toolsets.OptionalParam[string](request, "labelSelector")
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

			deployments, err := client.AppsV1().Deployments(namespace).List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
			}

			r, err := json.Marshal(deployments)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// Scale creates a tool to scale a deployment
func (h *Handler) Scale() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("scale_deployment",
			mcp.WithDescription(h.t("TOOL_SCALE_DEPLOYMENT_DESCRIPTION", "Scale a deployment to a specified number of replicas")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Deployment name"),
			),
			mcp.WithNumber("replicas",
				mcp.Required(),
				mcp.Description("Number of replicas"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := toolsets.RequiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := toolsets.RequiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			replicasFloat, err := toolsets.RequiredParam[float64](request, "replicas")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			replicas := int32(replicasFloat)
			if float64(replicas) != replicasFloat {
				return mcp.NewToolResultError("replicas must be an integer"), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			// Get current deployment
			deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get deployment: %v", err)), nil
			}

			// Update replicas
			deployment.Spec.Replicas = &replicas

			// Update the deployment
			updatedDeployment, err := client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to scale deployment: %v", err)), nil
			}

			r, err := json.Marshal(updatedDeployment)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
