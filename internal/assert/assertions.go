// Package assert provides assertion functions for unit testing.
package assert

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// True asserts that an expression is true.
func True(tb testing.TB, ok bool, msgAndArgs ...any) {
	tb.Helper()
	if ok {
		return
	}
	tb.Fatal(formatMsgAndArgs("Expected expression to be true", msgAndArgs...))
}

// False asserts that an expression is false.
func False(tb testing.TB, ok bool, msgAndArgs ...any) {
	tb.Helper()
	if !ok {
		return
	}
	tb.Fatal(formatMsgAndArgs("Expected expression to be false", msgAndArgs...))
}

// Equal asserts that "expected" and "actual" are equal.
func Equal[T any](tb testing.TB, expected, actual T, msgAndArgs ...any) {
	tb.Helper()
	if objectsAreEqual(expected, actual) {
		return
	}
	msg := formatMsgAndArgs("Expected values to be equal:", msgAndArgs...)
	tb.Fatalf("%s\n%s", msg, diff(expected, actual))
}

// Error asserts that an error is not nil.
func Error(tb testing.TB, err error, msgAndArgs ...any) {
	tb.Helper()
	if err != nil {
		return
	}
	tb.Fatal(formatMsgAndArgs("Expected an error", msgAndArgs...))
}

// NoError asserts that an error is nil.
func NoError(tb testing.TB, err error, msgAndArgs ...any) {
	tb.Helper()
	if err == nil {
		return
	}
	msg := formatMsgAndArgs("Unexpected error:", msgAndArgs...)
	tb.Fatalf("%s\n%+v", msg, err)
}

// Panics asserts that the given function panics.
func Panics(tb testing.TB, fn func(), msgAndArgs ...any) {
	tb.Helper()
	defer func() {
		if recover() == nil {
			msg := formatMsgAndArgs("Expected function to panic", msgAndArgs...)
			tb.Fatal(msg)
		}
	}()
	fn()
}

// Zero asserts that a value is its zero value.
func Zero[T any](tb testing.TB, value T, msgAndArgs ...any) {
	tb.Helper()
	var zero T
	if objectsAreEqual(value, zero) {
		return
	}
	val := reflect.ValueOf(value)
	if (val.Kind() == reflect.Slice || val.Kind() == reflect.Map || val.Kind() == reflect.Array) && val.Len() == 0 {
		return
	}
	msg := formatMsgAndArgs("Expected zero value but got:", msgAndArgs...)
	tb.Fatalf("%s\n%v", msg, value)
}

func NotZero[T any](tb testing.TB, value T, msgAndArgs ...any) {
	tb.Helper()
	var zero T
	if !objectsAreEqual(value, zero) {
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			if val.Len() > 0 {
				return
			}
		default:
			return
		}
	}
	msg := formatMsgAndArgs("Unexpected zero value:", msgAndArgs...)
	tb.Fatalf("%s\n%v", msg, value)
}

func formatMsgAndArgs(msg string, args ...any) string {
	if len(args) == 0 {
		return msg
	}
	format, ok := args[0].(string)
	if !ok {
		panic("message argument must be a fmt string")
	}
	return fmt.Sprintf(format, args[1:]...)
}

func diff(expected, actual any) string {
	lines := []string{
		"expected:",
		fmt.Sprintf("%v", expected),
		"actual:",
		fmt.Sprintf("%v", actual),
	}
	return strings.Join(lines, "\n")
}

func objectsAreEqual(expected, actual any) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}
	if exp, eok := expected.([]byte); eok {
		if act, aok := actual.([]byte); aok {
			return bytes.Equal(exp, act)
		}
	}
	if exp, eok := expected.(string); eok {
		if act, aok := actual.(string); aok {
			return exp == act
		}
	}

	return reflect.DeepEqual(expected, actual)
}
