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

// detab removes the tab characters from the string if it's a multiline string.
// That's because efftesting uses backticks for multiline strings and tab indents them.
// Also remove the first newline character since that's how ` quoted strings start.
func detab(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	return strings.TrimPrefix(strings.ReplaceAll(s, "\t", ""), "\n")
}

// Expect checks that want is got.
// want must be a string literal otherwise the rewrite feature won't work.
func (et ET) Expect(desc string, got any, want expectationString) {
	g, w := fmt.Sprint(got), detab(string(want))
	if g == w {
		return
	}
	const format = "Non-empty diff for effect \"%s\", diff (-want, +got):\n%s"
	diff := Diff(w, g)
	switch reporter := et.t.(type) {
	case errorfer:
		reporter.Errorf(format, desc, diff)
	case printfer:
		reporter.Printf(format, desc, diff)
	default:
		log.Printf(format, desc, diff)
	}
}

// Context is the number of lines to display before and after the diff starts and ends.
var Context = 2

// Diff is the function to diff the expectation against the got value.
// Defaults to a very simple diff treats all lines changed from the first until the last change.
var Diff = dummydiff

func dummydiff(lts, rts string) string {
	if lts == rts {
		return ""
	}
	lt, rt := strings.Split(lts, "\n"), strings.Split(rts, "\n")
	minlen := min(len(lt), len(rt))
	var commonStart, commonEnd int
	for commonStart < minlen && lt[commonStart] == rt[commonStart] {
		commonStart++
	}
	for commonEnd < minlen && lt[len(lt)-1-commonEnd] == rt[len(rt)-1-commonEnd] {
		commonEnd++
	}
	d := make([]string, 0, 2*Context+len(lt)-commonStart-commonEnd+len(rt)-commonStart-commonEnd)
	for i := max(0, commonStart-Context); i < commonStart; i++ {
		d = append(d, " "+lt[i])
	}
	for i := commonStart; i < len(lt)-commonEnd; i++ {
		d = append(d, "-"+lt[i])
	}
	for i := commonStart; i < len(rt)-commonEnd; i++ {
		d = append(d, "+"+rt[i])
	}
	for i := len(lt) - commonEnd; i < min(len(lt), len(lt)-commonEnd+Context); i++ {
		d = append(d, " "+lt[i])
	}
	return strings.Join(d, "\n") + "\n"
}
