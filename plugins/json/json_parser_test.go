package main

import (
	"reflect"
	"testing"
)

func TestJSONParser(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    interface{}
		wantErr bool
	}{
		{
			name:    "valid json object",
			input:   []byte(`{"key": "value"}`),
			want:    map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "valid json array",
			input:   []byte(`["item1", "item2"]`),
			want:    []interface{}{"item1", "item2"},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   []byte(`{"key": invalid}`),
			want:    nil,
			wantErr: true,
		},
	}

	parser := &JSONParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("verify name", func(t *testing.T) {
		if got := parser.Name(); got != "json" {
			t.Errorf("Name() = %v, want %v", got, "json")
		}
	})

	t.Run("verify version", func(t *testing.T) {
		if got := parser.Version(); got != "1.0.0" {
			t.Errorf("Version() = %v, want %v", got, "1.0.0")
		}
	})
}
