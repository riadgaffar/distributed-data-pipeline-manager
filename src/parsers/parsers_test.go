package parsers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"testing"
)

func getTestParser(t *testing.T, pluginFile string) Parser {
	// Extract parser type from plugin filename
	re := regexp.MustCompile(`(.+?)\.so`)
	match := re.FindStringSubmatch(pluginFile)
	if len(match) < 2 {
		t.Fatalf("Invalid plugin filename format: %s", pluginFile)
	}
	parserType := match[1]

	// Build test plugin before running tests
	pluginPath := filepath.Join("plugins", parserType, "testdata", pluginFile)
	sourceFile := filepath.Join("../../plugins", parserType, parserType+"_parser.go")

	if err := buildTesPlugin(pluginPath, sourceFile); err != nil {
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

func buildTesPlugin(path string, pluginSource string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", path, pluginSource)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}
	return nil
}

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
	parser := getTestParser(t, "json.so")

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

func TestParser_ParseAvro(t *testing.T) {
	schema := `{
			"type": "record",
			"name": "test",
			"fields": [
					{"name": "name", "type": "string"},
					{"name": "age", "type": "int"},
					{"name": "email", "type": "string"}
			]
	}`

	// Get and initialize parser
	parser := getTestParser(t, "avro.so")

	// Initialize Avro parser with schema
	initializedParser, err := parser.Initialize(schema)
	if err != nil {
		t.Fatalf("Failed to initialize Avro parser: %v", err)
	}

	// Use the initialized parser for subsequent operations
	parser = initializedParser.(Parser)

	// Create test data
	nativeData := map[string]interface{}{
		"name":  "test",
		"age":   25,
		"email": "test@test.com",
	}

	// Create binary data using the initialized parser
	binaryData, err := parser.Serialize(nativeData)
	if err != nil {
		t.Fatalf("Failed to create binary data: %v", err)
	}

	// Type assertion for binaryData
	binaryBytes, ok := binaryData.([]byte)
	if !ok {
		t.Fatalf("Failed to convert binaryData to []byte")
	}

	tests := []struct {
		name    string
		input   []byte
		want    interface{}
		wantErr bool
	}{
		{
			name:    "valid avro record",
			input:   binaryBytes,
			want:    nativeData,
			wantErr: false,
		},
		{
			name:    "invalid data",
			input:   []byte{0x01},
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
			name:    "partial data",
			input:   binaryBytes[:len(binaryBytes)/2],
			want:    nil,
			wantErr: true,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Convert both to JSON strings for comparison
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(tt.want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("Parse() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
