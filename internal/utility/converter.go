package utility

import (
	"errors"
	"math"
	"strconv"
)

func ToInt32(s string) (int32, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("invalid number format")
	}

	if val < math.MinInt32 || val > math.MaxInt32 {
		return 0, errors.New("value out of range for int32")
	}

	return int32(val), nil
}
