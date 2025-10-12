package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// LoadEnvFile Tests
// ============================================================================

// TestLoadEnvFile_ValidFile tests loading a valid .env file
func TestLoadEnvFile_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=value1
KEY2=value2
KEY3=value3
`
	err := os.WriteFile(envPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(envVars) != 3 {
		t.Errorf("expected 3 variables, got %d", len(envVars))
	}

	if envVars["KEY1"] != "value1" {
		t.Errorf("expected KEY1='value1', got '%s'", envVars["KEY1"])
	}
	if envVars["KEY2"] != "value2" {
		t.Errorf("expected KEY2='value2', got '%s'", envVars["KEY2"])
	}
}

// TestLoadEnvFile_FileNotExists tests that missing file returns an empty map
func TestLoadEnvFile_FileNotExists(t *testing.T) {
	envVars, err := LoadEnvFile("/nonexistent/path/.env")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	if len(envVars) != 0 {
		t.Errorf("expected empty map, got %d variables", len(envVars))
	}
}

// TestLoadEnvFile_EmptyFile tests loading an empty file
func TestLoadEnvFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	os.WriteFile(envPath, []byte(""), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(envVars) != 0 {
		t.Errorf("expected empty map, got %d variables", len(envVars))
	}
}

// TestLoadEnvFile_WithComments tests that comments are ignored
func TestLoadEnvFile_WithComments(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
# This is a comment
KEY1=value1
# Another comment
KEY2=value2
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(envVars) != 2 {
		t.Errorf("expected 2 variables (comments ignored), got %d", len(envVars))
	}
}

// TestLoadEnvFile_WithBlankLines tests that blank lines are ignored
func TestLoadEnvFile_WithBlankLines(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=value1

KEY2=value2

`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(envVars) != 2 {
		t.Errorf("expected 2 variables (blank lines ignored), got %d", len(envVars))
	}
}

// TestLoadEnvFile_WithQuotes tests parsing quoted values
func TestLoadEnvFile_WithQuotes(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1="value with spaces"
KEY2='single quoted'
KEY3=unquoted
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["KEY1"] != "value with spaces" {
		t.Errorf("expected 'value with spaces', got '%s'", envVars["KEY1"])
	}
	if envVars["KEY2"] != "single quoted" {
		t.Errorf("expected 'single quoted', got '%s'", envVars["KEY2"])
	}
	if envVars["KEY3"] != "unquoted" {
		t.Errorf("expected 'unquoted', got '%s'", envVars["KEY3"])
	}
}

// TestLoadEnvFile_WithWhitespace tests that whitespace is handled correctly
func TestLoadEnvFile_WithWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
  KEY1  =  value1
KEY2=value2
 KEY3 = value3
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["KEY1"] != "value1" {
		t.Errorf("expected 'value1', got '%s'", envVars["KEY1"])
	}
	if envVars["KEY3"] != "value3" {
		t.Errorf("expected 'value3', got '%s'", envVars["KEY3"])
	}
}

// TestLoadEnvFile_WithSpecialCharacters tests values with special characters
func TestLoadEnvFile_WithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
DATABASE_URL=postgres://user:pass@localhost:5432/db
API_KEY=abc-123-xyz_789
PORT=8080
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["DATABASE_URL"] != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("unexpected DATABASE_URL value: %s", envVars["DATABASE_URL"])
	}
	if envVars["API_KEY"] != "abc-123-xyz_789" {
		t.Errorf("unexpected API_KEY value: %s", envVars["API_KEY"])
	}
}

// TestLoadEnvFile_WithEmptyValue tests that empty values are allowed
func TestLoadEnvFile_WithEmptyValue(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=
KEY2=value
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["KEY1"] != "" {
		t.Errorf("expected empty value for KEY1, got '%s'", envVars["KEY1"])
	}
	if envVars["KEY2"] != "value" {
		t.Errorf("expected 'value' for KEY2, got '%s'", envVars["KEY2"])
	}
}

// TestLoadEnvFile_LineWithoutEquals tests lines without = are skipped
func TestLoadEnvFile_LineWithoutEquals(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=value1
INVALID_LINE
KEY2=value2
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only have 2 valid entries (invalid line skipped)
	if len(envVars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(envVars))
	}
}

