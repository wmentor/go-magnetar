package types

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestFlexibleFloat_UnmarshalJSON_Integer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{"positive integer", []byte("42"), 42},
		{"negative integer", []byte("-17"), -17},
		{"zero", []byte("0"), 0},
		{"large integer", []byte("999999999"), 999999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ff.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ff.Float())
			}
		})
	}
}

func TestFlexibleFloat_UnmarshalJSON_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{"float with decimal", []byte("5.7"), 5.7},
		{"float zero integer part", []byte("0.99"), 0.99},
		{"negative float", []byte("-3.14"), -3.14},
		{"large float", []byte("123.456"), 123.456},
		{"scientific notation", []byte("1e10"), 1e10},
		{"negative scientific notation", []byte("1.5e-2"), 0.015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ff.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ff.Float())
			}
		})
	}
}

func TestFlexibleFloat_UnmarshalJSON_StringInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{"string integer", []byte(`"123"`), 123},
		{"string negative", []byte(`"-456"`), -456},
		{"string zero", []byte(`"0"`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ff.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ff.Float())
			}
		})
	}
}

func TestFlexibleFloat_UnmarshalJSON_StringFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{"string float", []byte(`"10.5"`), 10.5},
		{"string negative float", []byte(`"-7.8"`), -7.8},
		{"string scientific", []byte(`"1.5e2"`), 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ff.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ff.Float())
			}
		})
	}
}

func TestFlexibleFloat_UnmarshalJSON_Invalid(t *testing.T) {
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

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFlexibleFloat_UnmarshalJSON_Null(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
	}{
		{"null value", []byte("null"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ff FlexibleFloat
			err := json.Unmarshal(tt.data, &ff)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ff.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, ff.Float())
			}
		})
	}
}

func TestFlexibleFloat_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleFloat
		expected float64
	}{
		{"positive value", 42.5, 42.5},
		{"negative value", -17.3, -17.3},
		{"zero", 0, 0},
		{"large value", 999999.99, 999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.value.Float() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.value.Float())
			}
		})
	}
}

func TestFlexibleFloat_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleFloat
		expected string
	}{
		{"positive value", 42.5, "42.5"},
		{"negative value", -17.3, "-17.3"},
		{"zero", 0, "0"},
		{"large value", 999999.99, "999999.99"},
		{"scientific notation", 1e10, "10000000000"},
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

func TestFlexibleFloat_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
	}{
		{"positive", 123.456},
		{"negative", -456.789},
		{"zero", 0},
		{"large", 999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal float to JSON
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			// Unmarshal to FlexibleFloat
			var ff FlexibleFloat
			err = json.Unmarshal(data, &ff)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// Verify value matches
			if ff.Float() != tt.value {
				t.Errorf("expected %v, got %v", tt.value, ff.Float())
			}

			// Verify String() works
			expectedStr := strconv.FormatFloat(tt.value, 'f', -1, 64)
			if ff.String() != expectedStr {
				t.Errorf("expected %s, got %s", expectedStr, ff.String())
			}
		})
	}
}
