package configmap

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

func TestGetConfigMap(t *testing.T) {
	// Create test configmap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testConfigMap)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Get()

	assert.Equal(t, "get_configmap", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name              string
		client            kubernetes.Interface
		requestArgs       map[string]interface{}
		expectError       bool
		expectedConfigMap *corev1.ConfigMap
		expectedErrMsg    string
	}{
		{
			name:   "successful configmap fetch",
			client: fake.NewSimpleClientset(testConfigMap),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-configmap",
			},
			expectError:       false,
			expectedConfigMap: testConfigMap,
		},
		{
			name:   "configmap not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent-configmap",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get configmap",
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name": "test-configmap",
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

			// Unmarshal and verify the returned configmap
			var returnedConfigMap corev1.ConfigMap
			err = json.Unmarshal([]byte(textContent.Text), &returnedConfigMap)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedConfigMap.Name, returnedConfigMap.Name)
			assert.Equal(t, tc.expectedConfigMap.Namespace, returnedConfigMap.Namespace)
			assert.Equal(t, tc.expectedConfigMap.Data["key1"], returnedConfigMap.Data["key1"])
			assert.Equal(t, tc.expectedConfigMap.Data["key2"], returnedConfigMap.Data["key2"])
		})
	}
}

func TestListConfigMaps(t *testing.T) {
	// Create test configmaps
	testConfigMaps := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap-1",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Data: map[string]string{
					"key1": "value1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-configmap-2",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Data: map[string]string{
					"key2": "value2",
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testConfigMaps.Items[0], &testConfigMaps.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_configmaps", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name                  string
		client                kubernetes.Interface
		requestArgs           map[string]interface{}
		expectError           bool
		expectedConfigMapList *corev1.ConfigMapList
		expectedErrMsg        string
	}{
		{
			name:   "successful configmaps list",
			client: fake.NewSimpleClientset(&testConfigMaps.Items[0], &testConfigMaps.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:           false,
			expectedConfigMapList: testConfigMaps,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "empty-namespace",
			},
			expectError:           false,
			expectedConfigMapList: &corev1.ConfigMapList{},
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testConfigMaps.Items[0], &testConfigMaps.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace":     "default",
				"labelSelector": "app=test",
			},
			expectError:           false,
			expectedConfigMapList: testConfigMaps,
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

			// Unmarshal and verify the returned configmap list
			var returnedConfigMapList corev1.ConfigMapList
			err = json.Unmarshal([]byte(textContent.Text), &returnedConfigMapList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedConfigMapList.Items) == 0 {
				assert.Len(t, returnedConfigMapList.Items, 0)
				return
			}

			// Otherwise check that the configmaps match
			assert.Len(t, returnedConfigMapList.Items, len(tc.expectedConfigMapList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedConfigMapMap := make(map[string]corev1.ConfigMap)
			for _, cm := range tc.expectedConfigMapList.Items {
				expectedConfigMapMap[cm.Name] = cm
			}

			for _, cm := range returnedConfigMapList.Items {
				expectedCM, exists := expectedConfigMapMap[cm.Name]
				assert.True(t, exists, "Returned unexpected configmap: %s", cm.Name)
				assert.Equal(t, expectedCM.Namespace, cm.Namespace)

				// Check data if it exists
				if len(expectedCM.Data) > 0 {
					for key, value := range expectedCM.Data {
						assert.Equal(t, value, cm.Data[key])
					}
				}
			}
		})
	}
}
