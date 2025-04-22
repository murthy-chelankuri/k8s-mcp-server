package namespace

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

func TestListNamespaces(t *testing.T) {
	// Create test namespaces
	testNamespaces := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
					Labels: map[string]string{
						"kubernetes.io/metadata.name": "default",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
					Labels: map[string]string{
						"kubernetes.io/metadata.name": "kube-system",
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testNamespaces.Items[0], &testNamespaces.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_namespaces", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.Empty(t, tool.InputSchema.Required)

	tests := []struct {
		name                  string
		client                kubernetes.Interface
		requestArgs           map[string]interface{}
		expectError           bool
		expectedNamespaceList *corev1.NamespaceList
		expectedErrMsg        string
	}{
		{
			name:                  "successful namespaces list",
			client:                fake.NewSimpleClientset(&testNamespaces.Items[0], &testNamespaces.Items[1]),
			requestArgs:           map[string]interface{}{},
			expectError:           false,
			expectedNamespaceList: testNamespaces,
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testNamespaces.Items[0], &testNamespaces.Items[1]),
			requestArgs: map[string]interface{}{
				"labelSelector": "kubernetes.io/metadata.name=default",
			},
			expectError: false,
			expectedNamespaceList: &corev1.NamespaceList{
				Items: []corev1.Namespace{
					testNamespaces.Items[0],
				},
			},
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

			// Unmarshal and verify the returned namespace list
			var returnedNamespaceList corev1.NamespaceList
			err = json.Unmarshal([]byte(textContent.Text), &returnedNamespaceList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedNamespaceList.Items) == 0 {
				assert.Len(t, returnedNamespaceList.Items, 0)
				return
			}

			// Otherwise check that the namespaces match
			assert.Len(t, returnedNamespaceList.Items, len(tc.expectedNamespaceList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedNamespaceMap := make(map[string]corev1.Namespace)
			for _, ns := range tc.expectedNamespaceList.Items {
				expectedNamespaceMap[ns.Name] = ns
			}

			for _, ns := range returnedNamespaceList.Items {
				expectedNS, exists := expectedNamespaceMap[ns.Name]
				assert.True(t, exists, "Returned unexpected namespace: %s", ns.Name)
				assert.Equal(t, expectedNS.Status.Phase, ns.Status.Phase)
			}
		})
	}
}
