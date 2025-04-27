package toolsets

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

// Tests for the ResourceRegistry

func TestResourceRegistry(t *testing.T) {
	registry := NewResourceRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry.handlers)

	// Create a mock resource handler
	mockHandler := &mockResourceHandler{}

	// Register the handler
	registry.Register("mock", mockHandler)

	// Verify the handler was registered
	handler, ok := registry.GetHandler("mock")
	assert.True(t, ok)
	assert.Equal(t, mockHandler, handler)

	// Verify GetAllHandlers returns all handlers
	handlers := registry.GetAllHandlers()
	assert.Len(t, handlers, 1)
	assert.Contains(t, handlers, "mock")
}

// Tests for the parameter helper functions

func TestRequiredParam(t *testing.T) {
	// Test with valid parameter
	request := createTestRequest(map[string]interface{}{
		"param": "value",
	})
	value, err := RequiredParam[string](request, "param")
	assert.NoError(t, err)
	assert.Equal(t, "value", value)

	// Test with missing parameter
	_, err = RequiredParam[string](request, "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter")

	// Test with wrong type
	request = createTestRequest(map[string]interface{}{
		"param": 123,
	})
	_, err = RequiredParam[string](request, "param")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not of type")

	// Test with zero value
	request = createTestRequest(map[string]interface{}{
		"param": "",
	})
	_, err = RequiredParam[string](request, "param")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter")
}

func TestOptionalParam(t *testing.T) {
	// Test with valid parameter
	request := createTestRequest(map[string]interface{}{
		"param": "value",
	})
	value, err := OptionalParam[string](request, "param")
	assert.NoError(t, err)
	assert.Equal(t, "value", value)

	// Test with missing parameter
	value, err = OptionalParam[string](request, "missing")
	assert.NoError(t, err)
	assert.Equal(t, "", value)

	// Test with wrong type
	request = createTestRequest(map[string]interface{}{
		"param": 123,
	})
	_, err = OptionalParam[string](request, "param")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not of type")
}

// Helper functions for testing

type mockResourceHandler struct{}

func (m *mockResourceHandler) RegisterTools(toolset *Toolset) {}

func createTestRequest(args map[string]interface{}) mcp.CallToolRequest {
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
