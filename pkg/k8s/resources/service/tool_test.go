package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// Helper function to get text result from tool response
func getTextResult(t *testing.T, result *mcp.CallToolResult) mcp.TextContent {
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	require.Equal(t, "text", result.Content[0].(mcp.TextContent).Type)
	return result.Content[0].(mcp.TextContent)
}

// Helper function to create a fake client
func stubGetClientFn(client kubernetes.Interface) toolsets.GetClientFn {
	return func(ctx context.Context) (kubernetes.Interface, error) {
		return client, nil
	}
}

// Helper function to create a MCP request
func createMCPRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: args,
		},
	}
}

func TestGetService(t *testing.T) {
	// Create test service
	testService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testService)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Get()

	assert.Equal(t, "get_service", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name            string
		client          kubernetes.Interface
		requestArgs     map[string]interface{}
		expectError     bool
		expectedService *corev1.Service
		expectedErrMsg  string
	}{
		{
			name:   "successful service fetch",
			client: fake.NewSimpleClientset(testService),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-service",
			},
			expectError:     false,
			expectedService: testService,
		},
		{
			name:   "service not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent-service",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get service",
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name": "test-service",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
		{
			name:   "missing required param: name",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			_, handlerFn := handler.Get()
			request := createMCPRequest(tc.requestArgs)
			result, err := handlerFn(context.Background(), request)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// If we're expecting an error message in the result
			if tc.expectedErrMsg != "" {
				assert.True(t, result.IsError)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getTextResult(t, result)

			// Unmarshal and verify the returned service
			var returnedService corev1.Service
			err = json.Unmarshal([]byte(textContent.Text), &returnedService)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedService.Name, returnedService.Name)
			assert.Equal(t, tc.expectedService.Namespace, returnedService.Namespace)
			assert.Equal(t, tc.expectedService.Spec.Type, returnedService.Spec.Type)
			assert.Equal(t, tc.expectedService.Spec.Ports[0].Port, returnedService.Spec.Ports[0].Port)
		})
	}
}

func TestListServices(t *testing.T) {
	// Create test services
	testServices := &corev1.ServiceList{
		Items: []corev1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-1",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "test",
					},
					Ports: []corev1.ServicePort{
						{
							Port:     80,
							Protocol: corev1.ProtocolTCP,
						},
					},
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service-2",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "test",
					},
					Ports: []corev1.ServicePort{
						{
							Port:     8080,
							Protocol: corev1.ProtocolTCP,
						},
					},
					Type: corev1.ServiceTypeNodePort,
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testServices.Items[0], &testServices.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_services", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name                string
		client              kubernetes.Interface
		requestArgs         map[string]interface{}
		expectError         bool
		expectedServiceList *corev1.ServiceList
		expectedErrMsg      string
	}{
		{
			name:   "successful services list",
			client: fake.NewSimpleClientset(&testServices.Items[0], &testServices.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:         false,
			expectedServiceList: testServices,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "empty-namespace",
			},
			expectError:         false,
			expectedServiceList: &corev1.ServiceList{},
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testServices.Items[0], &testServices.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace":     "default",
				"labelSelector": "app=test",
			},
			expectError:         false,
			expectedServiceList: testServices,
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"labelSelector": "app=test",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			_, handlerFn := handler.List()
			request := createMCPRequest(tc.requestArgs)
			result, err := handlerFn(context.Background(), request)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// If we're expecting an error message in the result
			if tc.expectedErrMsg != "" {
				assert.True(t, result.IsError)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getTextResult(t, result)

			// Unmarshal and verify the returned service list
			var returnedServiceList corev1.ServiceList
			err = json.Unmarshal([]byte(textContent.Text), &returnedServiceList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedServiceList.Items) == 0 {
				assert.Len(t, returnedServiceList.Items, 0)
				return
			}

			// Otherwise check that the services match
			assert.Len(t, returnedServiceList.Items, len(tc.expectedServiceList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedServiceMap := make(map[string]corev1.Service)
			for _, svc := range tc.expectedServiceList.Items {
				expectedServiceMap[svc.Name] = svc
			}

			for _, svc := range returnedServiceList.Items {
				expectedSvc, exists := expectedServiceMap[svc.Name]
				assert.True(t, exists, "Returned unexpected service: %s", svc.Name)
				assert.Equal(t, expectedSvc.Namespace, svc.Namespace)
				assert.Equal(t, expectedSvc.Spec.Type, svc.Spec.Type)
				assert.Equal(t, expectedSvc.Spec.Ports[0].Port, svc.Spec.Ports[0].Port)
			}
		})
	}
}
