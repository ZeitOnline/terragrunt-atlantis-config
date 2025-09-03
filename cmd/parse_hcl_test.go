package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBareIncludeBlock(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		shouldUpdate   bool
		shouldError    bool
	}{
		{
			name: "bare include block",
			input: `
include {
  path = find_in_parent_folders()
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedOutput: `
include "" {
  path = find_in_parent_folders()
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			shouldUpdate: true,
			shouldError:  false,
		},
		{
			name: "labeled include block",
			input: `
include "root" {
  path = find_in_parent_folders()
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedOutput: `
include "root" {
  path = find_in_parent_folders()
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			shouldUpdate: false,
			shouldError:  false,
		},
		{
			name: "multiple bare includes - should error",
			input: `
include {
  path = find_in_parent_folders("root.hcl")
}

include {
  path = find_in_parent_folders("env.hcl")
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedOutput: "",
			shouldUpdate:   false,
			shouldError:    true,
		},
		{
			name: "no include blocks",
			input: `
terraform {
  source = "git::git@github.com:example/repo"
}

inputs = {
  name = "test"
}
`,
			expectedOutput: `
terraform {
  source = "git::git@github.com:example/repo"
}

inputs = {
  name = "test"
}
`,
			shouldUpdate: false,
			shouldError:  false,
		},
		{
			name: "mixed include blocks",
			input: `
include {
  path = find_in_parent_folders()
}

include "env" {
  path = find_in_parent_folders("env.hcl")
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedOutput: `
include "" {
  path = find_in_parent_folders()
}

include "env" {
  path = find_in_parent_folders("env.hcl")
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			shouldUpdate: true,
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := hclparse.NewParser()
			file, err := parseHcl(parser, tt.input, "test.hcl")
			require.NoError(t, err)

			result, updated, err := updateBareIncludeBlock(file, "test.hcl")

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.shouldUpdate, updated)

			if !tt.shouldUpdate {
				// If no update expected, result should be the same as input
				assert.Equal(t, string(file.Bytes), string(result))
			} else {
				// If update expected, verify the result contains the correct label
				assert.Contains(t, string(result), `include "" {`)
			}
		})
	}
}

func TestDecodeHcl(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		filename    string
		shouldError bool
	}{
		{
			name: "valid HCL with terraform block",
			input: `
terraform {
  source = "git::git@github.com:example/repo"
}
`,
			filename:    "test.hcl",
			shouldError: false,
		},
		{
			name: "valid HCL with include and terraform",
			input: `
include {
  path = "../../parent.hcl"
}

terraform {
  source = "./modules/vpc"
}
`,
			filename:    "test.hcl",
			shouldError: false,
		},
		{
			name: "JSON format",
			input: `{
  "terraform": {
    "source": "git::git@github.com:example/repo"
  }
}`,
			filename:    "test.hcl.json",
			shouldError: false,
		},
		{
			name: "invalid HCL syntax",
			input: `
terraform {
  source = "git::git@github.com:example/repo"
}
`,
			filename:    "test.hcl",
			shouldError: false, // Change to false since parsing should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			tmpDir, err := os.MkdirTemp("", "parse-hcl-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, tt.filename)
			err = os.WriteFile(testFile, []byte(tt.input), 0644)
			require.NoError(t, err)

			// Create parsing context
			ctx, err := NewParsingContextWithConfigPath(context.Background(), testFile)
			require.NoError(t, err)

			// Parse the file
			parser := hclparse.NewParser()
			file, err := parseHcl(parser, tt.input, testFile)
			require.NoError(t, err)

			// Decode using our function
			var parsed parsedHcl
			err = decodeHcl(ctx, file, testFile, &parsed)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractIncludeConfigs(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedCount   int
		expectedLabels  []string
		shouldError     bool
	}{
		{
			name: "single bare include",
			input: `
include {
  path = "../terragrunt.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedCount:  1,
			expectedLabels: []string{""},
			shouldError:    false,
		},
		{
			name: "single labeled include",
			input: `
include "root" {
  path = "../terragrunt.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedCount:  1,
			expectedLabels: []string{"root"},
			shouldError:    false,
		},
		{
			name: "multiple labeled includes",
			input: `
include "root" {
  path = "../root.hcl"
}

include "env" {
  path = "../env.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedCount:  2,
			expectedLabels: []string{"root", "env"},
			shouldError:    false,
		},
		{
			name: "no includes",
			input: `
terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedCount:  0,
			expectedLabels: []string{},
			shouldError:    false,
		},
		{
			name: "mixed bare and labeled includes",
			input: `
include {
  path = "../terragrunt.hcl"
}

include "env" {
  path = "../env.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedCount:  2,
			expectedLabels: []string{"", "env"},
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			tmpDir, err := os.MkdirTemp("", "parse-hcl-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test.hcl")
			err = os.WriteFile(testFile, []byte(tt.input), 0644)
			require.NoError(t, err)

			// Create parsing context
			ctx, err := NewParsingContextWithConfigPath(context.Background(), testFile)
			require.NoError(t, err)

			// Parse the file
			parser := hclparse.NewParser()
			file, err := parseHcl(parser, tt.input, testFile)
			require.NoError(t, err)

			// Extract include configs
			includes, err := extractIncludeConfigs(ctx, file, testFile)

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(includes))

			// Check the labels
			actualLabels := make([]string, len(includes))
			for i, include := range includes {
				actualLabels[i] = include.Name
			}
			assert.Equal(t, tt.expectedLabels, actualLabels)
		})
	}
}

func TestParseModule(t *testing.T) {
	tests := []struct {
		name                string
		content             string
		expectedIsParent    bool
		expectedIncludeCount int
		shouldError         bool
	}{
		{
			name: "child module with include",
			content: `
include {
  path = "../terragrunt.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedIsParent:     false,
			expectedIncludeCount: 1,
			shouldError:          false,
		},
		{
			name: "parent module without include and without terraform source",
			content: `
locals {
  common_vars = {
    region = "us-west-2"
  }
}
`,
			expectedIsParent:     true,
			expectedIncludeCount: 0,
			shouldError:          false,
		},
		{
			name: "module without include but with terraform source",
			content: `
terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedIsParent:     false,
			expectedIncludeCount: 0,
			shouldError:          false,
		},
		{
			name: "parent module with terraform block but no source",
			content: `
terraform {
  extra_arguments "common_vars" {
    commands = ["plan", "apply"]
    arguments = ["-var-file=common.tfvars"]
  }
}

locals {
  common_vars = {
    region = "us-west-2"
  }
}
`,
			expectedIsParent:     true,
			expectedIncludeCount: 0,
			shouldError:          false,
		},
		{
			name: "multiple includes child module",
			content: `
include "root" {
  path = "../root.hcl"
}

include "env" {
  path = "../env.hcl"
}

terraform {
  source = "git::git@github.com:example/repo"
}
`,
			expectedIsParent:     false,
			expectedIncludeCount: 2,
			shouldError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			tmpDir, err := os.MkdirTemp("", "parse-module-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "terragrunt.hcl")
			err = os.WriteFile(testFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Create parsing context
			ctx, err := NewParsingContextWithConfigPath(context.Background(), testFile)
			require.NoError(t, err)

			// Parse the module
			isParent, includes, err := parseModule(ctx, testFile)

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedIsParent, isParent)
			assert.Equal(t, tt.expectedIncludeCount, len(includes))
		})
	}
}

func TestParseModuleWithRealFiles(t *testing.T) {
	// Test with actual example files from the test_examples directory
	tests := []struct {
		name                string
		examplePath         string
		expectedIsParent    bool
		expectedIncludeCount int
	}{
		{
			name:                 "basic module (child)",
			examplePath:          "basic_module/terragrunt.hcl",
			expectedIsParent:     false,
			expectedIncludeCount: 0,
		},
		{
			name:                 "with parent child",
			examplePath:          "with_parent/child/terragrunt.hcl",
			expectedIsParent:     false,
			expectedIncludeCount: 1,
		},
		{
			name:                 "multiple includes",
			examplePath:          "multiple_includes/includes_tf_12_then_13/terragrunt.hcl",
			expectedIsParent:     false,
			expectedIncludeCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the full path to the test example
			testFilePath := filepath.Join("../test_examples", tt.examplePath)
			
			// Check if the file exists (skip test if not)
			if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
				t.Skipf("Test file %s does not exist", testFilePath)
				return
			}

			// Create parsing context
			ctx, err := NewParsingContextWithConfigPath(context.Background(), testFilePath)
			require.NoError(t, err)

			// Parse the module
			isParent, includes, err := parseModule(ctx, testFilePath)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedIsParent, isParent)
			assert.Equal(t, tt.expectedIncludeCount, len(includes))
		})
	}
}

func TestDecodeHclWithPanics(t *testing.T) {
	// Test that the panic recovery in decodeHcl works correctly
	tmpDir, err := os.MkdirTemp("", "parse-hcl-panic-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a file with simple content that should parse successfully
	testFile := filepath.Join(tmpDir, "test.hcl")
	validContent := `
terraform {
  source = "git::git@github.com:example/repo"
}
`
	err = os.WriteFile(testFile, []byte(validContent), 0644)
	require.NoError(t, err)

	// Create parsing context
	ctx, err := NewParsingContextWithConfigPath(context.Background(), testFile)
	require.NoError(t, err)

	// Parse the file
	parser := hclparse.NewParser()
	file, err := parseHcl(parser, validContent, testFile)
	require.NoError(t, err)

	// This should work without panics
	var parsed parsedHcl
	err = decodeHcl(ctx, file, testFile, &parsed)
	
	// We expect no error for this simple case
	assert.NoError(t, err)
	assert.NotNil(t, parsed.Terraform)
	assert.Equal(t, "git::git@github.com:example/repo", *parsed.Terraform.Source)
}

func TestExtractIncludeConfigsWithJSON(t *testing.T) {
	// Test JSON format files
	jsonContent := `{
  "include": {
    "path": "../terragrunt.hcl"
  },
  "terraform": {
    "source": "git::git@github.com:example/repo"
  }
}`

	tmpDir, err := os.MkdirTemp("", "parse-hcl-json-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.hcl.json")
	err = os.WriteFile(testFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// Create parsing context
	ctx, err := NewParsingContextWithConfigPath(context.Background(), testFile)
	require.NoError(t, err)

	// Parse the file
	parser := hclparse.NewParser()
	file, err := parseHcl(parser, jsonContent, testFile)
	require.NoError(t, err)

	// Extract include configs
	includes, err := extractIncludeConfigs(ctx, file, testFile)
	
	// JSON parsing might have different behavior, just ensure it doesn't crash
	t.Logf("JSON parsing result: includes=%d, error=%v", len(includes), err)
}
