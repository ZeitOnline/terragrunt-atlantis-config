package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/singleflight"
)

// Test directory constants
const (
	testArtifactsDir     = "../test/artifacts"
	testReferenceOutputs = "../test/reference_outputs"
	testFixturesDir      = "../test/fixtures"
)

// Resets all flag values to their defaults in between tests
func resetForRun() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// reset caches
	getDependenciesCache = newGetDependenciesCache()
	requestGroup = singleflight.Group{}
	// reset flags
	gitRoot = pwd
	autoPlan = false
	autoMerge = false
	cascadeDependencies = true
	ignoreParentTerragrunt = true
	ignoreDependencyBlocks = false
	parallel = true
	createWorkspace = false
	createProjectName = false
	preserveWorkflows = true
	preserveProjects = true
	defaultWorkflow = ""
	filterPaths = []string{}
	outputPath = ""
	defaultTerraformVersion = ""
	defaultApplyRequirements = []string{}
	projectHclFiles = []string{}
	createHclProjectChilds = false
	createHclProjectExternalChilds = true
	useProjectMarkers = false
	executionOrderGroups = false
	dependsOn = false

	return nil
}

// Runs a test, asserting the output produced matches a reference output file
func runTest(t *testing.T, referenceFile string, args []string) {
	err := resetForRun()
	if err != nil {
		t.Error("Failed to reset default flags")
		return
	}

	randomInt := rand.Int()
	filename := filepath.Join(testArtifactsDir, fmt.Sprintf("%d.yaml", randomInt))
	defer os.Remove(filename)

	allArgs := append([]string{
		"generate",
		"--output",
		filename,
	}, args...)

	contentBytes, err := RunWithFlags(filename, allArgs)
	content := &AtlantisConfig{}
	yaml.Unmarshal(contentBytes, content)
	if err != nil {
		t.Error(err)
		return
	}

	referenceContentsBytes, err := os.ReadFile(referenceFile)
	referenceContents := &AtlantisConfig{}
	yaml.Unmarshal(referenceContentsBytes, referenceContents)
	if err != nil {
		t.Error("Failed to read reference output file")
		return
	}

	assert.Equal(t, referenceContents, content)
}

func TestSettingRoot(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
	})
}

func TestRootPathBeingAbsolute(t *testing.T) {
	parent, err := filepath.Abs(filepath.Join(testFixturesDir, "basic_module"))
	if err != nil {
		t.Error("Failed to find parent directory")
	}

	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		parent,
	})
}

func TestRootPathHavingTrailingSlash(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module") + string(filepath.Separator),
	})
}

func TestWithNoTerragruntFiles(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "empty.yaml"), []string{
		"--root",
		".", // There are no terragrunt files in this directory
		filepath.Join(testFixturesDir, "no_modules"),
	})
}

func TestWithParallelizationDisabled(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "noParallel.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--parallel=false",
	})
}

func TestIgnoringParentTerragrunt(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withoutParent.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "with_parent"),
	})
}

func TestNotIgnoringParentTerragrunt(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withParent.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "with_parent"),
		"--ignore-parent-terragrunt=false",
	})
}

func TestEnablingAutoplan(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withAutoplan.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--autoplan",
	})
}

func TestSettingWorkflowName(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "namedWorkflow.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--workflow",
		"someWorkflow",
	})
}

func TestExtraDeclaredDependencies(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "extra_dependencies.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "extra_dependency"),
	})
}

func TestNonStringErrorOnExtraDeclaredDependencies(t *testing.T) {
	err := resetForRun()
	if err != nil {
		t.Error("Failed to reset default flags")
		return
	}

	rootCmd.SetArgs([]string{
		"generate",
		"--root",
		filepath.Join("..", "test/fixtures_errors", "extra_dependency_error"),
	})
	err = rootCmd.Execute()

	expectedError := "extra_atlantis_dependencies contains non-string value at position 4"
	if err == nil || err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%v'", expectedError, err)
		return
	}
	return
}

func TestLocalTerraformModuleSource(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "local_terraform_module.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "local_terraform_module_source"),
	})
}

func TestLocalTerraformAbsModuleSource(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "local_terraform_abs_module.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "local_terraform_abs_module_source"),
	})
}

func TestLocalTfModuleSource(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "local_tf_module.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "local_tf_module_source"),
	})
}

func TestTerragruntDependencies(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "terragrunt_dependency.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt_dependency"),
	})
}

func TestIgnoringTerragruntDependencies(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "terragrunt_dependency_ignored.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt_dependency"),
		"--ignore-dependency-blocks",
	})
}

func TestCustomWorkflowName(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "different_workflow_names.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "different_workflow_names"),
	})
}

