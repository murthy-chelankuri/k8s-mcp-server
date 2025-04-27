package resources

import (
	"context"
	"testing"

	"github.com/briankscheong/k8s-mcp-server/pkg/toolsets"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterAllK8sResources(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewSimpleClientset()
	getClient := func(ctx context.Context) (kubernetes.Interface, error) {
		return fakeClient, nil
	}

	// Create a registry
	registry := toolsets.NewK8sResourceRegistry()

	// Register all resources
	RegisterAllK8sResources(registry, getClient, translations.NullTranslationHelper)

	// Verify that all resources are registered
	handlers := registry.GetAllHandlers()
	assert.NotEmpty(t, handlers)
	assert.Contains(t, handlers, "pod")
	assert.Contains(t, handlers, "deployment")
	assert.Contains(t, handlers, "service")
	assert.Contains(t, handlers, "configmap")
	assert.Contains(t, handlers, "namespace")
	assert.Contains(t, handlers, "node")
}

func TestCreateToolset(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewSimpleClientset()
	getClient := func(ctx context.Context) (kubernetes.Interface, error) {
		return fakeClient, nil
	}

	// Create a registry
	registry := toolsets.NewK8sResourceRegistry()

	// Register all resources
	RegisterAllK8sResources(registry, getClient, translations.NullTranslationHelper)

	// Create a toolset
	toolset := CreateToolset(registry, "test_toolset")

	// Verify that the toolset is created
	assert.NotNil(t, toolset)
	assert.Equal(t, "test_toolset", toolset.Name)
	// Check that the toolset has tools
	assert.NotEmpty(t, toolset.Name)
}
