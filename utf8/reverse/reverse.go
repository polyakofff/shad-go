//go:build !solution

package reverse

import (
	"strings"
	"unicode/utf8"
)

func Reverse(input string) string {
	var sb strings.Builder
	sb.Grow(len(input))

	for i := len(input); i > 0; {
		r, size := utf8.DecodeLastRuneInString(input[:i])
		if r == utf8.RuneError && size == 1 {
			sb.WriteRune('\uFFFD')
			i--
		} else {
			sb.WriteRune(r)
			i -= size
		}
	}

	return sb.String()
}
