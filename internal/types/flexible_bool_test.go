package types

import (
	"encoding/json"
	"testing"
)

func TestFlexibleBool_UnmarshalJSON_Bool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"true", []byte("true"), true},
		{"false", []byte("false"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, fb.Bool())
			}
		})
	}
}

func TestFlexibleBool_UnmarshalJSON_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"string true", []byte(`"true"`), true},
		{"string false", []byte(`"false"`), false},
		{"string 1", []byte(`"1"`), true},
		{"string 0", []byte(`"0"`), false},
		{"string yes", []byte(`"yes"`), true},
		{"string no", []byte(`"no"`), false},
		{"string y", []byte(`"y"`), true},
		{"string n", []byte(`"n"`), false},
		{"string on", []byte(`"on"`), true},
		{"string off", []byte(`"off"`), false},
		{"string TRUE", []byte(`"TRUE"`), true},
		{"string FALSE", []byte(`"FALSE"`), false},
		{"string True", []byte(`"True"`), true},
		{"string False", []byte(`"False"`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, fb.Bool())
			}
		})
	}
}

func TestFlexibleBool_UnmarshalJSON_Integer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"integer 1", []byte("1"), true},
		{"integer 0", []byte("0"), false},
		{"integer 2", []byte("2"), true},
		{"integer -1", []byte("-1"), true},
		{"large integer", []byte("999999"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, fb.Bool())
			}
		})
	}
}

func TestFlexibleBool_UnmarshalJSON_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"float 1.5", []byte("1.5"), true},
		{"float 0.0", []byte("0.0"), false},
		{"float -1.5", []byte("-1.5"), true},
		{"float 0.01", []byte("0.01"), true},
		{"float -0.01", []byte("-0.01"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, fb.Bool())
			}
		})
	}
}

func TestFlexibleBool_UnmarshalJSON_Null(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"null value", []byte("null"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, fb.Bool())
			}
		})
	}
}

func TestFlexibleBool_UnmarshalJSON_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{"invalid string", []byte(`"not-a-bool"`)},
		{"invalid json", []byte("{invalid}")},
		{"array", []byte("[true]")},
		{"object", []byte(`{"value": true}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal(tt.data, &fb)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFlexibleBool_Bool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleBool
		expected bool
	}{
		{"true", FlexibleBool(true), true},
		{"false", FlexibleBool(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.value.Bool() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.value.Bool())
			}
		})
	}
}

func TestFlexibleBool_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleBool
		expected string
	}{
		{"true", FlexibleBool(true), "true"},
		{"false", FlexibleBool(false), "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.value.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.value.String())
			}
		})
	}
}

func TestFlexibleBool_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal bool to JSON
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			// Unmarshal to FlexibleBool
			var fb FlexibleBool
			err = json.Unmarshal(data, &fb)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// Verify value matches
			if fb.Bool() != tt.value {
				t.Errorf("expected %v, got %v", tt.value, fb.Bool())
			}

			// Verify String() works
			expectedStr := "true"
			if !tt.value {
				expectedStr = "false"
			}
			if fb.String() != expectedStr {
				t.Errorf("expected %s, got %s", expectedStr, fb.String())
			}
		})
	}
}

func TestFlexibleBool_StringVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// True variations
		{"string true lowercase", "true", true},
		{"string true uppercase", "TRUE", true},
		{"string true mixed case", "True", true},
		{"string 1", "1", true},
		{"string yes lowercase", "yes", true},
		{"string yes uppercase", "YES", true},
		{"string y lowercase", "y", true},
		{"string y uppercase", "Y", true},
		{"string on lowercase", "on", true},
		{"string on uppercase", "ON", true},

		// False variations
		{"string false lowercase", "false", false},
		{"string false uppercase", "FALSE", false},
		{"string false mixed case", "False", false},
		{"string 0", "0", false},
		{"string no lowercase", "no", false},
		{"string no uppercase", "NO", false},
		{"string n lowercase", "n", false},
		{"string n uppercase", "N", false},
		{"string off lowercase", "off", false},
		{"string off uppercase", "OFF", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fb FlexibleBool
			err := json.Unmarshal([]byte(`"`+tt.input+`"`), &fb)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fb.Bool() != tt.expected {
				t.Errorf("expected %v for %q, got %v", tt.expected, tt.input, fb.Bool())
			}
		})
	}
}
