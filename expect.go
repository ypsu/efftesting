// Package efftesting checks expectations and optionally rewrites them if the EFFTESTING_UPDATE=1 envvar is set.
//
// Its main feature is an Expect(effectName string, want any, got string) function.
// It stringifies want and compares that string to got and fails the test if they are not equal.
// The magic is this: if got is wrong, efftesting can automatically update the source code to the new value.
// It should make updating the tests for a code's effects a bit easier.
// effectName is just an arbitrary name to make the test log messages clearer.
//
// See https://github.com/ypsu/efftesting/tree/main/example/example_test.go for a full example.
// See pkgtrim_test.go in https://github.com/ypsu/pkgtrim for a more realistic example.
//
// Example:
//
//	func MyStringLength(s string) int {
//		return len(s)
//	}
//
//	func TestLength(t *testing.T) {
//		et := efftesting.New(t)
//		et.Expect("string length of tükör", MyStringLength("tükör"), "7")
//	}
//
//	func TestMain(m *testing.M) {
//		os.Exit(efftesting.Main(m))
//	}
//
// Suppose you change the string to count utf8 characters instead of bytes:
//
//	func MyStringLength(s string) int {
//		return utf8.RuneCountInString(s)
//	}
//
// The expectation now fails with this:
//
//	$ go test example_test.go
//	--- FAIL: TestLength (0.00s)
//
//	    example_test.go:17: Non-empty diff for effect "string length of tükör", diff (-want, +got):
//	        -7
//	        +5
//
//	FAIL
//	Expectations need updating, use `EFFTESTING_UPDATE=1 go test ./...` for that.
//
// Rerun the test with the EFFTESTING_UPDATE=1 envvar to update the test expectation to expect 5 if that was expected from the change.
//
// There's also a Check(effectName string, want any, got string) variant that quits the test if the expectation doesn't match.
// So instead of this:
//
//	...
//	foo, err = foolib.New()
//	if err != nil {
//		t.Failf("foolib.New() failed: %v.", err)
//	}
//	...
//
// it's possible to write this:
//
//	foo, err = foolib.New()
//	et.Check("foolib.New() succeeded", err, "null")
//
// You don't need to know beforehand that err's value will be stringified to null.
// Initially add only et.Check("foolib.New() succeeded", err, "") and then simply run update expectation command as described above.
// In fact you no longer need to know beforehand what any expression's result will be, you only need to tell if a result is correct or not.
// Ideal for code whose result is more a matter of taste rather than correctness (e.g. markdown rendering).
//
// Most typical expectations can be rewritten to efftesting expectations.
// E.g. a EXPECT_LESS_THAN(3, 4) can be rewritten to Expect("comparison", 3 < 4, "true").
// Or EXPECT_STRING_CONTAINS(str, "foo") can be rewritten to Expect("contains foo", strings.Contains(str, "foo"), "true").
// Expect and Check can be a powerful building block to make managing tests simpler.
//
// efftesting formats multiline strings with backticks.
// For convenience it formats structs and slices into a multiline json:
//
//	et.Expect("slice example", strings.Split("This is a sentence.", " "), `
//		[
//			"This",
//			"is",
//			"a",
//			"sentence."
//		]`)
//
// Tip: include the string "TODO" in effectName for expectations that are still under development and are thus incorrect.
// This allows committing the expectations first.
// Once the correct implementation is in, the tests can be quickly updated with a single command.
// The only additional work then needed is removing the TODO markers while verifying the correctness of the expectations.
// Makes a test driven development much easier.
package efftesting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// expectationString is a local type so that users cannot create it.
// Makes the library harder to misuse because users cannot pass in variables.
// This string must always be a string constant passed into the function due to the auto-rewrite feature.
type expectationString string

// ET (EffTesting) is an expectation tester.
type ET struct {
	t *testing.T
}

// New creates a new ET.
func New(t *testing.T) ET {
	return ET{t}
}

// detab removes the leading tab characters from the string if it's a multiline string.
// That's because efftesting uses backticks for multiline strings and tab indents them.
func detab(s string) string {
	if !strings.HasPrefix(s, "\n") {
		return s
	}
	indent := 1
	for indent < len(s) && s[indent] == '\t' {
		indent++
	}
	return strings.TrimPrefix(strings.TrimRight(strings.ReplaceAll(s, s[:indent], "\n"), "\t"), "\n")
}

