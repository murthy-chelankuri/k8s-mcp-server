package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/briankscheong/k8s-mcp-server/pkg/k8s/resourcetypes"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Handler implements the ResourceHandler interface for Pod resources
type Handler struct {
	getClient resourcetypes.GetClientFn
	t         translations.TranslationHelperFunc
}

// NewHandler creates a new Pod resource handler
func NewHandler(getClient resourcetypes.GetClientFn, t translations.TranslationHelperFunc) *Handler {
	return &Handler{
		getClient: getClient,
		t:         t,
	}
}

// RegisterTools registers all Pod resource tools with the provided toolset
func (h *Handler) RegisterTools(toolset *resourcetypes.Toolset) {
	// Register read tools
	getTool, getHandler := h.Get()
	toolset.AddReadTool(getTool, getHandler)

	listTool, listHandler := h.List()
	toolset.AddReadTool(listTool, listHandler)

	logsTool, logsHandler := h.Logs()
	toolset.AddReadTool(logsTool, logsHandler)

	// Register write tools
	deleteTool, deleteHandler := h.Delete()
	toolset.AddWriteTool(deleteTool, deleteHandler)
}

// Get creates a tool to get details of a specific pod
func (h *Handler) Get() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_pod",
			mcp.WithDescription(h.t("TOOL_GET_POD_DESCRIPTION", "Get details of a specific pod")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Pod name"),
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

			pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get pod: %v", err)), nil
			}

			r, err := json.Marshal(pod)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// List creates a tool to list pods in a namespace
func (h *Handler) List() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_pods",
			mcp.WithDescription(h.t("TOOL_LIST_PODS_DESCRIPTION", "List pods in a namespace")),
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

			pods, err := client.CoreV1().Pods(namespace).List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
			}

			r, err := json.Marshal(pods)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// Delete creates a tool to delete a pod
func (h *Handler) Delete() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("delete_pod",
			mcp.WithDescription(h.t("TOOL_DELETE_POD_DESCRIPTION", "Delete a pod")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Pod name"),
			),
			mcp.WithBoolean("gracePeriodSeconds",
				mcp.Description("The duration in seconds before the pod should be deleted"),
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

			gracePeriodSeconds, err := resourcetypes.OptionalParam[int64](request, "gracePeriodSeconds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			deleteOptions := metav1.DeleteOptions{}
			if gracePeriodSeconds > 0 {
				deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
			}

			err = client.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to delete pod: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Pod %s in namespace %s deleted successfully", name, namespace)), nil
		}
}

// Logs creates a tool to get logs from a pod
func (h *Handler) Logs() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_pod_logs",
			mcp.WithDescription(h.t("TOOL_GET_POD_LOGS_DESCRIPTION", "Get logs from a pod")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Pod name"),
			),
			mcp.WithString("container",
				mcp.Description("Container name (optional if pod has only one container)"),
			),
			mcp.WithNumber("tailLines",
				mcp.Description("Number of lines from the end of the logs to show"),
			),
			mcp.WithBoolean("previous",
				mcp.Description("Return previous terminated container logs"),
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

			container, err := resourcetypes.OptionalParam[string](request, "container")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			tailLinesFloat, err := resourcetypes.OptionalParam[float64](request, "tailLines")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			previous, err := resourcetypes.OptionalParam[bool](request, "previous")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := h.getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			podLogOptions := &corev1.PodLogOptions{
				Previous: previous,
			}

			if container != "" {
				podLogOptions.Container = container
			}

			if tailLinesFloat > 0 {
				tailLines := int64(tailLinesFloat)
				podLogOptions.TailLines = &tailLines
			}

			req := client.CoreV1().Pods(namespace).GetLogs(name, podLogOptions)
			logs, err := req.Stream(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
			}
			defer logs.Close()

			logBytes, err := io.ReadAll(logs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to read logs: %v", err)), nil
			}

			return mcp.NewToolResultText(string(logBytes)), nil
		}
}
