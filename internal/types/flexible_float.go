package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FlexibleFloat float64

func (ff *FlexibleFloat) UnmarshalJSON(data []byte) error {
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		*ff = FlexibleFloat(num)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			*ff = FlexibleFloat(val)
			return nil
		}
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleFloat", string(data))
}

func (ff FlexibleFloat) Float() float64 {
	return float64(ff)
}

func (ff FlexibleFloat) String() string {
	return strconv.FormatFloat(ff.Float(), 'f', -1, 64)
}
