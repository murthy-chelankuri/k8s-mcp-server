package pod

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

func TestGetPod(t *testing.T) {
	// Create test pod
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testPod)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Get()

	assert.Equal(t, "get_pod", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedPod    *corev1.Pod
		expectedErrMsg string
	}{
		{
			name:   "successful pod fetch",
			client: fake.NewSimpleClientset(testPod),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-pod",
			},
			expectError: false,
			expectedPod: testPod,
		},
		{
			name:   "pod not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent-pod",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get pod",
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name": "test-pod",
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

			// Unmarshal and verify the returned pod
			var returnedPod corev1.Pod
			err = json.Unmarshal([]byte(textContent.Text), &returnedPod)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPod.Name, returnedPod.Name)
			assert.Equal(t, tc.expectedPod.Namespace, returnedPod.Namespace)
			assert.Equal(t, tc.expectedPod.Spec.Containers[0].Name, returnedPod.Spec.Containers[0].Name)
			assert.Equal(t, tc.expectedPod.Spec.Containers[0].Image, returnedPod.Spec.Containers[0].Image)
			assert.Equal(t, tc.expectedPod.Status.Phase, returnedPod.Status.Phase)
		})
	}
}

func TestListPods(t *testing.T) {
	// Create test pods
	testPods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-1",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-2",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testPods.Items[0], &testPods.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_pods", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name            string
		client          kubernetes.Interface
		requestArgs     map[string]interface{}
		expectError     bool
		expectedPodList *corev1.PodList
		expectedErrMsg  string
	}{
		{
			name:   "successful pods list",
			client: fake.NewSimpleClientset(&testPods.Items[0], &testPods.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:     false,
			expectedPodList: testPods,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "empty-namespace",
			},
			expectError:     false,
			expectedPodList: &corev1.PodList{},
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testPods.Items[0], &testPods.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace":     "default",
				"labelSelector": "app=test",
			},
			expectError:     false,
			expectedPodList: testPods,
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

			// Unmarshal and verify the returned pod list
			var returnedPodList corev1.PodList
			err = json.Unmarshal([]byte(textContent.Text), &returnedPodList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedPodList.Items) == 0 {
				assert.Len(t, returnedPodList.Items, 0)
				return
			}

			// Otherwise check that the pods match
			assert.Len(t, returnedPodList.Items, len(tc.expectedPodList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedPodMap := make(map[string]corev1.Pod)
			for _, pod := range tc.expectedPodList.Items {
				expectedPodMap[pod.Name] = pod
			}

			for _, pod := range returnedPodList.Items {
				expectedPod, exists := expectedPodMap[pod.Name]
				assert.True(t, exists, "Returned unexpected pod: %s", pod.Name)
				assert.Equal(t, expectedPod.Namespace, pod.Namespace)
				assert.Equal(t, expectedPod.Status.Phase, pod.Status.Phase)
			}
		})
	}
}
