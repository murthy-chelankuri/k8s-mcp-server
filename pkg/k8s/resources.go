package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPod creates a tool to get details of a specific pod.
func GetPod(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_pod",
			mcp.WithDescription(t("TOOL_GET_POD_DESCRIPTION", "Get details of a specific pod")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// ListPods creates a tool to list pods in a namespace.
func ListPods(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_pods",
			mcp.WithDescription(t("TOOL_LIST_PODS_DESCRIPTION", "List pods in a namespace")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// DeletePod creates a tool to delete a pod.
func DeletePod(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("delete_pod",
			mcp.WithDescription(t("TOOL_DELETE_POD_DESCRIPTION", "Delete a pod")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			gracePeriodSeconds, err := OptionalParam[int64](request, "gracePeriodSeconds")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// GetPodLogs creates a tool to get logs from a pod.
func GetPodLogs(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_pod_logs",
			mcp.WithDescription(t("TOOL_GET_POD_LOGS_DESCRIPTION", "Get logs from a pod")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			container, err := OptionalParam[string](request, "container")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			tailLinesFloat, err := OptionalParam[float64](request, "tailLines")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			previous, err := OptionalParam[bool](request, "previous")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// GetDeployment creates a tool to get details of a specific deployment.
func GetDeployment(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_deployment",
			mcp.WithDescription(t("TOOL_GET_DEPLOYMENT_DESCRIPTION", "Get details of a specific deployment")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// ListDeployments creates a tool to list deployments in a namespace.
func ListDeployments(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_deployments",
			mcp.WithDescription(t("TOOL_LIST_DEPLOYMENTS_DESCRIPTION", "List deployments in a namespace")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// ScaleDeployment creates a tool to scale a deployment.
func ScaleDeployment(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("scale_deployment",
			mcp.WithDescription(t("TOOL_SCALE_DEPLOYMENT_DESCRIPTION", "Scale a deployment to a specified number of replicas")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			replicasFloat, err := requiredParam[float64](request, "replicas")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			replicas := int32(replicasFloat)
			if float64(replicas) != replicasFloat {
				return mcp.NewToolResultError("replicas must be an integer"), nil
			}

			client, err := getClient(ctx)
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

// GetService creates a tool to get details of a specific service.
func GetService(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_service",
			mcp.WithDescription(t("TOOL_GET_SERVICE_DESCRIPTION", "Get details of a specific service")),
			mcp.WithString("namespace",
				mcp.Required(),
				mcp.Description("Kubernetes namespace"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Service name"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			service, err := client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get service: %v", err)), nil
			}

			r, err := json.Marshal(service)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// ListServices creates a tool to list services in a namespace.
func ListServices(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_services",
			mcp.WithDescription(t("TOOL_LIST_SERVICES_DESCRIPTION", "List services in a namespace")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			options := metav1.ListOptions{
				FieldSelector: fieldSelector,
				LabelSelector: labelSelector,
			}

			services, err := client.CoreV1().Services(namespace).List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list services: %v", err)), nil
			}

			r, err := json.Marshal(services)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// GetConfigMap creates a tool to get details of a specific configmap.
func GetConfigMap(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_configmap",
			mcp.WithDescription(t("TOOL_GET_CONFIGMAP_DESCRIPTION", "Get details of a specific configmap")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := requiredParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// ListConfigMaps creates a tool to list configmaps in a namespace.
func ListConfigMaps(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_configmaps",
			mcp.WithDescription(t("TOOL_LIST_CONFIGMAPS_DESCRIPTION", "List configmaps in a namespace")),
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
			namespace, err := requiredParam[string](request, "namespace")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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

// ListNamespaces creates a tool to list all namespaces.
func ListNamespaces(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_namespaces",
			mcp.WithDescription(t("TOOL_LIST_NAMESPACES_DESCRIPTION", "List all namespaces")),
			mcp.WithString("fieldSelector",
				mcp.Description("Selector to restrict the list of returned objects by their fields"),
			),
			mcp.WithString("labelSelector",
				mcp.Description("Selector to restrict the list of returned objects by their labels"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
			}

			options := metav1.ListOptions{
				FieldSelector: fieldSelector,
				LabelSelector: labelSelector,
			}

			namespaces, err := client.CoreV1().Namespaces().List(ctx, options)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list namespaces: %v", err)), nil
			}

			r, err := json.Marshal(namespaces)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// ListNodes creates a tool to list all nodes.
func ListNodes(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_nodes",
			mcp.WithDescription(t("TOOL_LIST_NODES_DESCRIPTION", "List all nodes in the cluster")),
			mcp.WithString("fieldSelector",
				mcp.Description("Selector to restrict the list of returned objects by their fields"),
			),
			mcp.WithString("labelSelector",
				mcp.Description("Selector to restrict the list of returned objects by their labels"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			fieldSelector, err := OptionalParam[string](request, "fieldSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			labelSelector, err := OptionalParam[string](request, "labelSelector")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
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