// TestLoadEnvFile_DuplicateKeys tests that duplicate keys use the last value
func TestLoadEnvFile_DuplicateKeys(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=first
KEY1=second
KEY1=third
`
	os.WriteFile(envPath, []byte(content), 0644)

	envVars, err := LoadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Last value should win
	if envVars["KEY1"] != "third" {
		t.Errorf("expected 'third' (last value), got '%s'", envVars["KEY1"])
	}
}

// TestLoadEnvFile_PermissionDenied tests error when a file can't be read
func TestLoadEnvFile_PermissionDenied(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	// Create file
	os.WriteFile(envPath, []byte("KEY=value"), 0644)

	// Make it unreadable
	os.Chmod(envPath, 0000)
	defer os.Chmod(envPath, 0644) // Restore for cleanup

	_, err := LoadEnvFile(envPath)
	if err == nil {
		t.Fatal("expected error for permission denied, got nil")
	}
}

// ============================================================================
// LoadProjectEnv and LoadServiceEnv Tests
// ============================================================================

// TestLoadProjectEnv_FileExists tests loading a project .env
func TestLoadProjectEnv_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `PROJECT_VAR=project_value`
	os.WriteFile(envPath, []byte(content), 0644)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	envVars, err := LoadProjectEnv()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["PROJECT_VAR"] != "project_value" {
		t.Errorf("expected 'project_value', got '%s'", envVars["PROJECT_VAR"])
	}
}

// TestLoadProjectEnv_FileNotExists tests a missing project .env
func TestLoadProjectEnv_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	envVars, err := LoadProjectEnv()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	if len(envVars) != 0 {
		t.Errorf("expected empty map, got %d variables", len(envVars))
	}
}

// TestLoadServiceEnv_FileExists tests loading service-specific .env
func TestLoadServiceEnv_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env.api")

	content := `SERVICE_VAR=service_value`
	os.WriteFile(envPath, []byte(content), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	envVars, err := LoadServiceEnv("api")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if envVars["SERVICE_VAR"] != "service_value" {
		t.Errorf("expected 'service_value', got '%s'", envVars["SERVICE_VAR"])
	}
}

// TestLoadServiceEnv_FileNotExists tests missing service .env
func TestLoadServiceEnv_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	envVars, err := LoadServiceEnv("api")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	if len(envVars) != 0 {
		t.Errorf("expected empty map, got %d variables", len(envVars))
	}
}

// ============================================================================
// MergeEnvVars Tests
// ============================================================================

// TestMergeEnvVars_EmptyMaps tests merging empty maps
func TestMergeEnvVars_EmptyMaps(t *testing.T) {
	result := MergeEnvVars()
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d variables", len(result))
	}
}

// TestMergeEnvVars_SingleMap tests merging a single map
func TestMergeEnvVars_SingleMap(t *testing.T) {
	env1 := EnvVars{"KEY1": "value1", "KEY2": "value2"}
	result := MergeEnvVars(env1)

	if len(result) != 2 {
		t.Errorf("expected 2 variables, got %d", len(result))
	}
	if result["KEY1"] != "value1" {
		t.Errorf("expected 'value1', got '%s'", result["KEY1"])
	}
}

// TestMergeEnvVars_MultipleMapsNoConflict tests merging without conflicts
func TestMergeEnvVars_MultipleMapsNoConflict(t *testing.T) {
	env1 := EnvVars{"KEY1": "value1"}
	env2 := EnvVars{"KEY2": "value2"}
	env3 := EnvVars{"KEY3": "value3"}

	result := MergeEnvVars(env1, env2, env3)

	if len(result) != 3 {
		t.Errorf("expected 3 variables, got %d", len(result))
	}
	if result["KEY1"] != "value1" || result["KEY2"] != "value2" || result["KEY3"] != "value3" {
		t.Errorf("unexpected values in merged result")
	}
}

// TestMergeEnvVars_WithOverride tests that later maps override earlier ones
func TestMergeEnvVars_WithOverride(t *testing.T) {
	env1 := EnvVars{"KEY1": "first", "KEY2": "from_env1"}
	env2 := EnvVars{"KEY1": "second", "KEY3": "from_env2"}
	env3 := EnvVars{"KEY1": "third"}

	result := MergeEnvVars(env1, env2, env3)

	// KEY1 should have the last value
	if result["KEY1"] != "third" {
		t.Errorf("expected 'third' (last override), got '%s'", result["KEY1"])
	}
	// KEY2 should be from env1
	if result["KEY2"] != "from_env1" {
		t.Errorf("expected 'from_env1', got '%s'", result["KEY2"])
	}
	// KEY3 should be from env2
	if result["KEY3"] != "from_env2" {
		t.Errorf("expected 'from_env2', got '%s'", result["KEY3"])
	}
}

// ============================================================================
// LoadAllEnvForService Tests
// ============================================================================

// TestLoadAllEnvForService_AllSources tests loading from all sources
func TestLoadAllEnvForService_AllSources(t *testing.T) {
	tempDir := t.TempDir()

	// Create project .env
	projectEnv := filepath.Join(tempDir, ".env")
	os.WriteFile(projectEnv, []byte("PROJECT_VAR=project\nSHARED_VAR=from_project"), 0644)

	// Create service .env
	serviceEnv := filepath.Join(tempDir, ".env.api")
	os.WriteFile(serviceEnv, []byte("SERVICE_VAR=service\nSHARED_VAR=from_service"), 0644)

	// Config env
	configEnv := map[string]string{
		"CONFIG_VAR": "config",
		"SHARED_VAR": "from_config",
	}

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	result, err := LoadAllEnvForService("api", configEnv)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should have all three unique vars
	if result["PROJECT_VAR"] != "project" {
		t.Errorf("expected 'project', got '%s'", result["PROJECT_VAR"])
	}
	if result["SERVICE_VAR"] != "service" {
		t.Errorf("expected 'service', got '%s'", result["SERVICE_VAR"])
	}
	if result["CONFIG_VAR"] != "config" {
		t.Errorf("expected 'config', got '%s'", result["CONFIG_VAR"])
	}

	// SHARED_VAR should be from config (the highest priority)
	if result["SHARED_VAR"] != "from_config" {
		t.Errorf("expected 'from_config' (config has highest priority), got '%s'", result["SHARED_VAR"])
	}
}

// TestLoadAllEnvForService_OnlyConfigEnv tests with only config env
func TestLoadAllEnvForService_OnlyConfigEnv(t *testing.T) {
	tempDir := t.TempDir()

	configEnv := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	result, err := LoadAllEnvForService("api", configEnv)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 variables, got %d", len(result))
	}
}

// TestLoadAllEnvForService_EmptyConfigEnv tests with no config env
func TestLoadAllEnvForService_EmptyConfigEnv(t *testing.T) {
	tempDir := t.TempDir()

	// Create project .env
	projectEnv := filepath.Join(tempDir, ".env")
	os.WriteFile(projectEnv, []byte("PROJECT_VAR=project"), 0644)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	result, err := LoadAllEnvForService("api", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result["PROJECT_VAR"] != "project" {
		t.Errorf("expected 'project', got '%s'", result["PROJECT_VAR"])
	}
}

// ============================================================================
// parseLine Tests
// ============================================================================

// TestParseLine_ValidLine tests parsing a valid line
func TestParseLine_ValidLine(t *testing.T) {
	key, value, err := parseLine("KEY=value")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key != "KEY" || value != "value" {
		t.Errorf("expected KEY='value', got %s='%s'", key, value)
	}
}

// TestParseLine_BlankLine tests that blank lines return an empty key
func TestParseLine_BlankLine(t *testing.T) {
	key, value, err := parseLine("   ")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key != "" || value != "" {
		t.Errorf("expected empty key/value for blank line")
	}
}

// TestParseLine_Comment tests that comments return an empty key
func TestParseLine_Comment(t *testing.T) {
	key, value, err := parseLine("# This is a comment")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key != "" || value != "" {
		t.Errorf("expected empty key/value for comment")
	}
}

// TestParseLine_WithWhitespace tests whitespace handling
func TestParseLine_WithWhitespace(t *testing.T) {
	key, value, err := parseLine("  KEY  =  value  ")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key != "KEY" || value != "value" {
		t.Errorf("expected KEY='value' (trimmed), got %s='%s'", key, value)
	}
}

// TestParseLine_EmptyKey tests that an empty key returns an error
func TestParseLine_EmptyKey(t *testing.T) {
	_, _, err := parseLine("=value")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

// TestLoadEnvFile_WithEmptyKey tests that file with an empty key returns error
func TestLoadEnvFile_WithEmptyKey(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	content := `
KEY1=value1
=value_without_key
KEY2=value2
`
	os.WriteFile(envPath, []byte(content), 0644)

	_, err := LoadEnvFile(envPath)
	if err == nil {
		t.Fatal("expected error for empty key in file, got nil")
	}
}

// ============================================================================
// unquoteValue Tests
// ============================================================================

// TestUnquoteValue_DoubleQuotes tests removing double quotes
func TestUnquoteValue_DoubleQuotes(t *testing.T) {
	result := unquoteValue(`"value"`)
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

// TestUnquoteValue_SingleQuotes tests removing single quotes
func TestUnquoteValue_SingleQuotes(t *testing.T) {
	result := unquoteValue(`'value'`)
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

// TestUnquoteValue_NoQuotes tests that unquoted values are unchanged
func TestUnquoteValue_NoQuotes(t *testing.T) {
	result := unquoteValue("value")
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

// TestUnquoteValue_MismatchedQuotes tests mismatched quotes are not removed
func TestUnquoteValue_MismatchedQuotes(t *testing.T) {
	result := unquoteValue(`"value'`)
	if result != `"value'` {
		t.Errorf("expected mismatched quotes unchanged, got '%s'", result)
	}
}

