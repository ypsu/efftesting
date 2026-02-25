// Package internal contains internal helper functions.
package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

// Location represents a line in a file.
// The filename is absolute.
type Location struct {
	Fname string
	Line  int
}

func (loc Location) String() string {
	return fmt.Sprintf("%s:%d", loc.Fname, loc.Line)
}

// Replacer contains the new expectations for a set of locations.
type Replacer struct {
	sync.Mutex
	Replacements map[Location]string
	Incomplete   map[Location]bool
}

// Replace marks the current caller's location to be replaced with newstr.
func (r *Replacer) Replace(newstr string) Location {
	loc := Location{}
	_, loc.Fname, loc.Line, _ = runtime.Caller(2)
	r.Lock()
	defer r.Unlock()
	if _, found := r.Replacements[loc]; found {
		return Location{}
	}
	r.Incomplete[loc] = true
	r.Replacements[loc] = newstr
	return loc
}

func makelit(s string, indent int) *ast.BasicLit {
	// Replace the expectation with a string wrapped in " or ` quotes, whichever fits best.
	if strings.IndexByte(s, '\n') == -1 || strings.IndexByte(s, '`') != -1 {
		return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", s)}
	}
	indentstr := strings.Repeat("\t", indent)
	ss := strings.Split(s, "\n")
	for i, line := range ss {
		if i == 0 || line != "" {
			ss[i] = indentstr + line
		}
	}
	if ss[len(ss)-1] == "" {
		// The last line should be indented so that the closing `) is at the right place.
		ss[len(ss)-1] = indentstr
	}
	return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`\n%s`", strings.Join(ss, "\n"))}
}

// Apply applies are stored replacements to a given file.
func (r *Replacer) Apply(fname string) error {
	r.Lock()
	defer r.Unlock()
	if len(r.Replacements) == 0 {
		return nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("efft.ParseSource: %v", err)
	}

	var inspectErr error
	ast.Inspect(f, func(n ast.Node) bool {
		if inspectErr != nil {
			return false
		}
		if n == nil {
			return true
		}
		exprstmt, ok := n.(*ast.ExprStmt)
		if !ok {
			return true
		}
		callexpr, ok := exprstmt.X.(*ast.CallExpr)
		if !ok {
			return false // no need to dig deeper than expressions
		}
		selexpr, ok := callexpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return false // not a function call, so this cannot be efft.Effect() or ...Equals()
		}
		funcname, pos, rparen := selexpr.Sel.Name, fset.Position(callexpr.Pos()), callexpr.Rparen
		if funcname == "Equals" {
			// This might be an Effect's Equals so go to the caller then.
			callexpr, ok = selexpr.X.(*ast.CallExpr)
			if !ok {
				return false
			}
			selexpr, ok = callexpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return false
			}
			funcname, pos = selexpr.Sel.Name, fset.Position(callexpr.Pos())
		}
		loc := Location{pos.Filename, pos.Line}
		repl, found := r.Replacements[loc]
		if !found || funcname != "Effect" && funcname != "FatalEffect" {
			return false
		}
		delete(r.Replacements, loc)

		exprstmt.X = &ast.CallExpr{
			Fun:    &ast.SelectorExpr{X: callexpr, Sel: ast.NewIdent("Equals")},
			Args:   []ast.Expr{makelit(repl, pos.Column)},
			Rparen: rparen,
		}
		return false
	})
	if inspectErr != nil {
		return fmt.Errorf("efft.Rewrite: %v", inspectErr)
	}
	if len(r.Replacements) > 0 {
		lines := make([]int, 0, len(r.Replacements))
		for loc := range r.Replacements {
			lines = append(lines, loc.Line)
		}
		slices.Sort(lines)
		return fmt.Errorf("efft.ReplacementsFailed file=%s lines=%v", filepath.Base(fname), lines)
	}

	bs := &bytes.Buffer{}
	if err := format.Node(bs, fset, f); err != nil {
		return fmt.Errorf("efft.Format file=%s: %v", fname, err)
	}
	if err := os.WriteFile(fname, bs.Bytes(), 0644); err != nil {
		return fmt.Errorf("efft.WriteBack: %v", err)
	}
	return nil
}

// ApplyAll applies all replacements to all files.
func (r *Replacer) ApplyAll() error {
	filesmap := map[string]bool{}
	for loc := range r.Replacements {
		filesmap[loc.Fname] = true
	}
	for _, f := range slices.Sorted(maps.Keys(filesmap)) {
		if err := r.Apply(f); err != nil {
			return fmt.Errorf("efft.UpdateFile file=%s: %v", f, err)
		}
	}
	return nil
}

// Detab removes the leading tab characters from the string if it's a multiline string.
// That's because efftesting uses backticks for multiline strings and tab indents them.
func Detab(s string) string {
	if !strings.HasPrefix(s, "\n") {
		return s
	}
	indent := 1
	for indent < len(s) && s[indent] == '\t' {
		indent++
	}
	return strings.TrimPrefix(strings.ReplaceAll(s, s[:indent], "\n"), "\n")
}