// This test covers parent Terragrunt files that are not runnable as modules themselves.
// Sometimes it is possible to have parent files that only are runnable when included
// into child modules.
func TestUnparseableParent(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "invalid_parent_module.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "invalid_parent_module"),
	})
}

func TestWithWorkspaces(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withWorkspace.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--create-workspace",
	})
}

func TestWithProjectNames(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withProjectName.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "invalid_parent_module"),
		"--create-project-name",
	})
}

func TestMergingLocalDependenciesFromParent(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "mergeParentDependencies.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "parent_with_extra_deps"),
	})
}

func TestWorkflowFromParentInLocals(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "parentDefinedWorkflow.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "parent_with_workflow_local"),
	})
}

func TestChildWorkflowOverridesParentWorkflow(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "parentAndChildDefinedWorkflow.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "child_and_parent_specify_workflow"),
	})
}

func TestExtraArguments(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "extraArguments.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "extra_arguments"),
	})
}

func TestInfrastructureLive(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "infrastructureLive.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example"),
	})
}

func TestModulesWithNoTerraformSourceDefinitions(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "no_terraform_blocks.yml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "no_terraform_blocks"),
		"--parallel",
		"--autoplan",
	})
}

func TestInfrastructureMutliAccountsVPCRoute53TGWCascading(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "multi_accounts_vpc_route53_tgw.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "multi_accounts_vpc_route53_tgw"),
		"--cascade-dependencies",
	})
}

func TestAutoPlan(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "autoplan.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "autoplan"),
		"--autoplan=false",
	})
}

func TestSkippingModules(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "skip.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "skip"),
	})
}

func TestTerraformVersionConfig(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "terraform_version.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terraform_version"),
		"--terraform-version", "0.14.9001",
	})
}

func TestPreservingOldWorkflows(t *testing.T) {
	err := resetForRun()
	if err != nil {
		t.Error("Failed to reset default flags")
		return
	}

	randomInt := rand.Int()
	filename := filepath.Join(testArtifactsDir, fmt.Sprintf("%d.yaml", randomInt))
	defer os.Remove(filename)

	// Create an existing file to simulate an existing atlantis.yaml file
	contents := []byte(`workflows:
  terragrunt:
    apply:
      steps:
      - run: terragrunt apply -no-color $PLANFILE
    plan:
      steps:
      - run: terragrunt plan -no-color -out $PLANFILE
`)
	os.WriteFile(filename, contents, 0644)

	content, err := RunWithFlags(filename, []string{
		"generate",
		"--output",
		filename,
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
	})
	if err != nil {
		t.Error("Failed to read file")
		return
	}

	referenceContents, err := os.ReadFile(filepath.Join(testReferenceOutputs, "oldWorkflowsPreserved.yaml"))
	if err != nil {
		t.Error("Failed to read reference output file")
		return
	}

	if string(content) != string(referenceContents) {
		t.Errorf("Content did not match reference output file.\n\nExpected Content: %s\n\nContent: %s", string(referenceContents), string(content))
	}
}

func TestPreservingOldProjects(t *testing.T) {
	err := resetForRun()
	if err != nil {
		t.Error("Failed to reset default flags")
		return
	}

	randomInt := rand.Int()
	filename := filepath.Join(testArtifactsDir, fmt.Sprintf("%d.yaml", randomInt))
	defer os.Remove(filename)

	// Create an existing file to simulate an existing atlantis.yaml file
	contents := []byte(`projects:
- autoplan:
    enabled: false
    when_modified:
    - '*.hcl'
    - '*.tf*'
  dir: someDir
  name: projectFromPreviousRun
`)
	os.WriteFile(filename, contents, 0644)

	content, err := RunWithFlags(filename, []string{
		"generate",
		"--preserve-projects",
		"--output",
		filename,
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
	})
	if err != nil {
		t.Error("Failed to read file")
		return
	}

	referenceContents, err := os.ReadFile(filepath.Join(testReferenceOutputs, "oldProjectsPreserved.yaml"))
	if err != nil {
		t.Error("Failed to read reference output file")
		return
	}

	if string(content) != string(referenceContents) {
		t.Errorf("Content did not match reference output file.\n\nExpected Content: %s\n\nContent: %s", string(referenceContents), string(content))
	}
}

func TestEnablingAutomerge(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withAutomerge.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--automerge",
	})
}

func TestChainedDependencies(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "chained_dependency.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "chained_dependencies"),
		"--cascade-dependencies",
	})
}

func TestChainedDependenciesHiddenBehindFlag(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "chained_dependency_no_flag.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "chained_dependencies"),
		"--cascade-dependencies=false",
	})
}

