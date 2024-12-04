package parsers

import (
	"reflect"
	"testing"
)

func TestJSONParser_Parse(t *testing.T) {
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

	parser := &JSONParser{}

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
