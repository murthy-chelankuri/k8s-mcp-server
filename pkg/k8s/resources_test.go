package k8s

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// Helper function to get text result from tool response
func getK8sTextResult(t *testing.T, result *mcp.CallToolResult) mcp.TextContent {
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	require.Equal(t, "text", result.Content[0].(mcp.TextContent).Type)
	return result.Content[0].(mcp.TextContent)
}

func Test_GetPod(t *testing.T) {
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
	tool, _ := GetPod(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

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
			_, handler := GetPod(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

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

func Test_ListPods(t *testing.T) {
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
	tool, _ := ListPods(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

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
			_, handler := ListPods(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

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

func Test_GetDeployment(t *testing.T) {
	// Create test deployment
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
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
	tool, _ := GetDeployment(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

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
			_, handler := GetDeployment(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned deployment
			var returnedDeployment appsv1.Deployment
			err = json.Unmarshal([]byte(textContent.Text), &returnedDeployment)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDeployment.Name, returnedDeployment.Name)
			assert.Equal(t, tc.expectedDeployment.Namespace, returnedDeployment.Namespace)
			assert.Equal(t, *tc.expectedDeployment.Spec.Replicas, *returnedDeployment.Spec.Replicas)
			assert.Equal(t, tc.expectedDeployment.Spec.Template.Spec.Containers[0].Name,
				returnedDeployment.Spec.Template.Spec.Containers[0].Name)
			assert.Equal(t, tc.expectedDeployment.Spec.Template.Spec.Containers[0].Image,
				returnedDeployment.Spec.Template.Spec.Containers[0].Image)
			assert.Equal(t, tc.expectedDeployment.Status.ReadyReplicas, returnedDeployment.Status.ReadyReplicas)
		})
	}
}

func Test_ScaleDeployment(t *testing.T) {
	// Create test deployment
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
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
	}

	// Create a scaled deployment for the expected result
	scaledDeployment := testDeployment.DeepCopy()
	scaledDeployment.Spec.Replicas = int32Ptr(5)

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testDeployment)
	tool, _ := ScaleDeployment(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "scale_deployment", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "replicas")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name", "replicas"})

	tests := []struct {
		name                 string
		client               kubernetes.Interface
		requestArgs          map[string]interface{}
		expectError          bool
		expectedDeployment   *appsv1.Deployment
		expectedReplicaCount int32
		expectedErrMsg       string
	}{
		{
			name:   "successful deployment scaling",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-deployment",
				"replicas":  float64(5),
			},
			expectError:          false,
			expectedDeployment:   scaledDeployment,
			expectedReplicaCount: 5,
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
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"name":     "test-deployment",
				"replicas": float64(5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
		{
			name:   "missing required param: name",
			client: fake.NewSimpleClientset(testDeployment),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"replicas":  float64(5),
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: name",
		},
		{
			name:   "missing required param: replicas",
			client: fake.NewSimpleClientset(testDeployment),
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
			_, handler := ScaleDeployment(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned deployment
			var returnedDeployment appsv1.Deployment
			err = json.Unmarshal([]byte(textContent.Text), &returnedDeployment)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDeployment.Name, returnedDeployment.Name)
			assert.Equal(t, tc.expectedDeployment.Namespace, returnedDeployment.Namespace)
			assert.Equal(t, tc.expectedReplicaCount, *returnedDeployment.Spec.Replicas)
		})
	}
}

func Test_GetService(t *testing.T) {
	// Create test service
	testService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testService)
	tool, _ := GetService(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "get_service", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedSvc    *corev1.Service
		expectedErrMsg string
	}{
		{
			name:   "successful service retrieval",
			client: fake.NewSimpleClientset(testService),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-service",
			},
			expectError: false,
			expectedSvc: testService,
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
			client: fake.NewSimpleClientset(testService),
			requestArgs: map[string]interface{}{
				"name": "test-service",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
		{
			name:   "missing required param: name",
			client: fake.NewSimpleClientset(testService),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := GetService(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned service
			var returnedService corev1.Service
			err = json.Unmarshal([]byte(textContent.Text), &returnedService)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedSvc.Name, returnedService.Name)
			assert.Equal(t, tc.expectedSvc.Namespace, returnedService.Namespace)
			assert.Equal(t, len(tc.expectedSvc.Spec.Ports), len(returnedService.Spec.Ports))
		})
	}
}

func Test_ListServices(t *testing.T) {
	// Create test services
	testService1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-1",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test-1",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	testService2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-2",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test-2",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(9090),
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testService1, testService2)
	tool, _ := ListServices(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "list_services", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedCount  int
		expectedErrMsg string
	}{
		{
			name:   "successful services list",
			client: fake.NewSimpleClientset(testService1, testService2),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(testService1, testService2),
			requestArgs: map[string]interface{}{
				"namespace": "non-existent",
			},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:           "missing required param: namespace",
			client:         fake.NewSimpleClientset(testService1, testService2),
			requestArgs:    map[string]interface{}{},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := ListServices(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned services list
			var serviceList corev1.ServiceList
			err = json.Unmarshal([]byte(textContent.Text), &serviceList)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(serviceList.Items))
		})
	}
}