func TestApplyRequirementsLocals(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "apply_overrides.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "apply_requirements_overrides"),
	})
}

func TestApplyRequirementsFlag(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "apply_overrides_flag.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "basic_module"),
		"--apply-requirements=approved,mergeable",
	})
}

func TestFilterFlagWithInfraLiveProd(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "filterInfraLiveProd.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example"),
		"--filter",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example", "prod"),
	})
}

func TestFilterFlagWithInfraLiveNonProd(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "filterInfraLiveNonProd.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example"),
		"--filter",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example", "non-prod"),
	})
}

func TestFilterFlagWithInfraLiveProdAndNonProd(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "filterInfraLiveProdAndNonProd.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example"),
		"--filter",
		strings.Join(
			[]string{
				filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example", "non-prod"),
				filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example", "prod"),
			},
			",",
		),
	})
}

func TestFilterGlobFlagWithInfraLiveMySql(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "filterGlobInfraLiveMySQL.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example"),
		"--filter",
		filepath.Join(testFixturesDir, "terragrunt-infrastructure-live-example", "*", "*", "*", "mysql"),
	})
}

func TestMultipleIncludes(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "multiple_includes.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "multiple_includes"),
		"--terraform-version", "0.14.9001",
	})
}

func TestRemoteModuleSourceBitbucket(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_bitbucket"),
	})
}

func TestRemoteModuleSourceGCS(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_gcs"),
	})
}

func TestRemoteModuleSourceGitHTTPS(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_git_https"),
	})
}

func TestRemoteModuleSourceGitSCPLike(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_git_scp_like"),
	})
}

func TestRemoteModuleSourceGitSSH(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_git_ssh"),
	})
}

func TestRemoteModuleSourceGithubHTTPS(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_github_https"),
	})
}

func TestRemoteModuleSourceGithubSSH(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_github_ssh"),
	})
}

func TestRemoteModuleSourceHTTP(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_http"),
	})
}

func TestRemoteModuleSourceHTTPS(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_https"),
	})
}

func TestRemoteModuleSourceMercurial(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_mercurial"),
	})
}

func TestRemoteModuleSourceS3(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_s3"),
	})
}

func TestRemoteModuleSourceTerraformRegistry(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "basic.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "remote_module_source_terraform_registry"),
	})
}

func TestEnvHCLProjectsNoChilds(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "envhcl_nochilds.yaml"), []string{
		"--root",
		testFixturesDir,
		"--project-hcl-files=env.hcl",
		"--create-hcl-project-childs=false",
		"--create-hcl-project-external-childs=false",
	})
}

func TestEnvHCLProjectsSubChilds(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "envhcl_subchilds.yaml"), []string{
		"--root",
		testFixturesDir,
		"--project-hcl-files=env.hcl",
		"--create-hcl-project-childs=true",
		"--create-hcl-project-external-childs=false",
	})
}

func TestEnvHCLProjectsExternalChilds(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "envhcl_externalchilds.yaml"), []string{
		"--root",
		testFixturesDir,
		"--project-hcl-files=env.hcl",
		"--create-hcl-project-childs=false",
		"--create-hcl-project-external-childs=true",
	})
}

func TestEnvHCLProjectsAllChilds(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "envhcl_allchilds.yaml"), []string{
		"--root",
		testFixturesDir,
		"--project-hcl-files=env.hcl",
		"--create-hcl-project-childs=true",
		"--create-hcl-project-external-childs=true",
	})
}

func TestEnvHCLProjectMarker(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "project_marker.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "project_hcl_with_project_marker"),
		"--project-hcl-files=env.hcl",
		"--use-project-markers=true",
	})
}

func TestWithOriginalDir(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withOriginalDir.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "with_original_dir"),
	})
}

func TestWithExecutionOrderGroups(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withExecutionOrderGroups.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "chained_dependencies"),
		"--execution-order-groups",
	})
}

func TestWithExecutionOrderGroupsAndDependsOn(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withExecutionOrderGroupsAndDependsOn.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "chained_dependencies"),
		"--execution-order-groups",
		"--depends-on",
		"--create-project-name",
	})
}

func TestWithDependsOn(t *testing.T) {
	runTest(t, filepath.Join(testReferenceOutputs, "withDependsOn.yaml"), []string{
		"--root",
		filepath.Join(testFixturesDir, "chained_dependencies"),
		"--depends-on",
		"--create-project-name",
	})
}

