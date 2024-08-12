// Package efftesting checks expectations and optionally rewrites them if the EFFTESTING_UPDATE=1 envvar is set.
// See https://github.com/ypsu/efftesting for an example.
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
// want must be a string literal otherwise the rewrite feature won't work.
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
// want must be a string literal otherwise the rewrite feature won't work.
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
// If a _test.go file has a efftesting expectations then call this explicitly:
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
