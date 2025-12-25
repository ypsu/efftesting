package efft

import (
	"encoding/json"
	"fmt"
)

// Override overrides `p` for the duration of the test.
// Its value is reset when the test ends.
// This is a convenience helper.
func Override[T any](p *T, v T) {
	checkT()
	t.Helper()
	oldv := *p
	*p = v
	t.Cleanup(func() { *p = oldv })
}

// Must fails the current test if err is `false` or is a non-nil error.
// This is a convenience helper.
func Must(err any) {
	checkT()
	t.Helper()
	if v, ok := err.(bool); ok {
		if !v {
			t.Fatal("efft.UnexpectedFailure")
		}
		return
	}
	if err != nil {
		t.Fatalf("efft.UnexpectedError: %v", err)
	}
}

// Must1 fails the current test if err is `false` or is a non-nil error.
// The other return values are returned otherwise.
// This is a convenience helper.
func Must1[T any](v T, err any) T {
	checkT()
	t.Helper()
	Must(err)
	return v
}

// Must2 fails the current test if err is `false` or is a non-nil error.
// The other return values are returned otherwise.
// This is a convenience helper.
func Must2[A, B any](a A, b B, err any) (A, B) {
	checkT()
	t.Helper()
	Must(err)
	return a, b
}

func stringify1(v any) string {
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	if s, ok := v.(error); ok {
		return s.Error()
	}
	switch v := v.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(v)
	}

	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(js)
}

// Stringify stringifies an expression.
// Can be used to stringify a single value or even as "efft.Stringify(somefunc())".
// If there are multiple args and the last one indicates an error then only that part is stringified.
// Otherwise the rest of the args are stringified into a reasonable format.
// This is a convenience helper.
func Stringify(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return stringify1(args[0])
	}
	lastarg := args[len(args)-1]
	if v, ok := lastarg.(bool); ok {
		if !v {
			return stringify1(args)
		}
		if len(args) == 2 {
			return stringify1(args[0])
		}
		return stringify1(args[:len(args)-1])
	} else if err, ok := lastarg.(error); ok || lastarg == nil {
		if err != nil {
			return stringify1(err)
		}
		if len(args) == 2 {
			return Stringify(args[0])
		}
		return stringify1(args[:len(args)-1])
	}
	return stringify1(args)
}
