package types

import (
	"fmt"
	"unicode/utf8"
)

func sliceStringAsRuneSlice(s string, from, to int) string {
	switch {
	case from < 0 && to < 0:
		return s
	case from < 0 && to >= 0:
		return string([]rune(s)[:to])
	case from >= 0 && to < 0:
		return string([]rune(s)[from:])
	case from >= 0 && to >= 0:
		return string([]rune(s)[from:to])
	default:
		panic(fmt.Sprintf("invalid sliceStringAsRuneSlice: %d, %d", from, to))
	}
}

func sliceStringAsRuneSliceWithLength(s string, from, length int) string {
	maxLength := utf8.RuneCountInString(s)
	to := from + length
	if to >= maxLength {
		to = maxLength - 1
	}

	return sliceStringAsRuneSlice(s, from, to)
}
