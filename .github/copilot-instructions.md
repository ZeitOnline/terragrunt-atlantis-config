# Copilot Instructions for terragrunt-atlantis-config

## Project Overview

This is a Go CLI tool that generates Atlantis YAML configurations for Terragrunt projects by analyzing `terragrunt.hcl` files, building dependency graphs, and creating proper autoplan configurations.

## Development Philosophy & Patterns

### Systematic PR Integration Approach

When integrating upstream PRs or fixes:

1. **Analyze the full change scope** - Don't just copy code, understand the problem being solved
2. **Adapt to current codebase standards** - The codebase has evolved (e.g., *.tofu* pattern support)
3. **Create comprehensive test coverage** - Every bug fix needs a test case that demonstrates the issue
4. **Update all reference outputs** - Reference file tests require updating existing reference files for consistency
5. **Test iteratively** - Run individual tests first, then full suite to catch integration issues

### Bug Fix Methodology

- **Root cause analysis**: Understanding both the technical implementation and the business use case being addressed
- **Test-driven validation**: Create fixtures that reproduce the exact problem scenario before implementing fixes
- **Comprehensive fixture design**: Include nested directories, external dependencies, and edge cases that mirror real-world usage
- **Reference file maintenance**: Keep all reference files consistent with current patterns and supported file extensions

### Commit Message Standards

Based on our collaborative work, follow this pattern for clear, actionable commit messages:

```
fix: improve external dependency detection for project configuration

This commit addresses a bug in the dependency detection logic that affected
how external files were being included in project configurations.

Changes:
- Updated dependency detection algorithm to use relative path analysis
- Added comprehensive test case with fixture files demonstrating the fix
- Updated reference output files to include new test project entry

The fix ensures that dependencies outside the working directory are properly
detected and included in autoplan trigger configurations.
```

### Documentation and Communication Patterns

- **Explain the "why"** - Don't just describe what changed, explain the business impact
- **Provide concrete examples** - Reference specific files and code patterns
- **Include validation steps** - Show how to verify the fix works
- **Cross-reference related work** - Connect changes to broader architectural decisions

## Architecture & Key Components

### Core Processing Pipeline

1. **HCL Parsing** (`cmd/parse_hcl.go`) - Parses Terragrunt files with caching and pool management
2. **Locals Resolution** (`cmd/parse_locals.go`) - Extracts Atlantis-specific local values
3. **Dependency Analysis** (`cmd/generate.go`) - Builds dependency graphs and creates projects
4. **Config Generation** (`cmd/config.go`) - Outputs final Atlantis YAML structure

### Critical Caching Strategy

- **Global caches**: `parsedHclCache`, `parseLocalsCache`, `getDependenciesCache`, `pathAbsoluteCache`
- **Object pools**: `hclParserPool` for HCL parser reuse
- **Cache keys**: Use file paths + modification times for invalidation
- **Memory management**: Size limits prevent bloat (e.g., `len(cacheKey) < 512`)

### Concurrency Patterns

- **SingleFlight** (`requestGroup`): Prevents duplicate dependency calculations
- **Semaphore**: Controls parallel execution with `--num-executors` (default 15)
- **ErrGroup**: Manages concurrent project creation with context cancellation

## Development Workflows

### Testing

```bash
make test              # Creates test/artifacts/, runs tests, cleans up
mkdir -p test/artifacts/      # Required before running go test manually
go test -v ./cmd              # Run specific package tests
go test -run TestName  # Run specific test
```

### Building

```bash
make build     # Test + build current platform only
make build-all # Test + build all platforms via GoReleaser
```

### Test Structure

- **Fixtures**: `test/fixtures/` - Terragrunt project examples
- **Reference outputs**: `test/reference_outputs/*.yaml` - Expected Atlantis configs
- **Reference file testing**: `runTest()` compares generated vs reference YAML
- **Cache cleanup**: Tests automatically create/clean `test/artifacts/` directory

## Project-Specific Patterns

### HCL File Processing

```go
// Always use caching when parsing
file, err := parseHclWithCache(filePath)

// Get parsers from pool, always return
parser := getHCLParser()
defer putHCLParser(parser)
```

### Dependency Detection Logic

- **Relative paths**: Use `strings.HasPrefix(relativePath, "..")` for external deps (NOT strings.Contains on absolute paths)
- **Absolute paths**: Convert via `makePathAbsolute()` with caching
- **Cascade mode**: When enabled, includes transitive dependencies recursively
- **Path resolution**: Critical for --project-hcl-files flag functionality

### Error Handling Convention

```go
// Cache both successful results AND errors with modification time
parsedHclCache.Store(filePath, parsedHclEntry{file, err, modTime})
```

