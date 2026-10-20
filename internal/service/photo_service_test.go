package service

import "testing"

func TestCalculateSHA256(t *testing.T) {
	data := []byte("hello")

	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	result := CalcSHA256(data)

	if result != expected {
		t.Fatalf(
			"hash = %s, want %s",
			result,
			expected,
		)
	}
}
