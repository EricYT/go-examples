package foo

import (
	"errors"
)

func Division(a, b float64) (float64, error) {
	if 0 == b {
		return 0, errors.New("b can't be zero")
	}

	return a / b, nil
}