// TestUnquoteValue_PartialQuotes tests partial quotes are not removed
func TestUnquoteValue_PartialQuotes(t *testing.T) {
	result := unquoteValue(`"value`)
	if result != `"value` {
		t.Errorf("expected partial quote unchanged, got '%s'", result)
	}
}

// TestUnquoteValue_EmptyQuotes tests empty quoted string
func TestUnquoteValue_EmptyQuotes(t *testing.T) {
	result := unquoteValue(`""`)
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

// TestUnquoteValue_QuotesInMiddle tests quotes in the middle are kept
func TestUnquoteValue_QuotesInMiddle(t *testing.T) {
	result := unquoteValue(`val"ue`)
	if result != `val"ue` {
		t.Errorf("expected quotes in middle kept, got '%s'", result)
	}
}

// ============================================================================
// InterpolateEnvVars Tests
// ============================================================================

// TestInterpolateEnvVars_BasicInterpolation tests basic ${VAR} interpolation
func TestInterpolateEnvVars_BasicInterpolation(t *testing.T) {
	envVars := EnvVars{
		"DB_USER":      "admin",
		"DB_PASSWORD":  "secret123",
		"DB_NAME":      "myapp",
		"DATABASE_URL": "postgres://${DB_USER}:${DB_PASSWORD}@localhost:5432/${DB_NAME}",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "postgres://admin:secret123@localhost:5432/myapp"
	if result["DATABASE_URL"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["DATABASE_URL"])
	}

	// Original values should remain unchanged
	if result["DB_USER"] != "admin" {
		t.Errorf("expected 'admin', got '%s'", result["DB_USER"])
	}
}

// TestInterpolateEnvVars_ShortForm tests $VAR interpolation (no braces)
func TestInterpolateEnvVars_ShortForm(t *testing.T) {
	envVars := EnvVars{
		"HOST": "localhost",
		"PORT": "8080",
		"URL":  "http://$HOST:$PORT/api",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "http://localhost:8080/api"
	if result["URL"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["URL"])
	}
}

// TestInterpolateEnvVars_MixedForms tests both ${VAR} and $VAR in the same value
func TestInterpolateEnvVars_MixedForms(t *testing.T) {
	envVars := EnvVars{
		"USER": "admin",
		"PASS": "secret",
		"URL":  "https://$USER:${PASS}@example.com",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "https://admin:secret@example.com"
	if result["URL"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["URL"])
	}
}

// TestInterpolateEnvVars_DefaultValue tests ${VAR:-default} syntax
func TestInterpolateEnvVars_DefaultValue(t *testing.T) {
	envVars := EnvVars{
		"HOST":    "localhost",
		"API_URL": "http://${HOST}:${PORT:-8080}/api",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "http://localhost:8080/api"
	if result["API_URL"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["API_URL"])
	}
}

// TestInterpolateEnvVars_DefaultValueOverridden tests default is not used when var exists
func TestInterpolateEnvVars_DefaultValueOverridden(t *testing.T) {
	envVars := EnvVars{
		"PORT":    "3000",
		"API_URL": "http://localhost:${PORT:-8080}/api",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "http://localhost:3000/api"
	if result["API_URL"] != expected {
		t.Errorf("expected '%s' (should use actual value, not default), got '%s'", expected, result["API_URL"])
	}
}

// TestInterpolateEnvVars_MultipleVariables tests multiple variables in one value
func TestInterpolateEnvVars_MultipleVariables(t *testing.T) {
	envVars := EnvVars{
		"FIRST":  "foo",
		"SECOND": "bar",
		"THIRD":  "baz",
		"RESULT": "${FIRST}-${SECOND}-${THIRD}",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "foo-bar-baz"
	if result["RESULT"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["RESULT"])
	}
}

// TestInterpolateEnvVars_NestedReferences tests variables that reference other variables
func TestInterpolateEnvVars_NestedReferences(t *testing.T) {
	envVars := EnvVars{
		"BASE_URL": "https://api.example.com",
		"API_URL":  "${BASE_URL}/v1",
		"ENDPOINT": "${API_URL}/users",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result["BASE_URL"] != "https://api.example.com" {
		t.Errorf("expected 'https://api.example.com', got '%s'", result["BASE_URL"])
	}
	if result["API_URL"] != "https://api.example.com/v1" {
		t.Errorf("expected 'https://api.example.com/v1', got '%s'", result["API_URL"])
	}
	if result["ENDPOINT"] != "https://api.example.com/v1/users" {
		t.Errorf("expected 'https://api.example.com/v1/users', got '%s'", result["ENDPOINT"])
	}
}

// TestInterpolateEnvVars_SystemEnvVar tests fallback to system environment variables
func TestInterpolateEnvVars_SystemEnvVar(t *testing.T) {
	// Set a system environment variable for testing
	os.Setenv("TEST_SYSTEM_VAR", "system_value")
	defer os.Unsetenv("TEST_SYSTEM_VAR")

	envVars := EnvVars{
		"MY_VAR": "Using system var: ${TEST_SYSTEM_VAR}",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "Using system var: system_value"
	if result["MY_VAR"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["MY_VAR"])
	}
}

// TestInterpolateEnvVars_MissingVariable tests undefined variables resolve to empty string
func TestInterpolateEnvVars_MissingVariable(t *testing.T) {
	envVars := EnvVars{
		"URL": "http://localhost:${UNDEFINED_PORT}/api",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "http://localhost:/api"
	if result["URL"] != expected {
		t.Errorf("expected '%s' (empty for undefined var), got '%s'", expected, result["URL"])
	}
}

// TestInterpolateEnvVars_NoInterpolationNeeded tests literal strings are unchanged
func TestInterpolateEnvVars_NoInterpolationNeeded(t *testing.T) {
	envVars := EnvVars{
		"LITERAL1": "just a plain string",
		"LITERAL2": "has a dollar but no var: $123",
		"LITERAL3": "escaped? $$VAR",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result["LITERAL1"] != envVars["LITERAL1"] {
		t.Errorf("expected unchanged literal string")
	}
	if result["LITERAL2"] != envVars["LITERAL2"] {
		t.Errorf("expected unchanged string with invalid var pattern")
	}
}

// TestInterpolateEnvVars_CircularReference tests that circular references are detected
func TestInterpolateEnvVars_CircularReference(t *testing.T) {
	envVars := EnvVars{
		"VAR_A": "${VAR_B}",
		"VAR_B": "${VAR_A}",
	}

	_, err := InterpolateEnvVars(envVars)
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}

	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("expected 'circular reference' in error, got: %v", err)
	}
}

// TestInterpolateEnvVars_CircularReferenceThreeWay tests a circular reference with 3 variables
func TestInterpolateEnvVars_CircularReferenceThreeWay(t *testing.T) {
	envVars := EnvVars{
		"VAR_A": "${VAR_B}",
		"VAR_B": "${VAR_C}",
		"VAR_C": "${VAR_A}",
	}

	_, err := InterpolateEnvVars(envVars)
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}

	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("expected 'circular reference' in error, got: %v", err)
	}
}

// TestInterpolateEnvVars_SelfReference tests a variable referencing itself
func TestInterpolateEnvVars_SelfReference(t *testing.T) {
	envVars := EnvVars{
		"PATH": "/usr/bin:${PATH}",
	}

	_, err := InterpolateEnvVars(envVars)
	if err == nil {
		t.Fatal("expected error for self-reference, got nil")
	}

	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("expected 'circular reference' in error, got: %v", err)
	}
}

