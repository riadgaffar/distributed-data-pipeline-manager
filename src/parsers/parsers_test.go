package parsers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
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

func TestParser_Parse(t *testing.T) {
	parser := getTestParser(t)

	tests := []struct {
		name    string
		input   []byte
		want    interface{}
		wantErr bool
	}{
		{
			name:    "valid simple object",
			input:   []byte(`{"name": "John", "age": 30}`),
			want:    map[string]interface{}{"name": "John", "age": float64(30)},
			wantErr: false,
		},
		{
			name:    "valid array",
			input:   []byte(`[1, 2, 3, 4]`),
			want:    []interface{}{float64(1), float64(2), float64(3), float64(4)},
			wantErr: false,
		},
		{
			name:  "valid nested object",
			input: []byte(`{"user": {"name": "John", "details": {"age": 30, "active": true}}}`),
			want: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"details": map[string]interface{}{
						"age":    float64(30),
						"active": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "valid empty object",
			input:   []byte(`{}`),
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "valid null value",
			input:   []byte(`null`),
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{"name": "John", "age": 30`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "mixed types array",
			input:   []byte(`[1, "string", true, null, {"key": "value"}]`),
			want:    []interface{}{float64(1), "string", true, nil, map[string]interface{}{"key": "value"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)

			// The error log is expected for invalid cases
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getTestParser(t *testing.T) Parser {
	// Build test plugin before running tests
	pluginPath := filepath.Join("plugins", "json", "testdata", "json.so")

	if err := buildTestPlugin(pluginPath); err != nil {
		t.Fatalf("Failed to build test plugin: %v", err)
	}

	// Load test plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		t.Fatalf("Failed to load test plugin: %v", err)
	}

	symParser, err := p.Lookup("Parser")
	if err != nil {
		t.Fatalf("Failed to lookup Parser symbol: %v", err)
	}

	parser, ok := symParser.(Parser)
	if !ok {
		t.Fatalf("Invalid parser type: %T", symParser)
	}

	return parser
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
