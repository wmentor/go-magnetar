package types

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FlexibleInt int

func (fi *FlexibleInt) UnmarshalJSON(data []byte) error {
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		*fi = FlexibleInt(int(num)) // Отрезаем дробную часть (5.7 станет 5)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if val, err := strconv.Atoi(str); err == nil {
			*fi = FlexibleInt(val)
			return nil
		}
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			*fi = FlexibleInt(int(val))
			return nil
		}
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleInt", string(data))
}

func (fi FlexibleInt) Int() int {
	return int(fi)
}

func (fi FlexibleInt) String() string {
	return strconv.Itoa(fi.Int())
}
