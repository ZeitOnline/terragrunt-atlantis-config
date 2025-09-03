package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestParseHcl(t *testing.T) {
	parser := hclparse.NewParser()

	t.Run("valid HCL", func(t *testing.T) {
		hclContent := `
locals {
  atlantis_workflow = "test"
}
`
		file, err := parseHcl(parser, hclContent, "test.hcl")
		require.NoError(t, err)
		assert.NotNil(t, file)
	})

	t.Run("valid JSON", func(t *testing.T) {
		jsonContent := `{
  "locals": {
    "atlantis_workflow": "test"
  }
}`
		file, err := parseHcl(parser, jsonContent, "test.json")
		require.NoError(t, err)
		assert.NotNil(t, file)
	})

	t.Run("invalid HCL", func(t *testing.T) {
		invalidContent := `locals { invalid syntax`
		file, err := parseHcl(parser, invalidContent, "invalid.hcl")
		assert.Error(t, err)
		assert.Nil(t, file)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidContent := `{ "locals": { invalid json } }`
		file, err := parseHcl(parser, invalidContent, "invalid.json")
		assert.Error(t, err)
		assert.Nil(t, file)
	})
}

func TestMergeResolvedLocals(t *testing.T) {
	boolTrue := true
	boolFalse := false

	parent := ResolvedLocals{
		AtlantisWorkflow:          "parent-workflow",
		TerraformVersion:          "1.0.0",
		AutoPlan:                  &boolTrue,
		Skip:                      &boolFalse,
		ApplyRequirements:         []string{"approved"},
		ExtraAtlantisDependencies: []string{"parent-dep"},
		markedProject:             &boolFalse,
	}

	child := ResolvedLocals{
		AtlantisWorkflow:          "child-workflow",
		TerraformVersion:          "1.1.0",
		AutoPlan:                  &boolFalse,
		Skip:                      &boolTrue,
		ApplyRequirements:         []string{"mergeable"},
		ExtraAtlantisDependencies: []string{"child-dep"},
		markedProject:             &boolTrue,
	}

	result := mergeResolvedLocals(parent, child)

	// Child values should override parent values
	assert.Equal(t, "child-workflow", result.AtlantisWorkflow)
	assert.Equal(t, "1.1.0", result.TerraformVersion)
	assert.Equal(t, &boolFalse, result.AutoPlan)
	assert.Equal(t, &boolTrue, result.Skip)
	assert.Equal(t, []string{"mergeable"}, result.ApplyRequirements)
	assert.Equal(t, &boolTrue, result.markedProject)

	// ExtraAtlantisDependencies should be appended
	assert.Equal(t, []string{"parent-dep", "child-dep"}, result.ExtraAtlantisDependencies)
}

func TestMergeResolvedLocals_EmptyChild(t *testing.T) {
	boolTrue := true
	parent := ResolvedLocals{
		AtlantisWorkflow:          "workflow",
		TerraformVersion:          "1.0.0",
		AutoPlan:                  &boolTrue,
		ApplyRequirements:         []string{"approved"},
		ExtraAtlantisDependencies: []string{"dep"},
	}

	child := ResolvedLocals{}

	result := mergeResolvedLocals(parent, child)

	// Parent values should be preserved when child is empty
	assert.Equal(t, "workflow", result.AtlantisWorkflow)
	assert.Equal(t, "1.0.0", result.TerraformVersion)
	assert.Equal(t, &boolTrue, result.AutoPlan)
	assert.Equal(t, []string{"approved"}, result.ApplyRequirements)
	assert.Equal(t, []string{"dep"}, result.ExtraAtlantisDependencies)
}

