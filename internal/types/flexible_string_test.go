package types

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestFlexibleString_UnmarshalJSON_Bool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"bool true", []byte("true"), "true"},
		{"bool false", []byte("false"), "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}

			// Verify Bool() works
			b, err := fs.Bool()
			if err != nil {
				t.Fatalf("Bool() failed: %v", err)
			}
			expectedBool := tt.expected == "true"
			if b != expectedBool {
				t.Errorf("Bool() expected %v, got %v", expectedBool, b)
			}
		})
	}
}

func TestFlexibleString_UnmarshalJSON_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"simple string", []byte(`"hello"`), "hello"},
		{"string with spaces", []byte(`"hello world"`), "hello world"},
		{"empty string", []byte(`""`), ""},
		{"string with numbers", []byte(`"123"`), "123"},
		{"string with special chars", []byte(`"hello\nworld"`), "hello\nworld"},
		{"unicode string", []byte(`"Привет"`), "Привет"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_UnmarshalJSON_Integer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"positive integer", []byte("42"), "42"},
		{"negative integer", []byte("-17"), "-17"},
		{"zero", []byte("0"), "0"},
		{"large integer", []byte("999999999"), "999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_UnmarshalJSON_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"float with decimal", []byte("5.7"), "5.7"},
		{"float zero integer part", []byte("0.99"), "0.99"},
		{"negative float", []byte("-3.14"), "-3.14"},
		{"large float", []byte("123.456"), "123.456"},
		{"scientific notation", []byte("1e10"), "10000000000"},
		{"negative scientific notation", []byte("1.5e-2"), "0.015"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_UnmarshalJSON_Null(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"null value", []byte("null"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_UnmarshalJSON_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{"invalid json", []byte("{invalid}")},
		{"array", []byte("[123]")},
		{"object", []byte(`{"value": 123}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFlexibleString_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    FlexibleString
		expected string
	}{
		{"simple string", "hello", "hello"},
		{"empty string", "", ""},
		{"string with spaces", "hello world", "hello world"},
		{"string with numbers", "123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.value.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.value.String())
			}
		})
	}
}

func TestFlexibleString_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"simple", "hello"},
		{"with spaces", "hello world"},
		{"with numbers", "123"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal string to JSON
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			// Unmarshal to FlexibleString
			var fs FlexibleString
			err = json.Unmarshal(data, &fs)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// Verify value matches
			if fs.String() != tt.value {
				t.Errorf("expected %q, got %q", tt.value, fs.String())
			}

			// Verify String() works
			expectedStr := tt.value
			if fs.String() != expectedStr {
				t.Errorf("expected %s, got %s", expectedStr, fs.String())
			}
		})
	}
}

func TestFlexibleString_NumberToStringConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"integer to string", []byte("42"), "42"},
		{"float to string", []byte("3.14"), "3.14"},
		{"negative float to string", []byte("-2.5"), "-2.5"},
		{"scientific notation to string", []byte("1e5"), "100000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check if it's a number and converted properly
			var expectedNum float64
			json.Unmarshal(tt.data, &expectedNum)
			expected := strconv.FormatFloat(expectedNum, 'f', -1, 64)
			if fs.String() != expected {
				t.Errorf("expected %q, got %q", expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_Int64Precision(t *testing.T) {
	t.Parallel()

	// Test with a large integer that could lose precision with float64
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"large int64", []byte("9223372036854775807"), "9223372036854775807"}, // Max int64
		{"negative large int64", []byte("-9223372036854775807"), "-9223372036854775807"},
		{"max safe float", []byte("9007199254740991"), "9007199254740991"}, // Max safe int in float64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, fs.String())
			}
		})
	}
}

func TestFlexibleString_BoolConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
		isBool   bool
	}{
		{"bool true from JSON", []byte("true"), true, true},
		{"bool false from JSON", []byte("false"), false, true},
		{"string true", []byte(`"true"`), true, true},
		{"string false", []byte(`"false"`), false, true},
		{"string yes", []byte(`"yes"`), false, false}, // strconv.ParseBool doesn't accept "yes"
		{"string 1", []byte(`"1"`), true, true},
		{"string 0", []byte(`"0"`), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Test Bool() method
			b, err := fs.Bool()
			if tt.isBool {
				if err != nil {
					t.Fatalf("Bool() failed: %v", err)
				}
				if b != tt.expected {
					t.Errorf("Bool() expected %v, got %v", tt.expected, b)
				}
			} else {
				// For non-boolean values, Bool() should return an error
				if err == nil {
					t.Errorf("Bool() should return error for non-boolean value")
				}
			}
		})
	}
}

func TestFlexibleString_IntConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected int64
		hasInt   bool
	}{
		{"integer 42", []byte("42"), 42, true},
		{"negative integer", []byte("-17"), -17, true},
		{"zero", []byte("0"), 0, true},
		{"string 123", []byte(`"123"`), 123, true},
		{"string not-a-number", []byte(`"not-a-number"`), 0, false},
		{"float 3.14", []byte("3.14"), 0, false}, // strconv.ParseInt doesn't accept floats
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Test Int() method
			i, err := fs.Int()
			if tt.hasInt {
				if err != nil {
					t.Fatalf("Int() failed: %v", err)
				}
				if i != tt.expected {
					t.Errorf("Int() expected %v, got %v", tt.expected, i)
				}
			} else {
				// For non-integer values, Int() should return an error
				if err == nil {
					t.Errorf("Int() should return error for non-integer value")
				}
			}
		})
	}
}

func TestFlexibleString_FloatConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected float64
		hasFloat bool
	}{
		{"float 3.14", []byte("3.14"), 3.14, true},
		{"negative float", []byte("-2.5"), -2.5, true},
		{"zero", []byte("0"), 0, true},
		{"string 123.45", []byte(`"123.45"`), 123.45, true},
		{"string not-a-number", []byte(`"not-a-number"`), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var fs FlexibleString
			err := json.Unmarshal(tt.data, &fs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Test Float() method
			f, err := fs.Float()
			if tt.hasFloat {
				if err != nil {
					t.Fatalf("Float() failed: %v", err)
				}
				if f != tt.expected {
					t.Errorf("Float() expected %v, got %v", tt.expected, f)
				}
			} else {
				// For non-float values, Float() should return an error
				if err == nil {
					t.Errorf("Float() should return error for non-float value")
				}
			}
		})
	}
}
