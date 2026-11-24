package service

import (
	"errors"
	"testing"
)

func TestCalcMerch(t *testing.T) {
	tests := []struct {
		name         string
		photos       int
		photosPerOne int
		want         int
	}{
		{name: "no photos", photos: 0, photosPerOne: 4, want: 0},
		{name: "one photo of four is less than half", photos: 1, photosPerOne: 4, want: 0},
		{name: "two photos of four is exactly half", photos: 2, photosPerOne: 4, want: 1},
		{name: "exact units of four", photos: 8, photosPerOne: 4, want: 2},
		{name: "quarter is rounded down", photos: 9, photosPerOne: 4, want: 2},
		{name: "half is rounded up", photos: 10, photosPerOne: 4, want: 3},
		{name: "three quarters is rounded up", photos: 11, photosPerOne: 4, want: 3},
		{name: "one photo of two is exactly half", photos: 1, photosPerOne: 2, want: 1},
		{name: "exact units of two", photos: 6, photosPerOne: 2, want: 3},
		{name: "odd count of two is rounded up", photos: 7, photosPerOne: 2, want: 4},
		{name: "one photo per unit", photos: 5, photosPerOne: 1, want: 5},
		{name: "third of three is rounded down", photos: 4, photosPerOne: 3, want: 1},
		{name: "two thirds of three is rounded up", photos: 5, photosPerOne: 3, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcMerch(tt.photos, tt.photosPerOne)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("CalcMerch(%d, %d) = %d, want %d", tt.photos, tt.photosPerOne, got, tt.want)
			}
		})
	}
}

func TestCalcMerchInvalidCoefficient(t *testing.T) {
	for _, photosPerOne := range []int{0, -1, -4} {
		got, err := CalcMerch(10, photosPerOne)
		if !errors.Is(err, ErrInvalidCoefficient) {
			t.Fatalf("CalcMerch(10, %d) error = %v, want ErrInvalidCoefficient", photosPerOne, err)
		}

		if got != 0 {
			t.Fatalf("CalcMerch(10, %d) = %d, want 0 on error", photosPerOne, got)
		}
	}
}