### Locals Configuration Keys

Critical Atlantis-specific locals in Terragrunt files:

- `atlantis_workflow`, `atlantis_apply_requirements`, `atlantis_autoplan`
- `atlantis_skip`, `extra_atlantis_dependencies`, `atlantis_project`

## Integration Points

### Terragrunt Integration (`cmd/terragrunt_integration.go`)

- Uses Terragrunt's internal parser for dependency resolution
- Handles `include` blocks, `dependency` blocks, and `terraform.source`
- Respects Terragrunt's evaluation context and functions

### File Pattern Matching

- **Terraform**: `*.tf*` pattern matches `.tf`, `.tfvars`, etc.
- **OpenTofu**: `*.tofu*` pattern for OpenTofu support
- **HCL**: `*.hcl` for Terragrunt configuration files

### Project Generation Modes

- **Standard**: One project per `terragrunt.hcl` file
- **HCL Projects**: Use `--project-hcl-files` to create projects for arbitrary `.hcl` files
- **Child projects**: Control with `--create-hcl-project-childs/external-childs`

## Testing & Debugging

### Reference File Testing Workflow (Critical Pattern)

```bash
# 1. Create test fixtures that reproduce the issue
mkdir -p test/fixtures/your-case/
# 2. Generate expected output
terragrunt-atlantis-config generate --root test/fixtures/your-case > test/reference_outputs/your-case.yaml
# 3. Add test function calling runTest(t, "your-case.yaml", args)
# 4. Run specific test to validate
go test -v ./cmd -run TestYourCase
# 5. Run full suite to check for regressions
go test -v ./cmd
```

### Pattern Consistency Requirements

- **File patterns**: Always support both `*.tf*` and `*.tofu*` for Terraform/OpenTofu compatibility
- **Test naming**: Use descriptive test names that indicate the scenario being tested
- **Fixture structure**: Mirror real-world Terragrunt project layouts with proper include relationships
- **Reference outputs**: Must include all supported file patterns in when_modified lists

### Debugging Failed Tests

- **Individual test runs**: `go test -v ./cmd -run TestSpecificName` to isolate issues
- **Pattern mismatches**: Check if new reference outputs include both *.tf* and *.tofu* patterns
- **Cache invalidation**: Tests automatically clean up, but be aware of cache behavior in development
- **Relative path logic**: Verify `strings.HasPrefix(relativePath, "..")` vs absolute path containment

### Integration Testing Strategy

- **Start with failing test**: Create test case that reproduces the bug first
- **Fix incrementally**: Make minimal changes to fix the specific issue
- **Validate comprehensively**: Ensure fix doesn't break existing functionality
- **Update systematically**: All related reference files must be updated for pattern consistency

## Key Development Learnings

### Integration Experience Insights

1. **Always understand the problem deeply** - Complex dependency detection bugs require understanding both technical implementation and business use cases
2. **Test fixtures must be comprehensive** - Create complete directory structures with realistic file hierarchies to demonstrate exact scenarios
3. **Pattern evolution awareness** - Modern codebase supports multiple file format patterns; new features must maintain this consistency
4. **Reference file synchronization** - When adding new test cases, existing reference outputs often need updates to maintain pattern consistency across the test suite

### Critical Path Analysis Patterns

- **External dependency detection**: Use proper relative path analysis rather than string matching on absolute paths
- **File pattern matching**: Always include all supported file format patterns in configuration lists
- **Test isolation**: Run individual tests first before full suite to isolate issues
- **Cache behavior**: Be aware of caching in development - tests clean up automatically but manual testing may hit cached results

### Quality Assurance Insights

From our collaborative debugging and testing:

- **Duplicate code detection**: Watch for accidental duplicate lines and common copy-paste errors
- **Context preservation**: Always include sufficient context when making targeted code changes
- **Test validation**: Run tests after every change to catch regressions early
- **Reference file maintenance**: When test patterns change, multiple reference files often need updates simultaneously

### Architectural Decision Making

Based on our analysis and implementation work:

- **Performance over simplicity**: Complex caching is justified for large monorepos
- **Backward compatibility**: New features must work with existing Terragrunt configurations
- **Extensibility**: Support both Terraform and OpenTofu ecosystems from the start
- **Modularity**: Separate concerns between parsing, dependency analysis, and config generation

## Performance Considerations

- Cache hit rates are critical for large monorepos
- Parser pooling prevents memory spikes
- Context cancellation enables graceful shutdown on large repositories
- Concurrent processing with semaphore-controlled parallelism for optimal resource usage

## Architecture & Key Components

### Core Processing Pipeline

