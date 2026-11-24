package service

import "errors"

var ErrInvalidCoefficient = errors.New("coefficient must be greater than zero")

func CalcMerch(photoNum int, photoPerUnit int) (int, error) {
	if photoPerUnit <= 0 {
		return 0, ErrInvalidCoefficient
	}
	merch := photoNum / photoPerUnit
	remainder := photoNum % photoPerUnit

	if remainder*2 >= photoPerUnit {
		merch++
	}
	return merch, nil
}
