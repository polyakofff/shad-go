//go:build !change

package tabletest

import (
	"strings"
	"testing"
	"time"
)

// min coverage: . 95%

type InvalidDurationError struct{}

func (e InvalidDurationError) Error() string {
	return "time: invalid duration"
}

type UnknownUnitError struct{}

func (e UnknownUnitError) Error() string {
	return "time: unknown unit"
}

func Test(t *testing.T) {

	for _, tc := range []struct {
		name     string
		input    string
		duration time.Duration
		if_error error
	}{
		{name: "hour", input: "1h", duration: time.Hour, if_error: nil},
		{name: "minute", input: "1m", duration: time.Minute, if_error: nil},
		{name: "second", input: "1s", duration: time.Second, if_error: nil},
		{name: "millisecond", input: "1ms", duration: time.Millisecond, if_error: nil},
		{name: "microsecond", input: "1us", duration: time.Microsecond, if_error: nil},
		{name: "nanosecond", input: "1ns", duration: time.Nanosecond, if_error: nil},
		{name: "half hour", input: "0.5h", duration: 30 * time.Minute, if_error: nil},
		{name: "negative duration", input: "-1h", duration: -time.Hour, if_error: nil},
		{name: "zero", input: "0", duration: 0, if_error: nil},
		{name: "max int64", input: "9223372036854775807ns", duration: time.Duration(9223372036854775807), if_error: nil},
		{name: "overflow fraction", input: ".9223372036854775809ns", duration: 0, if_error: nil},
		{name: "overflow fraction", input: ".922337203685477581ns", duration: 0, if_error: nil},
		{name: "overflow int", input: "9223372036854775809ns", duration: 0, if_error: InvalidDurationError{}},
		{name: "overflow int", input: "9223372036854775810ns", duration: 0, if_error: InvalidDurationError{}},
		{name: "no digit", input: ".s", duration: 0, if_error: InvalidDurationError{}},
		{name: "invalid duration - empty", input: "", duration: 0, if_error: InvalidDurationError{}},
		{name: "invalid duration", input: "abh", duration: 0, if_error: InvalidDurationError{}},

		{name: "unknown unit", input: "1x", duration: 0, if_error: UnknownUnitError{}},
	} {
		t.Run(tc.name, func(t *testing.T) {

			dur, err := ParseDuration(tc.input)

			switch {
			case err != nil && tc.if_error == nil:
				t.Errorf("expected no error, got %v", err)

			case err != nil && tc.if_error != nil:
				if !strings.Contains(err.Error(), tc.if_error.Error()) {
					t.Errorf("expected error containing %v, got %v", tc.if_error, err)
				}

			case err == nil && tc.if_error != nil:
				t.Errorf("expected error %v, got nil", tc.if_error)

			case dur != tc.duration:
				t.Errorf("expected duration %v, got %v", tc.duration, dur)

			}

		})
	}

}
