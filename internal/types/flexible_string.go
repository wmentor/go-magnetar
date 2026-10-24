package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*fs = FlexibleString(str)
		return nil
	}

	// Try to unmarshal as bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fs = FlexibleString(strconv.FormatBool(b))
		return nil
	}

	// Try to unmarshal as int64 first (to avoid precision loss)
	var intVal int64
	if err := json.Unmarshal(data, &intVal); err == nil {
		*fs = FlexibleString(strconv.FormatInt(intVal, 10))
		return nil
	}

	// Try to unmarshal as float64
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		*fs = FlexibleString(strconv.FormatFloat(num, 'f', -1, 64))
		return nil
	}

	// Handle null value
	var nilVal any
	if err := json.Unmarshal(data, &nilVal); err == nil && nilVal == nil {
		*fs = ""
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

func (fs FlexibleString) String() string {
	return string(fs)
}

// Bool returns the boolean representation of the string value.
func (fs FlexibleString) Bool() (bool, error) {
	return strconv.ParseBool(string(fs))
}

// Int returns the integer representation of the string value.
func (fs FlexibleString) Int() (int64, error) {
	return strconv.ParseInt(string(fs), 10, 64)
}

// Float returns the float representation of the string value.
func (fs FlexibleString) Float() (float64, error) {
	return strconv.ParseFloat(string(fs), 64)
}
