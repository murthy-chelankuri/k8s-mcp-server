package toolsets

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestNewToolsetGroup(t *testing.T) {
	tsg := NewToolsetGroup(false)
	if tsg == nil {
		t.Fatal("Expected NewToolsetGroup to return a non-nil pointer")
	}
	if tsg.Toolsets == nil {
		t.Fatal("Expected Toolsets map to be initialized")
	}
	if len(tsg.Toolsets) != 0 {
		t.Fatalf("Expected Toolsets map to be empty, got %d items", len(tsg.Toolsets))
	}
	if tsg.everythingOn {
		t.Fatal("Expected everythingOn to be initialized as false")
	}
}

func TestAddToolset(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Test adding a toolset
	toolset := NewToolset("test-toolset", "A test toolset")
	toolset.Enabled = true
	tsg.AddToolset(toolset)

	// Verify toolset was added correctly
	if len(tsg.Toolsets) != 1 {
		t.Errorf("Expected 1 toolset, got %d", len(tsg.Toolsets))
	}

	toolset, exists := tsg.Toolsets["test-toolset"]
	if !exists {
		t.Fatal("Feature was not added to the map")
	}

	if toolset.Name != "test-toolset" {
		t.Errorf("Expected toolset name to be 'test-toolset', got '%s'", toolset.Name)
	}

	if toolset.Description != "A test toolset" {
		t.Errorf("Expected toolset description to be 'A test toolset', got '%s'", toolset.Description)
	}

	if !toolset.Enabled {
		t.Error("Expected toolset to be enabled")
	}

	// Test adding another toolset
	anotherToolset := NewToolset("another-toolset", "Another test toolset")
	tsg.AddToolset(anotherToolset)

	if len(tsg.Toolsets) != 2 {
		t.Errorf("Expected 2 toolsets, got %d", len(tsg.Toolsets))
	}

	// Test overriding existing toolset
	updatedToolset := NewToolset("test-toolset", "Updated description")
	tsg.AddToolset(updatedToolset)

	toolset = tsg.Toolsets["test-toolset"]
	if toolset.Description != "Updated description" {
		t.Errorf("Expected toolset description to be updated to 'Updated description', got '%s'", toolset.Description)
	}

	if toolset.Enabled {
		t.Error("Expected toolset to be disabled after update")
	}
}

func TestIsEnabled(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Test with non-existent toolset
	if tsg.IsEnabled("non-existent") {
		t.Error("Expected IsEnabled to return false for non-existent toolset")
	}

	// Test with disabled toolset
	disabledToolset := NewToolset("disabled-toolset", "A disabled toolset")
	tsg.AddToolset(disabledToolset)
	if tsg.IsEnabled("disabled-toolset") {
		t.Error("Expected IsEnabled to return false for disabled toolset")
	}

	// Test with enabled toolset
	enabledToolset := NewToolset("enabled-toolset", "An enabled toolset")
	enabledToolset.Enabled = true
	tsg.AddToolset(enabledToolset)
	if !tsg.IsEnabled("enabled-toolset") {
		t.Error("Expected IsEnabled to return true for enabled toolset")
	}
}

func TestEnableFeature(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Test enabling non-existent toolset
	err := tsg.EnableToolset("non-existent")
	if err == nil {
		t.Error("Expected error when enabling non-existent toolset")
	}

	// Test enabling toolset
	testToolset := NewToolset("test-toolset", "A test toolset")
	tsg.AddToolset(testToolset)

	if tsg.IsEnabled("test-toolset") {
		t.Error("Expected toolset to be disabled initially")
	}

	err = tsg.EnableToolset("test-toolset")
	if err != nil {
		t.Errorf("Expected no error when enabling toolset, got: %v", err)
	}

	if !tsg.IsEnabled("test-toolset") {
		t.Error("Expected toolset to be enabled after EnableFeature call")
	}

	// Test enabling already enabled toolset
	err = tsg.EnableToolset("test-toolset")
	if err != nil {
		t.Errorf("Expected no error when enabling already enabled toolset, got: %v", err)
	}
}

func TestEnableToolsets(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Prepare toolsets
	toolset1 := NewToolset("toolset1", "Feature 1")
	toolset2 := NewToolset("toolset2", "Feature 2")
	tsg.AddToolset(toolset1)
	tsg.AddToolset(toolset2)

	// Test enabling multiple toolsets
	err := tsg.EnableToolsets([]string{"toolset1", "toolset2"})
	if err != nil {
		t.Errorf("Expected no error when enabling toolsets, got: %v", err)
	}

	if !tsg.IsEnabled("toolset1") {
		t.Error("Expected toolset1 to be enabled")
	}

	if !tsg.IsEnabled("toolset2") {
		t.Error("Expected toolset2 to be enabled")
	}

	// Test with non-existent toolset in the list
	err = tsg.EnableToolsets([]string{"toolset1", "non-existent"})
	if err == nil {
		t.Error("Expected error when enabling list with non-existent toolset")
	}

	// Test with empty list
	err = tsg.EnableToolsets([]string{})
	if err != nil {
		t.Errorf("Expected no error with empty toolset list, got: %v", err)
	}

	// Test enabling everything through EnableToolsets
	tsg = NewToolsetGroup(false)
	err = tsg.EnableToolsets([]string{"all"})
	if err != nil {
		t.Errorf("Expected no error when enabling 'all', got: %v", err)
	}

	if !tsg.everythingOn {
		t.Error("Expected everythingOn to be true after enabling 'all' via EnableToolsets")
	}
}

