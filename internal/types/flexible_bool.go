package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type FlexibleBool bool

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*fb = FlexibleBool(b)
		return nil
	}

	// Try to unmarshal as string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		switch str {
		case "true", "1", "yes", "y", "on", "True", "TRUE":
			*fb = FlexibleBool(true)
			return nil
		case "false", "0", "no", "n", "off", "False", "FALSE":
			*fb = FlexibleBool(false)
			return nil
		case "YES", "NO", "ON", "OFF", "Y", "N":
			// Convert to lowercase and check
			lowerStr := strings.ToLower(str)
			switch lowerStr {
			case "yes", "y", "on":
				*fb = FlexibleBool(true)
				return nil
			case "no", "n", "off":
				*fb = FlexibleBool(false)
				return nil
			}
		}
		// Try to parse as integer
		if val, err := strconv.Atoi(str); err == nil {
			*fb = FlexibleBool(val != 0)
			return nil
		}
		// Try to parse as float
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			*fb = FlexibleBool(val != 0)
			return nil
		}
	}

	// Try to unmarshal as integer
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		*fb = FlexibleBool(intVal != 0)
		return nil
	}

	// Try to unmarshal as float64
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		*fb = FlexibleBool(num != 0)
		return nil
	}

	// Handle null value
	var nilVal any
	if err := json.Unmarshal(data, &nilVal); err == nil && nilVal == nil {
		*fb = FlexibleBool(false)
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleBool", string(data))
}

func (fb FlexibleBool) Bool() bool {
	return bool(fb)
}

func (fb FlexibleBool) String() string {
	if fb {
		return "true"
	}
	return "false"
}
