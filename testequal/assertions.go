//go:build !solution

package testequal

import "fmt"

func equal(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch ta := a.(type) {
	case int:
		tb, ok := b.(int)
		return ok && ta == tb
	case int8:
		tb, ok := b.(int8)
		return ok && ta == tb
	case int16:
		tb, ok := b.(int16)
		return ok && ta == tb
	case int32:
		tb, ok := b.(int32)
		return ok && ta == tb
	case int64:
		tb, ok := b.(int64)
		return ok && ta == tb
	case uint:
		tb, ok := b.(uint)
		return ok && ta == tb
	case uint8:
		tb, ok := b.(uint8)
		return ok && ta == tb
	case uint16:
		tb, ok := b.(uint16)
		return ok && ta == tb
	case uint32:
		tb, ok := b.(uint32)
		return ok && ta == tb
	case uint64:
		tb, ok := b.(uint64)
		return ok && ta == tb
	case string:
		tb, ok := b.(string)
		return ok && ta == tb
	case map[string]string:
		tb, ok := b.(map[string]string)
		if !ok {
			return false
		}
		if (ta == nil) != (tb == nil) {
			return false
		}
		if len(ta) != len(tb) {
			return false
		}
		for k, v := range ta {
			v2, ok := tb[k]
			if !ok || v != v2 {
				return false
			}
		}
		return true
	case []int:
		tb, ok := b.([]int)
		if !ok {
			return false
		}
		if (ta == nil) != (tb == nil) {
			return false
		}
		if len(ta) != len(tb) {
			return false
		}
		for i, v := range ta {
			if v != tb[i] {
				return false
			}
		}
		return true
	case []byte:
		tb, ok := b.([]byte)
		if !ok {
			return false
		}
		if (ta == nil) != (tb == nil) {
			return false
		}
		if len(ta) != len(tb) {
			return false
		}
		for i, v := range ta {
			if v != tb[i] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// AssertEqual checks that expected and actual are equal.
//
// Marks caller function as having failed but continues execution.
//
// Returns true iff arguments are equal.
func AssertEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	t.Helper()

	equal := equal(expected, actual)
	if !equal {
		var msg string
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		} else {
			msg = fmt.Sprintf("expected: %v, actual: %v", expected, actual)
		}
		t.Errorf(msg)
		return false
	}
	return true
}

// AssertNotEqual checks that expected and actual are not equal.
//
// Marks caller function as having failed but continues execution.
//
// Returns true iff arguments are not equal.
func AssertNotEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	t.Helper()

	equal := equal(expected, actual)
	if equal {
		var msg string
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		} else {
			msg = fmt.Sprintf("expected: %v, actual: %v", expected, actual)
		}
		t.Errorf(msg)
		return false
	}
	return true
}

// RequireEqual does the same as AssertEqual but fails caller test immediately.
func RequireEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if !AssertEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// RequireNotEqual does the same as AssertNotEqual but fails caller test immediately.
func RequireNotEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	if !AssertNotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}
