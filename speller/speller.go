//go:build !solution

package speller

import "strings"

func Append(builder *strings.Builder, s string) {
	if builder.Len() > 0 {
		builder.WriteString(" ")
	}
	builder.WriteString(s)
}

func SpellUpTo9(n int64) string {
	switch n {
	case 1:
		return "one"
	case 2:
		return "two"
	case 3:
		return "three"
	case 4:
		return "four"
	case 5:
		return "five"
	case 6:
		return "six"
	case 7:
		return "seven"
	case 8:
		return "eight"
	case 9:
		return "nine"
	default:
		return ""
	}
}

func SpellUpTo19(n int64) string {
	switch n {
	case 10:
		return "ten"
	case 11:
		return "eleven"
	case 12:
		return "twelve"
	case 13:
		return "thirteen"
	case 14:
		return "fourteen"
	case 15:
		return "fifteen"
	case 16:
		return "sixteen"
	case 17:
		return "seventeen"
	case 18:
		return "eighteen"
	case 19:
		return "nineteen"
	default:
		return ""
	}
}

func SpellUpTo99(n int64) string {
	switch {
	case n < 10:
		return SpellUpTo9(n)
	case n < 20:
		return SpellUpTo19(n)
	default:
		tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
		if n%10 == 0 {
			return tens[n/10]
		}
		return tens[n/10] + "-" + SpellUpTo9(n%10)
	}
}

func SpellUpTo999(n int64) string {
	var result strings.Builder
	if n >= 100 {
		Append(&result, SpellUpTo9(n/100))
		Append(&result, "hundred")
	}
	if n%100 > 0 {
		Append(&result, SpellUpTo99(n%100))
	}
	return result.String()
}

func Spell(n int64) string {
	if n == 0 {
		return "zero"
	}

	var result strings.Builder
	if n < 0 {
		Append(&result, "minus")
		n = -n
	}

	if n >= 1e9 {
		Append(&result, SpellUpTo999(n/1e9))
		Append(&result, "billion")
	}
	if n%1e9 >= 1e6 {
		Append(&result, SpellUpTo999(n%1e9/1e6))
		Append(&result, "million")
	}
	if n%1e6 >= 1e3 {
		Append(&result, SpellUpTo999(n%1e6/1e3))
		Append(&result, "thousand")
	}
	if n%1e3 > 0 {
		Append(&result, SpellUpTo999(n%1e3))
	}
	return result.String()
}
