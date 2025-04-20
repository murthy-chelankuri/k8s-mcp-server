package k8s

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMCPRequest is a helper function to create a MCP request with the given arguments.
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

func TestOptionalParamOK(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		paramName   string
		expectedVal interface{}
		expectedOk  bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "present and correct type (string)",
			args:        map[string]interface{}{"myParam": "hello"},
			paramName:   "myParam",
			expectedVal: "hello",
			expectedOk:  true,
			expectError: false,
		},
		{
			name:        "present and correct type (bool)",
			args:        map[string]interface{}{"myParam": true},
			paramName:   "myParam",
			expectedVal: true,
			expectedOk:  true,
			expectError: false,
		},
		{
			name:        "present and correct type (number)",
			args:        map[string]interface{}{"myParam": float64(123)},
			paramName:   "myParam",
			expectedVal: float64(123),
			expectedOk:  true,
			expectError: false,
		},
		{
			name:        "present but wrong type (string expected, got bool)",
			args:        map[string]interface{}{"myParam": true},
			paramName:   "myParam",
			expectedVal: "",   // Zero value for string
			expectedOk:  true, // ok is true because param exists
			expectError: true,
			errorMsg:    "parameter myParam is not of type string, is bool",
		},
		{
			name:        "present but wrong type (bool expected, got string)",
			args:        map[string]interface{}{"myParam": "true"},
			paramName:   "myParam",
			expectedVal: false, // Zero value for bool
			expectedOk:  true,  // ok is true because param exists
			expectError: true,
			errorMsg:    "parameter myParam is not of type bool, is string",
		},
		{
			name:        "parameter not present",
			args:        map[string]interface{}{"anotherParam": "value"},
			paramName:   "myParam",
			expectedVal: "", // Zero value for string
			expectedOk:  false,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.args)

			// Test with string type assertion
			if _, isString := tc.expectedVal.(string); isString || tc.errorMsg == "parameter myParam is not of type string, is bool" {
				val, ok, err := OptionalParamOK[string](request, tc.paramName)
				if tc.expectError {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMsg)
					assert.Equal(t, tc.expectedOk, ok)   // Check ok even on error
					assert.Equal(t, tc.expectedVal, val) // Check zero value on error
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expectedOk, ok)
					assert.Equal(t, tc.expectedVal, val)
				}
			}

			// Test with bool type assertion
			if _, isBool := tc.expectedVal.(bool); isBool || tc.errorMsg == "parameter myParam is not of type bool, is string" {
				val, ok, err := OptionalParamOK[bool](request, tc.paramName)
				if tc.expectError {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMsg)
					assert.Equal(t, tc.expectedOk, ok)   // Check ok even on error
					assert.Equal(t, tc.expectedVal, val) // Check zero value on error
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expectedOk, ok)
					assert.Equal(t, tc.expectedVal, val)
				}
			}

			// Test with float64 type assertion (for number case)
			if _, isFloat := tc.expectedVal.(float64); isFloat {
				val, ok, err := OptionalParamOK[float64](request, tc.paramName)
				if tc.expectError {
					// This case shouldn't happen for float64 in the defined tests
					require.Fail(t, "Unexpected error case for float64")
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expectedOk, ok)
					assert.Equal(t, tc.expectedVal, val)
				}
			}
		})
	}
}

func int32Ptr(val int32) *int32 {
	return &val
}
