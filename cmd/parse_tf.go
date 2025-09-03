package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gruntwork-io/terragrunt/util"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

var (
	// Unix-style relative path prefixes
	unixLocalModulePrefixes = []string{"./", "../"}
	// Windows-style relative path prefixes
	windowsLocalModulePrefixes = []string{".\\", "..\\"}
)

// localModuleSourcePrefixes contains all platform-specific prefixes for local module sources
var localModuleSourcePrefixes = append(unixLocalModulePrefixes, windowsLocalModulePrefixes...)

func parseTerraformLocalModuleSource(path string) ([]string, error) {
	moduleCallSources, err := extractModuleCallSources(path)
	if err != nil {
		return nil, err
	}

	var sourceMap = make(map[string]struct{})
	for _, source := range moduleCallSources {
		if isLocalTerraformModuleSource(source) {
			modulePath := util.JoinPath(path, source)
			// Include both .tf* and .tofu* files
			modulePathGlobTf := util.JoinPath(modulePath, "*.tf*")
			modulePathGlobTofu := util.JoinPath(modulePath, "*.tofu*")

			sourceMap[modulePathGlobTf] = struct{}{}
			sourceMap[modulePathGlobTofu] = struct{}{}

			// find local module source recursively
			subSources, err := parseTerraformLocalModuleSource(modulePath)
			if err != nil {
				return nil, err
			}

			for _, subSource := range subSources {
				sourceMap[subSource] = struct{}{}
			}
		}
	}

	var sources = []string{}
	for source := range sourceMap {
		sources = append(sources, source)
	}

	return sources, nil
}

// extractModuleCallSources parses HCL files in a directory and extracts module call sources
func extractModuleCallSources(dir string) ([]string, error) {
	var sources []string

	// File patterns to search for
	patterns := []string{"*.tf", "*.tf.json", "*.tofu", "*.tofu.json"}
	var files []string

	// Collect all matching files
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}

	parser := hclparse.NewParser()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			// Log skipped files for debugging
			logger := createLogger()
			logger.Debugf("Skipping unreadable file %s: %v", file, err)
			continue
		}

		var f *hcl.File
		var diags hcl.Diagnostics

		if strings.HasSuffix(file, ".json") {
			f, diags = parser.ParseJSON(content, file)
		} else {
			f, diags = parser.ParseHCL(content, file)
		}

		if diags.HasErrors() {
			// Log parse errors for debugging
			logger := createLogger()
			logger.Debugf("Skipping file with parse errors %s: %v", file, diags)
			continue
		}

		// Extract module calls from the parsed file
		fileSources := extractModuleCallsFromFile(f)
		sources = append(sources, fileSources...)
	}

	return sources, nil
}

// extractModuleCallsFromFile extracts module call sources from a parsed HCL file
func extractModuleCallsFromFile(file *hcl.File) []string {
	var sources []string

	// Handle HCL native syntax
	if body, ok := file.Body.(*hclsyntax.Body); ok {
		for _, block := range body.Blocks {
			if block.Type == "module" && len(block.Labels) > 0 {
				// Look for the source attribute
				if sourceAttr, exists := block.Body.Attributes["source"]; exists {
					// Try to evaluate the expression to get the string value
					sourceVal, diags := sourceAttr.Expr.Value(nil)
					if !diags.HasErrors() && sourceVal.Type() == cty.String {
						sources = append(sources, sourceVal.AsString())
					}
				}
			}
		}
	} else {
		// Handle JSON syntax using generic HCL body content extraction
		content, diags := file.Body.Content(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type:       "module",
					LabelNames: []string{"name"},
				},
			},
		})
		if !diags.HasErrors() {
			for _, block := range content.Blocks {
				if block.Type == "module" {
					attrs, diags := block.Body.JustAttributes()
					if !diags.HasErrors() {
						if sourceAttr, exists := attrs["source"]; exists {
							sourceVal, diags := sourceAttr.Expr.Value(nil)
							if !diags.HasErrors() && sourceVal.Type() == cty.String {
								sources = append(sources, sourceVal.AsString())
							}
						}
					}
				}
			}
		}
	}

	return sources
}

func isLocalTerraformModuleSource(raw string) bool {
	for _, prefix := range localModuleSourcePrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}

	return false
}
