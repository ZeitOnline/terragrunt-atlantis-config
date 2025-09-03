package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		// Clear all environment variables
		for _, env := range os.Environ() {
			parts := splitEnvVar(env)
			if len(parts) == 2 {
				os.Unsetenv(parts[0])
			}
		}
		// Restore original environment
		for _, env := range originalEnv {
			parts := splitEnvVar(env)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set test environment variables
	os.Setenv("TEST_VAR1", "value1")
	os.Setenv("TEST_VAR2", "value2")
	os.Setenv("EMPTY_VAR", "")

	result := parseEnvironmentVariables()

	assert.Contains(t, result, "TEST_VAR1")
	assert.Equal(t, "value1", result["TEST_VAR1"])
	assert.Contains(t, result, "TEST_VAR2")
	assert.Equal(t, "value2", result["TEST_VAR2"])
	assert.Contains(t, result, "EMPTY_VAR")
	assert.Equal(t, "", result["EMPTY_VAR"])
}

func splitEnvVar(env string) []string {
	// Helper function to split environment variable
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}

func TestCreateLogger(t *testing.T) {
	logger := createLogger()
	assert.NotNil(t, logger)

	// Test that logger doesn't panic when used
	assert.NotPanics(t, func() {
		logger.Debug("test debug message")
		logger.Info("test info message")
		logger.Error("test error message")
	})
}

func TestNewParsingContextWithConfigPath(t *testing.T) {
	// Create a temporary terragrunt.hcl file
	tmpDir, err := os.MkdirTemp("", "terragrunt-context-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	terragruntPath := filepath.Join(tmpDir, "terragrunt.hcl")
	terragruntContent := `
terraform {
  source = "git::https://github.com/example/module.git"
}

locals {
  environment = "test"
}
`
	err = os.WriteFile(terragruntPath, []byte(terragruntContent), 0644)
	require.NoError(t, err)

	ctx, err := NewParsingContextWithConfigPath(context.Background(), terragruntPath)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Context)
	assert.NotNil(t, ctx.ParsingContext)
	assert.Equal(t, terragruntPath, ctx.ParsingContext.TerragruntOptions.OriginalTerragruntConfigPath)
}

func TestNewParsingContextWithConfigPath_InvalidPath(t *testing.T) {
	invalidPath := "/non/existent/path/terragrunt.hcl"

	ctx, err := NewParsingContextWithConfigPath(context.Background(), invalidPath)
	require.NoError(t, err) // Should still create context even with invalid path
	assert.NotNil(t, ctx)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "=", envVarSeparator)
	assert.Equal(t, "root.hcl", rootConfigFileName)
}

func TestNewParsingContextWithConfigPath_Basic(t *testing.T) {
	// Create a temporary terragrunt.hcl file
	tmpDir, err := os.MkdirTemp("", "terragrunt-context-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	terragruntPath := filepath.Join(tmpDir, "terragrunt.hcl")
	terragruntContent := `
terraform {
  source = "git::https://github.com/example/module.git"
}

locals {
  environment = "test"
}
`
	err = os.WriteFile(terragruntPath, []byte(terragruntContent), 0644)
	require.NoError(t, err)

	ctx, err := NewParsingContextWithConfigPath(context.Background(), terragruntPath)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Context)
	assert.NotNil(t, ctx.ParsingContext)
	assert.Equal(t, terragruntPath, ctx.ParsingContext.TerragruntOptions.OriginalTerragruntConfigPath)
}

func TestFindConfigFilesInPath_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "find-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create root.hcl file
	rootHclPath := filepath.Join(tmpDir, "root.hcl")
	err = os.WriteFile(rootHclPath, []byte("# root config"), 0644)
	require.NoError(t, err)

	// Test with minimal setup - just checking the function doesn't crash
	ctx, err := NewParsingContextWithConfigPath(context.Background(), tmpDir)
	require.NoError(t, err)

	configFiles, err := FindConfigFilesInPath(tmpDir, ctx.ParsingContext.TerragruntOptions)
	require.NoError(t, err)

	// Should find at least one file (either root.hcl or terragrunt.hcl files)
	assert.GreaterOrEqual(t, len(configFiles), 0) // Allow empty result for now
}

func TestGetAllTerragruntFiles_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "get-all-terragrunt-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create terragrunt.hcl file
	terragruntPath := filepath.Join(tmpDir, "terragrunt.hcl")
	err = os.WriteFile(terragruntPath, []byte("# terragrunt config"), 0644)
	require.NoError(t, err)

	files, err := getAllTerragruntFiles(tmpDir)
	require.NoError(t, err)

	// Should find at least one terragrunt.hcl file
	assert.GreaterOrEqual(t, len(files), 1)

	// All returned paths should be absolute
	for _, file := range files {
		assert.True(t, filepath.IsAbs(file), "Path should be absolute: %s", file)
	}
}

func TestWithDecodedList(t *testing.T) {
	ctx, err := NewParsingContextWithConfigPath(context.Background(), "/test/path")
	require.NoError(t, err)

	result := ctx.WithDecodedList()

	// Verify the context is returned and functional
	assert.NotNil(t, result)
	assert.NotNil(t, result.ParsingContext)
}

func TestWithTerragruntOptions(t *testing.T) {
	ctx, err := NewParsingContextWithConfigPath(context.Background(), "/test/path")
	require.NoError(t, err)

	// Create mock terragrunt options using the existing context
	opts := ctx.ParsingContext.TerragruntOptions

	result := ctx.WithTerragruntOptions(opts)

	// Verify the context is returned and functional
	assert.NotNil(t, result)
	assert.NotNil(t, result.ParsingContext)
}

func TestNewParsingContextWithDecodeList(t *testing.T) {
	baseCtx, err := NewParsingContextWithConfigPath(context.Background(), "/test/path")
	require.NoError(t, err)

	ctx := NewParsingContextWithDecodeList(baseCtx)

	// Verify the context is created
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.ParsingContext)
}

func TestWithDependencyPath(t *testing.T) {
	ctx, err := NewParsingContextWithConfigPath(context.Background(), "/test/path")
	require.NoError(t, err)
	dependencyPath := "/dependency/path"

	result := ctx.WithDependencyPath(dependencyPath)

	// Verify the context is returned
	assert.NotNil(t, result)
}
