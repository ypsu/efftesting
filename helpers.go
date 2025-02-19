package efftesting

import "testing"

var currentT *testing.T

// Override overrides `p` for the duration of the test.
// Its value is reset when the test ends.
// This is a convenience helper.
func Override[T any](p *T, v T) {
	if currentT == nil {
		panic("efftesting.MissedNew")
	}
	currentT.Helper()
	oldv := *p
	*p = v
	currentT.Cleanup(func() { *p = oldv })
}

// Must fails the current test if err is `false` or is a non-nil error.
// This is a convenience helper.
func Must(err any) {
	if currentT == nil {
		panic("efftesting.MissedNew")
	}
	currentT.Helper()
	if v, ok := err.(bool); ok {
		if !v {
			currentT.Fatal("efftesting.UnexpectedFailure")
		}
		return
	}
	if err != nil {
		currentT.Fatalf("efftesting.UnexpectedError: %v", err)
	}
}

// Must1 fails the current test if err is `false` or is a non-nil error.
// The other return values are returned otherwise.
// This is a convenience helper.
func Must1[T any](v T, err any) T {
	if currentT == nil {
		panic("efftesting.MissedNew")
	}
	currentT.Helper()
	Must(err)
	return v
}

// Must2 fails the current test if err is `false` or is a non-nil error.
// The other return values are returned otherwise.
// This is a convenience helper.
func Must2[A, B any](a A, b B, err any) (A, B) {
	if currentT == nil {
		panic("efftesting.MissedNew")
	}
	currentT.Helper()
	Must(err)
	return a, b
}

// Stringify stringifies an expression.
// Can be used to stringify a single value or even as "efftesting.Stringify(somefunc())".
// If there are multiple args and the last one indicates an error then only that part is stringified.
// Otherwise the rest of the args are stringified into a reasonable format.
// This is a convenience helper.
func Stringify(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return stringify(args[0])
	}
	lastarg := args[len(args)-1]
	if v, ok := lastarg.(bool); ok {
		if !v {
			return "failed"
		}
		if len(args) == 2 {
			return stringify(args[0])
		}
		return stringify(args[:len(args)-1])
	} else if err, ok := lastarg.(error); ok || lastarg == nil {
		if err != nil {
			return stringify(err)
		}
		if len(args) == 2 {
			return stringify(args[0])
		}
		return stringify(args[:len(args)-1])
	}
	return stringify(args)
}
