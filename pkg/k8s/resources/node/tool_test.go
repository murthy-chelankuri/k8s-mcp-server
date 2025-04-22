package node

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

func TestGetNode(t *testing.T) {
	// Create test node
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Labels: map[string]string{
				"kubernetes.io/hostname": "test-node",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testNode)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Get()

	assert.Equal(t, "get_node", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"name"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedNode   *corev1.Node
		expectedErrMsg string
	}{
		{
			name:   "successful node fetch",
			client: fake.NewSimpleClientset(testNode),
			requestArgs: map[string]interface{}{
				"name": "test-node",
			},
			expectError:  false,
			expectedNode: testNode,
		},
		{
			name:   "node not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name": "non-existent-node",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get node",
		},
		{
			name:           "missing required param: name",
			client:         fake.NewSimpleClientset(),
			requestArgs:    map[string]interface{}{},
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

			// Unmarshal and verify the returned node
			var returnedNode corev1.Node
			err = json.Unmarshal([]byte(textContent.Text), &returnedNode)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedNode.Name, returnedNode.Name)
			assert.Equal(t, tc.expectedNode.Status.Conditions[0].Type, returnedNode.Status.Conditions[0].Type)
			assert.Equal(t, tc.expectedNode.Status.Conditions[0].Status, returnedNode.Status.Conditions[0].Status)
		})
	}
}

func TestListNodes(t *testing.T) {
	// Create test nodes
	testNodes := &corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node-1",
					Labels: map[string]string{
						"kubernetes.io/hostname":                "test-node-1",
						"node-role.kubernetes.io/control-plane": "",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node-2",
					Labels: map[string]string{
						"kubernetes.io/hostname": "test-node-2",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testNodes.Items[0], &testNodes.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_nodes", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.Empty(t, tool.InputSchema.Required)

	tests := []struct {
		name             string
		client           kubernetes.Interface
		requestArgs      map[string]interface{}
		expectError      bool
		expectedNodeList *corev1.NodeList
		expectedErrMsg   string
	}{
		{
			name:             "successful nodes list",
			client:           fake.NewSimpleClientset(&testNodes.Items[0], &testNodes.Items[1]),
			requestArgs:      map[string]interface{}{},
			expectError:      false,
			expectedNodeList: testNodes,
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testNodes.Items[0], &testNodes.Items[1]),
			requestArgs: map[string]interface{}{
				"labelSelector": "node-role.kubernetes.io/control-plane",
			},
			expectError: false,
			expectedNodeList: &corev1.NodeList{
				Items: []corev1.Node{
					testNodes.Items[0],
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

			// Unmarshal and verify the returned node list
			var returnedNodeList corev1.NodeList
			err = json.Unmarshal([]byte(textContent.Text), &returnedNodeList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedNodeList.Items) == 0 {
				assert.Len(t, returnedNodeList.Items, 0)
				return
			}

			// Otherwise check that the nodes match
			assert.Len(t, returnedNodeList.Items, len(tc.expectedNodeList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedNodeMap := make(map[string]corev1.Node)
			for _, node := range tc.expectedNodeList.Items {
				expectedNodeMap[node.Name] = node
			}

			for _, node := range returnedNodeList.Items {
				expectedNode, exists := expectedNodeMap[node.Name]
				assert.True(t, exists, "Returned unexpected node: %s", node.Name)
				assert.Equal(t, expectedNode.Status.Conditions[0].Type, node.Status.Conditions[0].Type)
				assert.Equal(t, expectedNode.Status.Conditions[0].Status, node.Status.Conditions[0].Status)
			}
		})
	}
}
