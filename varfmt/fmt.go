//go:build !solution

package varfmt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func Sprintf(format string, args ...interface{}) string {
	// "{1} {0}", "Hello", "World" -> "World Hello"
	// "{0} {}", "Hello", "World" -> "Hello World"
	// "{ kek } {}", "kek" -> "{ kek } kek"
	// "{0 }" -> "{0 }"

	cache := make(map[int]string)

	getArg := func(idx int) string {
		if val, ok := cache[idx]; ok {
			return val
		}
		val := fmt.Sprint(args[idx])
		cache[idx] = val
		return val
	}

	var sb strings.Builder

	pos := 0

	for i := 0; i < len(format); {
		r, size := utf8.DecodeRuneInString(format[i:])

		if r == '{' {
			var ri rune
			var si int
			number := 0
			length := 0
			for {
				ri, si = utf8.DecodeRuneInString(format[i+1+length:])

				if ri < '0' || ri > '9' {
					break
				}

				number = number*10 + int(ri-'0')
				length += si
			}
			i2 := i + 1 + length + si

			if ri == '}' {
				if length > 0 {
					sb.WriteString(getArg(number))
				} else {
					sb.WriteString(getArg(pos))
				}
				pos++
			} else {
				sb.WriteString(format[i:i2])
			}
			i = i2

		} else {
			sb.WriteRune(r)
			i += size
		}
	}
	return sb.String()
}