// Test cache-related functions for coverage
func TestCacheFunctions(t *testing.T) {
	// Test newGetDependenciesCache
	cache := newGetDependenciesCache()
	assert.NotNil(t, cache, "Cache should not be nil")
	assert.NotNil(t, cache.data, "Cache data should not be nil")

	// Test cache set and get operations
	testOutput := getDependenciesOutput{
		dependencies: []string{"dep1", "dep2"},
		err:          nil,
	}

	// Test set operation
	cache.set("test-key", testOutput)

	// Test get operation with existing key
	result, found := cache.get("test-key")
	assert.True(t, found, "Should find the key")
	assert.Equal(t, testOutput.dependencies, result.dependencies, "Dependencies should match")
	assert.Equal(t, testOutput.err, result.err, "Error should match")

	// Test get operation with non-existing key
	_, found = cache.get("non-existent-key")
	assert.False(t, found, "Should not find non-existent key")

	// Test cleanupCaches function
	oldCache := getDependenciesCache
	getDependenciesCache.set("cleanup-test", testOutput)

	cleanupCaches()

	// After cleanup, cache should be reset
	_, found = getDependenciesCache.get("cleanup-test")
	assert.False(t, found, "Cache should be cleared after cleanup")

	// Restore original cache
	getDependenciesCache = oldCache
}

// Test uniqueStrings function
func TestUniqueStringsDetailed(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all same elements",
			input:    []string{"x", "x", "x"},
			expected: []string{"x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			if tt.input == nil {
				assert.Nil(t, result, "Result should be nil for nil input")
				return
			}

			// Check length
			assert.Len(t, result, len(tt.expected), "Result length should match expected")

			// Check that all expected elements are present
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Result should contain expected element")
			}

			// Check that there are no duplicates in result
			seen := make(map[string]bool)
			for _, item := range result {
				assert.False(t, seen[item], "Result should not contain duplicates")
				seen[item] = true
			}
		})
	}
}

func TestLookupProjectHcl(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string][]string
		value    string
		expected string
	}{
		{
			name: "value found in first key",
			m: map[string][]string{
				"project1": {"path1", "path2"},
				"project2": {"path3", "path4"},
			},
			value:    "path1",
			expected: "project1",
		},
		{
			name: "value found in second key",
			m: map[string][]string{
				"project1": {"path1", "path2"},
				"project2": {"path3", "path4"},
			},
			value:    "path3",
			expected: "project2",
		},
		{
			name: "value not found",
			m: map[string][]string{
				"project1": {"path1", "path2"},
				"project2": {"path3", "path4"},
			},
			value:    "path5",
			expected: "",
		},
		{
			name:     "empty map",
			m:        map[string][]string{},
			value:    "path1",
			expected: "",
		},
		{
			name: "empty value",
			m: map[string][]string{
				"project1": {"path1", ""},
				"project2": {"path2", "path3"},
			},
			value:    "",
			expected: "project1",
		},
		{
			name: "multiple occurrences - returns one of them",
			m: map[string][]string{
				"project1": {"path1", "path2"},
				"project2": {"path1", "path3"}, // path1 appears in both
			},
			value: "path1",
			// Since map iteration order is not guaranteed in Go,
			// we can't predict which key will be returned first
			// Just verify that one of the valid keys is returned
			expected: "", // We'll check this differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookupProjectHcl(tt.m, tt.value)
			if tt.name == "multiple occurrences - returns one of them" {
				// Special case: map iteration order is not guaranteed
				// Just verify that a valid key is returned
				assert.Contains(t, []string{"project1", "project2"}, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSliceUnion(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "both slices empty",
			a:        []string{},
			b:        []string{},
			expected: []string{},
		},
		{
			name:     "first slice empty",
			a:        []string{},
			b:        []string{"b1", "b2"},
			expected: []string{"b1", "b2"},
		},
		{
			name:     "second slice empty",
			a:        []string{"a1", "a2"},
			b:        []string{},
			expected: []string{"a1", "a2"},
		},
		{
			name:     "no overlap",
			a:        []string{"a1", "a2"},
			b:        []string{"b1", "b2"},
			expected: []string{"a1", "a2", "b1", "b2"},
		},
		{
			name:     "complete overlap",
			a:        []string{"a1", "a2"},
			b:        []string{"a1", "a2"},
			expected: []string{"a1", "a2"},
		},
		{
			name:     "partial overlap",
			a:        []string{"a1", "a2", "a3"},
			b:        []string{"a2", "a3", "b1"},
			expected: []string{"a1", "a2", "a3", "b1"},
		},
		{
			name:     "nil slices",
			a:        nil,
			b:        nil,
			expected: nil,
		},
		{
			name:     "first nil",
			a:        nil,
			b:        []string{"b1", "b2"},
			expected: []string{"b1", "b2"},
		},
		{
			name:     "second nil",
			a:        []string{"a1", "a2"},
			b:        nil,
			expected: []string{"a1", "a2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sliceUnion(tt.a, tt.b)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