1. **HCL Parsing** (`cmd/parse_hcl.go`) - Parses Terragrunt files with caching and pool management
2. **Locals Resolution** (`cmd/parse_locals.go`) - Extracts Atlantis-specific local values
3. **Dependency Analysis** (`cmd/generate.go`) - Builds dependency graphs and creates projects
4. **Config Generation** (`cmd/config.go`) - Outputs final Atlantis YAML structure

### Critical Caching Strategy

- **Global caches**: `parsedHclCache`, `parseLocalsCache`, `getDependenciesCache`, `pathAbsoluteCache`
- **Object pools**: `hclParserPool` for HCL parser reuse
- **Cache keys**: Use file paths + modification times for invalidation
- **Memory management**: Size limits prevent bloat (e.g., `len(cacheKey) < 512`)

### Concurrency Patterns

- **SingleFlight** (`requestGroup`): Prevents duplicate dependency calculations
- **Semaphore**: Controls parallel execution with `--num-executors` (default 15)
- **ErrGroup**: Manages concurrent project creation with context cancellation

## Development Workflows

### Testing

```bash
make test              # Creates test/artifacts/, runs tests, cleans up
mkdir -p test/artifacts/      # Required before running go test manually
go test -v ./cmd              # Run specific package tests
go test -run TestName  # Run specific test
```

### Building

```bash
make build     # Test + build current platform only
make build-all # Test + build all platforms via GoReleaser
```

### Test Structure

- **Fixtures**: `test/fixtures/` - Terragrunt project examples
- **Reference outputs**: `test/reference_outputs/*.yaml` - Expected Atlantis configs
- **Reference file testing**: `runTest()` compares generated vs reference YAML
- **Cache cleanup**: Tests automatically create/clean `test/artifacts/` directory

## Project-Specific Patterns

### HCL File Processing

```go
// Always use caching when parsing
file, err := parseHclWithCache(filePath)

// Get parsers from pool, always return
parser := getHCLParser()
defer putHCLParser(parser)
```

### Dependency Detection Logic

- **Relative paths**: Use `strings.HasPrefix(relativePath, "..")` for external deps (NOT strings.Contains on absolute paths)
- **Absolute paths**: Convert via `makePathAbsolute()` with caching
- **Cascade mode**: When enabled, includes transitive dependencies recursively
- **Path resolution**: Critical for --project-hcl-files flag functionality

### Error Handling Convention

```go
// Cache both successful results AND errors with modification time
parsedHclCache.Store(filePath, parsedHclEntry{file, err, modTime})
```

### Locals Configuration Keys

Critical Atlantis-specific locals in Terragrunt files:

- `atlantis_workflow`, `atlantis_apply_requirements`, `atlantis_autoplan`
- `atlantis_skip`, `extra_atlantis_dependencies`, `atlantis_project`

## Integration Points

### Terragrunt Integration (`cmd/terragrunt_integration.go`)

- Uses Terragrunt's internal parser for dependency resolution
- Handles `include` blocks, `dependency` blocks, and `terraform.source`
- Respects Terragrunt's evaluation context and functions

### File Pattern Matching

- **Terraform**: `*.tf*` pattern matches `.tf`, `.tfvars`, etc.
- **OpenTofu**: `*.tofu*` pattern for OpenTofu support
- **HCL**: `*.hcl` for Terragrunt configuration files

### Project Generation Modes

- **Standard**: One project per `terragrunt.hcl` file
- **HCL Projects**: Use `--project-hcl-files` to create projects for arbitrary `.hcl` files
- **Child projects**: Control with `--create-hcl-project-childs/external-childs`

## Testing & Debugging

### Adding New Test Cases

1. Create fixture in `test/fixtures/your-case/`
2. Generate expected output: `terragrunt-atlantis-config generate --root test/fixtures/your-case`
3. Save as `test/reference_outputs/your-case.yaml`
4. Add test function calling `runTest(t, "your-case.yaml", args)`

### Manual Test Execution Requirements

**Critical**: When running tests manually with `go test` (not via `make test`), you must first create the artifacts directory:

```bash
mkdir -p test/artifacts/  # Required for tests to succeed
go test -v ./cmd          # Now tests will pass
```

The `make test` command handles this automatically, but individual `go test` runs require manual setup.

### Common Debug Patterns

- Enable verbose logging: `log.SetLevel(log.DebugLevel)`
- Check cache states in `getDependenciesCache`
- Verify relative path calculations in dependency detection
- Test with `--cascade-dependencies` vs without for different behaviors

### Performance Considerations

- Cache hit rates are critical for large monorepos
- Parser pooling prevents memory spikes
- Context cancellation enables graceful shutdown on large repositories
