package types

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestFlexibleInt_UnmarshalJSON_Integer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"positive integer", []byte("42"), 42},
		{"negative integer", []byte("-17"), -17},
		{"zero", []byte("0"), 0},
		{"large integer", []byte("999999999"), 999999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fi.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, fi.Int())
			}
		})
	}
}

func TestFlexibleInt_UnmarshalJSON_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"float with decimal", []byte("5.7"), 5},
		{"float zero integer part", []byte("0.99"), 0},
		{"negative float", []byte("-3.14"), -3},
		{"large float", []byte("123.456"), 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fi.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, fi.Int())
			}
		})
	}
}

func TestFlexibleInt_UnmarshalJSON_StringInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"string integer", []byte(`"123"`), 123},
		{"string negative", []byte(`"-456"`), -456},
		{"string zero", []byte(`"0"`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fi.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, fi.Int())
			}
		})
	}
}

func TestFlexibleInt_UnmarshalJSON_StringFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"string float", []byte(`"10.5"`), 10},
		{"string negative float", []byte(`"-7.8"`), -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fi.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, fi.Int())
			}
		})
	}
}

func TestFlexibleInt_UnmarshalJSON_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{"invalid string", []byte(`"not-a-number"`)},
		{"invalid json", []byte("{invalid}")},
		{"boolean", []byte("true")},
		{"array", []byte("[123]")},
		{"object", []byte(`{"value": 123}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFlexibleInt_UnmarshalJSON_Null(t *testing.T) {
	t.Parallel()

	// null should not return an error (Go default behavior)
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"null value", []byte("null"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fi FlexibleInt
			err := json.Unmarshal(tt.data, &fi)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fi.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, fi.Int())
			}
		})
	}
}

func TestFlexibleInt_Int(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleInt
		expected int
	}{
		{"positive value", 42, 42},
		{"negative value", -17, -17},
		{"zero", 0, 0},
		{"large value", 999999, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.value.Int() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.value.Int())
			}
		})
	}
}

func TestFlexibleInt_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleInt
		expected string
	}{
		{"positive value", 42, "42"},
		{"negative value", -17, "-17"},
		{"zero", 0, "0"},
		{"large value", 999999, "999999"},
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

func TestFlexibleInt_Comprehensive(t *testing.T) {
	t.Parallel()

	// Test round-trip marshaling/unmarshaling
	tests := []struct {
		name  string
		value int
	}{
		{"positive", 123},
		{"negative", -456},
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal int to JSON
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			// Unmarshal to FlexibleInt
			var fi FlexibleInt
			err = json.Unmarshal(data, &fi)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// Verify value matches
			if fi.Int() != tt.value {
				t.Errorf("expected %d, got %d", tt.value, fi.Int())
			}

			// Verify String() works
			expectedStr := strconv.Itoa(tt.value)
			if fi.String() != expectedStr {
				t.Errorf("expected %s, got %s", expectedStr, fi.String())
			}
		})
	}
}
