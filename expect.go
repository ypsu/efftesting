// Package efftesting checks expectations.
package efftesting

import (
	"fmt"
	"log"
	"strings"
)

// expectationString is a local type so that users cannot create it.
// Makes the library harder to misuse because users cannot pass in variables.
// This string must always be a string constant passed into the function due to the auto-rewrite feature.
type expectationString string

type printfer interface {
	Printf(format string, args ...any)
}
type errorfer interface {
	Errorf(format string, args ...any)
}

// ET (EffTesting) is an expectation tester.
type ET struct {
	t any
}

// New creates a new ET.
// t can be either *testing.T, *log.Logger, or nil.
func New(t any) ET {
	return ET{t}
}

// detab removes the tab characters from the string.
// Also remove the first newline character since that's how ` quoted strings start.
func detab(s string) string {
	return strings.TrimPrefix(strings.ReplaceAll(s, "\t", ""), "\n")
}

// Expect checks that want is got.
// want must be a string literal otherwise the rewrite feature won't work.
func (et ET) Expect(desc string, got any, want expectationString) {
	g, w := fmt.Sprint(got), detab(string(want))
	if g == w {
		return
	}
	const format = "Non-empty diff for effect \"%s\"."
	switch reporter := et.t.(type) {
	case errorfer:
		reporter.Errorf(format, desc)
	case printfer:
		reporter.Printf(format, desc)
	default:
		log.Printf(format, desc)
	}
}