func TestResolveLocals(t *testing.T) {
	t.Run("empty locals", func(t *testing.T) {
		result, err := resolveLocals(cty.NilVal)
		require.NoError(t, err)
		assert.Equal(t, ResolvedLocals{}, result)
	})

	t.Run("atlantis_workflow", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_workflow": cty.StringVal("custom-workflow"),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.Equal(t, "custom-workflow", result.AtlantisWorkflow)
	})

	t.Run("atlantis_terraform_version", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_terraform_version": cty.StringVal("1.5.0"),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.Equal(t, "1.5.0", result.TerraformVersion)
	})

	t.Run("atlantis_autoplan", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_autoplan": cty.BoolVal(true),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.NotNil(t, result.AutoPlan)
		assert.True(t, *result.AutoPlan)
	})

	t.Run("atlantis_skip", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_skip": cty.BoolVal(true),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.NotNil(t, result.Skip)
		assert.True(t, *result.Skip)
	})

	t.Run("atlantis_apply_requirements", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_apply_requirements": cty.ListVal([]cty.Value{
				cty.StringVal("approved"),
				cty.StringVal("mergeable"),
			}),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.Equal(t, []string{"approved", "mergeable"}, result.ApplyRequirements)
	})

	t.Run("atlantis_project", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_project": cty.BoolVal(true),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.NotNil(t, result.markedProject)
		assert.True(t, *result.markedProject)
	})

	t.Run("extra_atlantis_dependencies", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"extra_atlantis_dependencies": cty.ListVal([]cty.Value{
				cty.StringVal("../shared/vpc"),
				cty.StringVal("../shared/security"),
			}),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)
		assert.Equal(t, []string{"../shared/vpc", "../shared/security"}, result.ExtraAtlantisDependencies)
	})

	t.Run("extra_atlantis_dependencies with non-string", func(t *testing.T) {
		// Create a list with mixed types that will cause an error during processing
		// First, we need to create the object with the list value
		listVal := cty.TupleVal([]cty.Value{
			cty.StringVal("../shared/vpc"),
			cty.NumberIntVal(123), // Invalid non-string value
		})

		locals := cty.ObjectVal(map[string]cty.Value{
			"extra_atlantis_dependencies": listVal,
		})

		result, err := resolveLocals(locals)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "extra_atlantis_dependencies contains non-string value")
		assert.NotEqual(t, ResolvedLocals{}, result) // Should return partial result
	})

	t.Run("all locals combined", func(t *testing.T) {
		locals := cty.ObjectVal(map[string]cty.Value{
			"atlantis_workflow":          cty.StringVal("custom"),
			"atlantis_terraform_version": cty.StringVal("1.5.0"),
			"atlantis_autoplan":          cty.BoolVal(false),
			"atlantis_skip":              cty.BoolVal(true),
			"atlantis_project":           cty.BoolVal(true),
			"atlantis_apply_requirements": cty.ListVal([]cty.Value{
				cty.StringVal("approved"),
			}),
			"extra_atlantis_dependencies": cty.ListVal([]cty.Value{
				cty.StringVal("../shared"),
			}),
		})

		result, err := resolveLocals(locals)
		require.NoError(t, err)

		assert.Equal(t, "custom", result.AtlantisWorkflow)
		assert.Equal(t, "1.5.0", result.TerraformVersion)
		assert.NotNil(t, result.AutoPlan)
		assert.False(t, *result.AutoPlan)
		assert.NotNil(t, result.Skip)
		assert.True(t, *result.Skip)
		assert.NotNil(t, result.markedProject)
		assert.True(t, *result.markedProject)
		assert.Equal(t, []string{"approved"}, result.ApplyRequirements)
		assert.Equal(t, []string{"../shared"}, result.ExtraAtlantisDependencies)
	})
}

func TestParseLocals_Integration(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "parse-locals-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a simple terragrunt.hcl file
	terragruntContent := `
locals {
  atlantis_workflow = "test-workflow"
  atlantis_autoplan = true
}

terraform {
  source = "git::https://github.com/example/module.git"
}
`
	terragruntPath := filepath.Join(tmpDir, "terragrunt.hcl")
	err = os.WriteFile(terragruntPath, []byte(terragruntContent), 0644)
	require.NoError(t, err)

	// Create parsing context
	ctx, err := NewParsingContextWithConfigPath(context.Background(), terragruntPath)
	require.NoError(t, err)

	// Test parseLocals function
	result, err := parseLocals(ctx, terragruntPath, nil)
	require.NoError(t, err)

	assert.Equal(t, "test-workflow", result.AtlantisWorkflow)
	assert.NotNil(t, result.AutoPlan)
	assert.True(t, *result.AutoPlan)
}

func TestParseLocalsCache(t *testing.T) {
	// Test cache operations without copying the sync.Map
	key := "test-key"
	value := "test-value"

	parseLocalsCache.Store(key, value)

	loaded, ok := parseLocalsCache.Load(key)
	assert.True(t, ok)
	assert.Equal(t, value, loaded)

	parseLocalsCache.Delete(key)

	_, ok = parseLocalsCache.Load(key)
	assert.False(t, ok)
}