// TestInterpolateEnvVars_EmptyMap tests interpolating an empty map
func TestInterpolateEnvVars_EmptyMap(t *testing.T) {
	envVars := EnvVars{}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d variables", len(result))
	}
}

// TestInterpolateEnvVars_ComplexExample tests a realistic complex scenario
func TestInterpolateEnvVars_ComplexExample(t *testing.T) {
	envVars := EnvVars{
		"ENV":          "production",
		"DB_HOST":      "db.example.com",
		"DB_PORT":      "5432",
		"DB_USER":      "app_user",
		"DB_PASSWORD":  "super_secret",
		"DB_NAME":      "myapp_${ENV}",
		"DATABASE_URL": "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}",
		"API_PORT":     "8080",
		"API_URL":      "http://localhost:${API_PORT:-3000}",
		"REDIS_URL":    "redis://${REDIS_HOST:-localhost}:${REDIS_PORT:-6379}",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check DATABASE_URL with nested DB_NAME
	expectedDBUrl := "postgres://app_user:super_secret@db.example.com:5432/myapp_production"
	if result["DATABASE_URL"] != expectedDBUrl {
		t.Errorf("expected '%s', got '%s'", expectedDBUrl, result["DATABASE_URL"])
	}

	// Check API_URL (PORT defined, should not use default)
	expectedAPIUrl := "http://localhost:8080"
	if result["API_URL"] != expectedAPIUrl {
		t.Errorf("expected '%s', got '%s'", expectedAPIUrl, result["API_URL"])
	}

	// Check REDIS_URL (HOST and PORT not defined, should use defaults)
	expectedRedisUrl := "redis://localhost:6379"
	if result["REDIS_URL"] != expectedRedisUrl {
		t.Errorf("expected '%s', got '%s'", expectedRedisUrl, result["REDIS_URL"])
	}

	// Check DB_NAME was interpolated
	if result["DB_NAME"] != "myapp_production" {
		t.Errorf("expected 'myapp_production', got '%s'", result["DB_NAME"])
	}
}

// TestInterpolateEnvVars_SpecialCharactersInDefault tests default values with special chars
func TestInterpolateEnvVars_SpecialCharactersInDefault(t *testing.T) {
	envVars := EnvVars{
		"URL": "http://${HOST:-localhost:8080}/api",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "http://localhost:8080/api"
	if result["URL"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["URL"])
	}
}

// TestInterpolateEnvVars_EmptyDefaultValue tests ${VAR:-} with empty default
func TestInterpolateEnvVars_EmptyDefaultValue(t *testing.T) {
	envVars := EnvVars{
		"VALUE": "prefix_${MISSING:-}_suffix",
	}

	result, err := InterpolateEnvVars(envVars)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "prefix__suffix"
	if result["VALUE"] != expected {
		t.Errorf("expected '%s', got '%s'", expected, result["VALUE"])
	}
}
