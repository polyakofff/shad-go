//go:build !solution

package spacecollapse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func CollapseSpaces(input string) string {
	var sb strings.Builder
	sb.Grow(len(input))

	space := false
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		i += size

		if r == utf8.RuneError && size == 1 {
			sb.WriteRune('\uFFFD')
			space = false
			continue
		}

		if unicode.IsSpace(r) {
			if !space {
				sb.WriteRune(' ')
			}
			space = true
		} else {
			sb.WriteRune(r)
			space = false
		}
	}

	return sb.String()
}
