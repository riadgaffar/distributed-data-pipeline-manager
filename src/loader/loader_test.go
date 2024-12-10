package loader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
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
	tests := []struct {
		name       string
		pluginPath string
		parserType string
		wantErr    bool
	}{
		{
			name:       "valid plugin",
			pluginPath: "../../bin/plugins/json.so",
			parserType: "json",
			wantErr:    false,
		},
		{
			name:       "non-existent plugin",
			pluginPath: "non-existent.so",
			parserType: "json",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := LoadParser(tt.pluginPath, tt.parserType)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, parser)
			assert.Equal(t, tt.parserType, parser.Name())
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
