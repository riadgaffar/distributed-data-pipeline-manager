package loader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Clean up plugins directory using absolute path
	pluginsDir := filepath.Join(currentDir, "plugins")
	if err := os.RemoveAll(pluginsDir); err != nil {
		fmt.Printf("Warning: Failed to cleanup directory %s: %v\n", pluginsDir, err)
	}

	os.Exit(code)
}

func TestLoadParser(t *testing.T) {
	// Skip if running in -short mode
	if testing.Short() {
		t.Skip("Skipping plugin tests in short mode")
	}

	// Build test plugin before running tests
	pluginPath := filepath.Join("plugins", "json", "testdata", "json.so")

	if err := buildTestPlugin(pluginPath); err != nil {
		t.Fatalf("Failed to build test plugin: %v", err)
	}

	defer os.Remove(pluginPath) // Clean up the test plugin

	tests := []struct {
		name       string
		pluginPath string
		parserType string
		wantErr    bool
	}{
		{
			name:       "load json plugin",
			pluginPath: pluginPath,
			parserType: "json",
			wantErr:    false,
		},
		{
			name:       "non-existent plugin",
			pluginPath: "non-existent.so",
			parserType: "avro",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := LoadParser(tt.pluginPath, tt.parserType)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && parser == nil {
				t.Error("LoadParser() returned nil parser without error")
			}
		})
	}
}

func TestGetDefaultParser(t *testing.T) {
	tests := []struct {
		name       string
		parserType string
		wantErr    bool
	}{
		{
			name:       "json parser",
			parserType: "json",
			wantErr:    false,
		},
		{
			name:       "avro parser",
			parserType: "avro",
			wantErr:    true,
		},
		{
			name:       "parquet parser",
			parserType: "parquet",
			wantErr:    true,
		},
		{
			name:       "unknown parser",
			parserType: "unknown",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := getDefaultParser(tt.parserType)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDefaultParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && parser == nil {
				t.Error("getDefaultParser() returned nil parser without error")
			}
		})
	}
}

func buildTestPlugin(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-buildmode=plugin",
		"-o", path,
		"../../plugins/json/json_parser.go")
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOROOT=%s", runtime.GOROOT()))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}
	return nil
}