func TestEnableEverything(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Add a disabled toolset
	testToolset := NewToolset("test-toolset", "A test toolset")
	tsg.AddToolset(testToolset)

	// Verify it's disabled
	if tsg.IsEnabled("test-toolset") {
		t.Error("Expected toolset to be disabled initially")
	}

	// Enable "all"
	err := tsg.EnableToolsets([]string{"all"})
	if err != nil {
		t.Errorf("Expected no error when enabling 'eall', got: %v", err)
	}

	// Verify everythingOn was set
	if !tsg.everythingOn {
		t.Error("Expected everythingOn to be true after enabling 'eall'")
	}

	// Verify the previously disabled toolset is now enabled
	if !tsg.IsEnabled("test-toolset") {
		t.Error("Expected toolset to be enabled when everythingOn is true")
	}

	// Verify a non-existent toolset is also enabled
	if !tsg.IsEnabled("non-existent") {
		t.Error("Expected non-existent toolset to be enabled when everythingOn is true")
	}
}

func TestIsEnabledWithEverythingOn(t *testing.T) {
	tsg := NewToolsetGroup(false)

	// Enable "everything"
	err := tsg.EnableToolsets([]string{"all"})
	if err != nil {
		t.Errorf("Expected no error when enabling 'all', got: %v", err)
	}

	// Test that any toolset name returns true with IsEnabled
	if !tsg.IsEnabled("some-toolset") {
		t.Error("Expected IsEnabled to return true for any toolset when everythingOn is true")
	}

	if !tsg.IsEnabled("another-toolset") {
		t.Error("Expected IsEnabled to return true for any toolset when everythingOn is true")
	}
}

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

func TestOptionalIntParam(t *testing.T) {
	// Test with valid parameter
	request := createTestRequest(map[string]interface{}{
		"param": float64(123),
	})
	value, err := OptionalIntParam(request, "param")
	assert.NoError(t, err)
	assert.Equal(t, 123, value)

	// Test with missing parameter
	value, err = OptionalIntParam(request, "missing")
	assert.NoError(t, err)
	assert.Equal(t, 0, value)
}

func TestOptionalIntParamWithDefault(t *testing.T) {
	// Test with valid parameter
	request := createTestRequest(map[string]interface{}{
		"param": float64(123),
	})
	value, err := OptionalIntParamWithDefault(request, "param", 456)
	assert.NoError(t, err)
	assert.Equal(t, 123, value)

	// Test with missing parameter
	value, err = OptionalIntParamWithDefault(request, "missing", 456)
	assert.NoError(t, err)
	assert.Equal(t, 456, value)

	// Test with zero value
	request = createTestRequest(map[string]interface{}{
		"param": float64(0),
	})
	value, err = OptionalIntParamWithDefault(request, "param", 456)
	assert.NoError(t, err)
	assert.Equal(t, 456, value)
}

func TestOptionalStringArrayParam(t *testing.T) {
	// Test with string array
	request := createTestRequest(map[string]interface{}{
		"param": []string{"value1", "value2"},
	})
	value, err := OptionalStringArrayParam(request, "param")
	assert.NoError(t, err)
	assert.Equal(t, []string{"value1", "value2"}, value)

	// Test with interface array
	request = createTestRequest(map[string]interface{}{
		"param": []interface{}{"value1", "value2"},
	})
	value, err = OptionalStringArrayParam(request, "param")
	assert.NoError(t, err)
	assert.Equal(t, []string{"value1", "value2"}, value)

	// Test with missing parameter
	value, err = OptionalStringArrayParam(request, "missing")
	assert.NoError(t, err)
	assert.Empty(t, value)

	// Test with wrong type in array
	request = createTestRequest(map[string]interface{}{
		"param": []interface{}{"value1", 123},
	})
	_, err = OptionalStringArrayParam(request, "param")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not of type string")

	// Test with wrong type
	request = createTestRequest(map[string]interface{}{
		"param": 123,
	})
	_, err = OptionalStringArrayParam(request, "param")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not be coerced to []string")
}

func TestPaginationParams(t *testing.T) {
	// Test with valid parameters
	request := createTestRequest(map[string]interface{}{
		"page":    float64(2),
		"perPage": float64(50),
	})
	params, err := OptionalPaginationParams(request)
	assert.NoError(t, err)
	assert.Equal(t, 2, params.Page)
	assert.Equal(t, 50, params.PerPage)

	// Test with default values
	request = createTestRequest(map[string]interface{}{})
	params, err = OptionalPaginationParams(request)
	assert.NoError(t, err)
	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 30, params.PerPage)
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
