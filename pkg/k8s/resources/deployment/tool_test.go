package deployment

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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

func TestGetDeployment(t *testing.T) {
	// Create test deployment
	replicas := int32(3)
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testDeployment)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Get()

	assert.Equal(t, "get_deployment", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name               string
		client             kubernetes.Interface
		requestArgs        map[string]interface{}
		expectError        bool
		expectedDeployment *appsv1.Deployment
		expectedErrMsg     string
	}{
		{
			name:   "successful deployment fetch",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
			},
			expectError:        false,
			expectedDeployment: testDeployment,
		},
		{
			name:   "deployment not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent-deployment",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get deployment",
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name": "test-deployment",
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

			// Unmarshal and verify the returned deployment
			var returnedDeployment appsv1.Deployment
			err = json.Unmarshal([]byte(textContent.Text), &returnedDeployment)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDeployment.Name, returnedDeployment.Name)
			assert.Equal(t, tc.expectedDeployment.Namespace, returnedDeployment.Namespace)
			assert.Equal(t, *tc.expectedDeployment.Spec.Replicas, *returnedDeployment.Spec.Replicas)
			assert.Equal(t, tc.expectedDeployment.Status.ReadyReplicas, returnedDeployment.Status.ReadyReplicas)
		})
	}
}

func TestListDeployments(t *testing.T) {
	// Create test deployments
	replicas := int32(3)
	testDeployments := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-1",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 3,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-2",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 2,
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(&testDeployments.Items[0], &testDeployments.Items[1])
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.List()

	assert.Equal(t, "list_deployments", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "fieldSelector")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name                   string
		client                 kubernetes.Interface
		requestArgs            map[string]interface{}
		expectError            bool
		expectedDeploymentList *appsv1.DeploymentList
		expectedErrMsg         string
	}{
		{
			name:   "successful deployments list",
			client: fake.NewSimpleClientset(&testDeployments.Items[0], &testDeployments.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:            false,
			expectedDeploymentList: testDeployments,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "empty-namespace",
			},
			expectError:            false,
			expectedDeploymentList: &appsv1.DeploymentList{},
		},
		{
			name:   "with label selector",
			client: fake.NewSimpleClientset(&testDeployments.Items[0], &testDeployments.Items[1]),
			requestArgs: map[string]interface{}{
				"namespace":     "default",
				"labelSelector": "app=test",
			},
			expectError:            false,
			expectedDeploymentList: testDeployments,
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

			// Unmarshal and verify the returned deployment list
			var returnedDeploymentList appsv1.DeploymentList
			err = json.Unmarshal([]byte(textContent.Text), &returnedDeploymentList)
			require.NoError(t, err)

			// For empty lists, just check the length
			if len(tc.expectedDeploymentList.Items) == 0 {
				assert.Len(t, returnedDeploymentList.Items, 0)
				return
			}

			// Otherwise check that the deployments match
			assert.Len(t, returnedDeploymentList.Items, len(tc.expectedDeploymentList.Items))

			// Create maps to make comparison easier since order isn't guaranteed
			expectedDeploymentMap := make(map[string]appsv1.Deployment)
			for _, deployment := range tc.expectedDeploymentList.Items {
				expectedDeploymentMap[deployment.Name] = deployment
			}

			for _, deployment := range returnedDeploymentList.Items {
				expectedDeployment, exists := expectedDeploymentMap[deployment.Name]
				assert.True(t, exists, "Returned unexpected deployment: %s", deployment.Name)
				assert.Equal(t, expectedDeployment.Namespace, deployment.Namespace)
				assert.Equal(t, expectedDeployment.Status.ReadyReplicas, deployment.Status.ReadyReplicas)
			}
		})
	}
}

func TestScaleDeployment(t *testing.T) {
	// Create test deployment
	replicas := int32(3)
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testDeployment)
	handler := NewHandler(stubGetClientFn(fakeClient), translations.NullTranslationHelper)
	tool, _ := handler.Scale()

	assert.Equal(t, "scale_deployment", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "replicas")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name", "replicas"})

	tests := []struct {
		name             string
		client           kubernetes.Interface
		requestArgs      map[string]interface{}
		expectError      bool
		expectedReplicas int32
		expectedErrMsg   string
	}{
		{
			name:   "successful scale up",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
				"replicas":  float64(5),
			},
			expectError:      false,
			expectedReplicas: 5,
		},
		{
			name:   "successful scale down",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
				"replicas":  float64(1),
			},
			expectError:      false,
			expectedReplicas: 1,
		},
		{
			name:   "deployment not found",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "non-existent-deployment",
				"replicas":  float64(5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "failed to get deployment",
		},
		{
			name:   "missing required param: namespace",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": float64(5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
		{
			name:   "missing required param: name",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"replicas":  float64(5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: name",
		},
		{
			name:   "missing required param: replicas",
			client: fake.NewSimpleClientset(),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: replicas",
		},
		{
			name:   "non-integer replicas",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
				"replicas":  float64(2.5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "replicas must be an integer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			_, handlerFn := handler.Scale()
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

			// Unmarshal and verify the returned deployment
			var returnedDeployment appsv1.Deployment
			err = json.Unmarshal([]byte(textContent.Text), &returnedDeployment)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedReplicas, *returnedDeployment.Spec.Replicas)
		})
	}
}