// Expect checks that want is got.
// want must be a string literal otherwise the update feature won't work.
func (et ET) Expect(desc string, got any, want expectationString) {
	g, w := stringify(got), detab(string(want))
	if g == w {
		return
	}
	const format = "Non-empty diff for effect \"%s\", diff (-want, +got):\n%s"
	diff := Diff(w, g)
	defaultReplacer.replace(g)
	et.t.Helper()
	et.t.Errorf(format, desc, diff)
}

// Check checks that want is got.
// If they are unequal, the test is aborted.
// want must be a string literal otherwise the update feature won't work.
func (et ET) Check(desc string, got any, want expectationString) {
	g, w := stringify(got), detab(string(want))
	if g == w {
		return
	}
	const format = "Non-empty diff for effect \"%s\", diff (-want, +got):\n%s"
	diff := Diff(w, g)
	defaultReplacer.replace(g)
	et.t.Helper()
	et.t.Fatalf(format, desc, diff)
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

var defaultReplacer = replacer{
	replacements: map[location]string{},
}

type location struct {
	fname string
	line  int
}

func (loc location) String() string {
	return fmt.Sprintf("%s:%d", loc.fname, loc.line)
}

type replacer struct {
	mu           sync.Mutex
	replacements map[location]string
}

func (r *replacer) replace(newstr string) bool {
	loc := location{}
	_, loc.fname, loc.line, _ = runtime.Caller(2)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.replacements[loc]; ok {
		return false
	}
	r.replacements[loc] = newstr
	return true
}

func (r *replacer) apply(fname string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.replacements) == 0 {
		return nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("efftesting/parse source: %w", err)
	}

	var inspectErr error
	ast.Inspect(f, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		if n == nil {
			return true
		}

		// Find the Expect and Check functions that have a pending replacement.
		callexpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if len(callexpr.Args) != 3 {
			return false
		}
		selexpr, ok := callexpr.Fun.(*ast.SelectorExpr)
		if !ok || selexpr.Sel.Name != "Expect" && selexpr.Sel.Name != "Check" {
			return true
		}
		pos := fset.Position(callexpr.Pos())
		loc := location{pos.Filename, pos.Line}
		repl, ok := r.replacements[loc]
		if !ok {
			return false
		}
		lit, ok := callexpr.Args[2].(*ast.BasicLit)
		if !ok {
			inspectErr = fmt.Errorf("%s: expectation is %T, want literal string", pos, callexpr.Args[2])
			return false
		}

		// Replace the expectation with a string wrapped in " or ` quotes, whichever fits best.
		delete(r.replacements, loc)
		if strings.IndexByte(repl, '\n') == -1 || strings.IndexByte(repl, '`') != -1 {
			lit.Value = fmt.Sprintf("%q", repl)
			return false
		}
		indent := strings.Repeat("\t", pos.Column)
		ss := strings.Split(repl, "\n")
		for i, line := range ss {
			if i == 0 || line != "" {
				ss[i] = indent + line
			}
		}
		if ss[len(ss)-1] == "" {
			// The last line should have one indent less so that the closing `) looks nicely indented.
			ss[len(ss)-1] = strings.TrimSuffix(indent, "\t")
		}
		lit.Value = fmt.Sprintf("`\n%s`", strings.Join(ss, "\n"))
		return false
	})
	if inspectErr != nil {
		return fmt.Errorf("efftesting/rewrite %s: %v", fname, inspectErr)
	}

	bs := &bytes.Buffer{}
	if err := format.Node(bs, fset, f); err != nil {
		return fmt.Errorf("efftesting/format the updated %s: %v", fname, err)
	}
	if err := os.WriteFile(fname, bs.Bytes(), 0644); err != nil {
		return fmt.Errorf("efftesting/write rewritten source: %v", err)
	}
	return nil
}

// Main is the TestMain for efftesting.
// If a _test.go file has efftesting expectations then call this explicitly:
//
//	func TestMain(m *testing.M) {
//		os.Exit(efftesting.Main(m))
//	}
func Main(m *testing.M) int {
	code := m.Run()
	if code == 0 || os.Getenv("EFFTESTING_UPDATE") != "1" {
		if len(defaultReplacer.replacements) != 0 {
			fmt.Fprintf(os.Stderr, "Expectations need updating, use `EFFTESTING_UPDATE=1 go test ./...` for that.\n")
		}
		return code
	}
	if len(defaultReplacer.replacements) != 0 {
		_, testfile, _, _ := runtime.Caller(1)
		if err := defaultReplacer.apply(testfile); err != nil {
			fmt.Fprintf(os.Stderr, "efftesting update failed: %v.\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "Expectations updated.\n")
	}
	return code
}

func stringify(v any) string {
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