func Test_GetConfigMap(t *testing.T) {
	// Create test configmap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testConfigMap)
	tool, _ := GetConfigMap(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "get_configmap", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace", "name"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedCM     *corev1.ConfigMap
		expectedErrMsg string
	}{
		{
			name:   "successful configmap retrieval",
			client: fake.NewSimpleClientset(testConfigMap),
			requestArgs: map[string]interface{}{
				"namespace": "default",
				"name":      "test-configmap",
			},
			expectError: false,
			expectedCM:  testConfigMap,
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
			client: fake.NewSimpleClientset(testConfigMap),
			requestArgs: map[string]interface{}{
				"name": "test-configmap",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
		{
			name:   "missing required param: name",
			client: fake.NewSimpleClientset(testConfigMap),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := GetConfigMap(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned configmap
			var returnedConfigMap corev1.ConfigMap
			err = json.Unmarshal([]byte(textContent.Text), &returnedConfigMap)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCM.Name, returnedConfigMap.Name)
			assert.Equal(t, tc.expectedCM.Namespace, returnedConfigMap.Namespace)
			assert.Equal(t, len(tc.expectedCM.Data), len(returnedConfigMap.Data))
		})
	}
}

func Test_ListConfigMaps(t *testing.T) {
	// Create test configmaps
	testConfigMap1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap-1",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	testConfigMap2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap-2",
			Namespace: "default",
		},
		Data: map[string]string{
			"key2": "value2",
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testConfigMap1, testConfigMap2)
	tool, _ := ListConfigMaps(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "list_configmaps", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"namespace"})

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedCount  int
		expectedErrMsg string
	}{
		{
			name:   "successful configmaps list",
			client: fake.NewSimpleClientset(testConfigMap1, testConfigMap2),
			requestArgs: map[string]interface{}{
				"namespace": "default",
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:   "empty namespace",
			client: fake.NewSimpleClientset(testConfigMap1, testConfigMap2),
			requestArgs: map[string]interface{}{
				"namespace": "non-existent",
			},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:           "missing required param: namespace",
			client:         fake.NewSimpleClientset(testConfigMap1, testConfigMap2),
			requestArgs:    map[string]interface{}{},
			expectError:    false, // Error is returned in tool result
			expectedErrMsg: "missing required parameter: namespace",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := ListConfigMaps(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned configmaps list
			var configMapList corev1.ConfigMapList
			err = json.Unmarshal([]byte(textContent.Text), &configMapList)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(configMapList.Items))
		})
	}
}

func Test_ListNamespaces(t *testing.T) {
	// Create test namespaces
	testNamespace1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace-1",
		},
	}

	testNamespace2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace-2",
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testNamespace1, testNamespace2)
	tool, _ := ListNamespaces(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "list_namespaces", tool.Name)
	assert.NotEmpty(t, tool.Description)
	// This tool shouldn't require any parameters since it lists all namespaces
	assert.Equal(t, 0, len(tool.InputSchema.Required))

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedCount  int
		expectedErrMsg string
	}{
		{
			name:          "successful namespaces list",
			client:        fake.NewSimpleClientset(testNamespace1, testNamespace2),
			requestArgs:   map[string]interface{}{},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "no namespaces",
			client:        fake.NewSimpleClientset(),
			requestArgs:   map[string]interface{}{},
			expectError:   false,
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := ListNamespaces(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned namespaces list
			var namespaceList corev1.NamespaceList
			err = json.Unmarshal([]byte(textContent.Text), &namespaceList)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(namespaceList.Items))
		})
	}
}

func Test_ListNodes(t *testing.T) {
	// Create test nodes
	testNode1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node-1",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.1",
				},
			},
		},
	}

	testNode2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node-2",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.2",
				},
			},
		},
	}

	// Verify tool definition
	fakeClient := fake.NewSimpleClientset(testNode1, testNode2)
	tool, _ := ListNodes(stubGetClientFn(fakeClient), translations.NullTranslationHelper)

	assert.Equal(t, "list_nodes", tool.Name)
	assert.NotEmpty(t, tool.Description)
	// This tool shouldn't require any parameters since it lists all nodes
	assert.Equal(t, 0, len(tool.InputSchema.Required))

	tests := []struct {
		name           string
		client         kubernetes.Interface
		requestArgs    map[string]interface{}
		expectError    bool
		expectedCount  int
		expectedErrMsg string
	}{
		{
			name:          "successful nodes list",
			client:        fake.NewSimpleClientset(testNode1, testNode2),
			requestArgs:   map[string]interface{}{},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "no nodes",
			client:        fake.NewSimpleClientset(),
			requestArgs:   map[string]interface{}{},
			expectError:   false,
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handler := ListNodes(stubGetClientFn(tc.client), translations.NullTranslationHelper)
			request := createMCPRequest(tc.requestArgs)
			result, err := handler(context.Background(), request)

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
				textContent := getK8sTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				return
			}

			// Otherwise we're expecting a successful result
			assert.False(t, result.IsError)
			textContent := getK8sTextResult(t, result)

			// Unmarshal and verify the returned nodes list
			var nodeList corev1.NodeList
			err = json.Unmarshal([]byte(textContent.Text), &nodeList)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(nodeList.Items))
		})
	}
}
