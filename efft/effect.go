// Package efft from efftesting checks expectations and optionally rewrites them if the EFFUP=1 envvar is set.
// Just write `efft.Effect(any arg here)`, efftesting deterministically stringifies the arguments and maintains the expectation for the result.
//
// Example usage:
//
//	func TestSplit(t *testing.T) {
//	  efft.Init(t)
//
//	  // simple cut, omits the success flag to keep things short
//	  efft.Effect(strings.CutPrefix("hello world", "hello")).Equals(" world")
//
//	  // failing cut, stringifies the success flag too on failure
//	  efft.Effect(strings.CutPrefix("hello world", "world")).Equals(`
//	    [
//	      "hello world",
//	      false
//	    ]`)
//	}
//
// You only need to write `efft.Expect(some-expression)` and `EFFUP=1 go test ./...` does the rest.
// I.e. write only this and the runner will rewrite it to the above:
//
//	func TestSplit(t *testing.T) {
//	  efft.Init(t)
//
//	  // simple cut
//	  efft.Effect(strings.CutPrefix("hello world", "hello"))
//
//	  // failing cut
//	  efft.Effect(strings.CutPrefix("hello world", "world"))
//	}
//
// Note that if the function's last arg is a nil error or true boolean then it's automatically omitted.
package efft

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/ypsu/efftesting/efft/internal"
)

// Note to include in the error message when there's a diff between the effect's got and wanted value.
var Note string

// expectationString is a local type so that users cannot create it.
// Makes the library harder to misuse because users cannot pass in variables.
// This string must always be a string constant passed into the function due to the auto-rewrite feature.
type expectationString string

var (
	t               *testing.T
	updatemode      bool
	rewriterPipe    io.Writer
	defaultReplacer internal.Replacer
)

func init() {
	updatemode = os.Getenv("EFFUP") == "1"
	if os.Getenv("EFFTESTING_REWRITE") != "1" {
		return
	}
	fname, line, newstr := "", 0, ""
	defaultReplacer.Replacements = map[internal.Location]string{}
	for {
		n, err := fmt.Scanf("%q %d %q\n", &fname, &line, &newstr)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: efft.ReadReplacements: %v\n", err)
			os.Exit(1)
		}
		defaultReplacer.Replacements[internal.Location{Fname: fname, Line: line}] = newstr
	}
	if err := defaultReplacer.ApplyAll(); err != nil {
		fmt.Fprintf(os.Stderr, "efft.ExpectationsUpdateFailure: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "efft.ExpectationsUpdatedSuccessfully")
	os.Exit(0)
}

// Init setup efft for this testcase.
// Note that efft doesn't support sub- or parallel tests.
func Init(tt *testing.T) {
	// Set up currentT from utils.go.
	tt.Helper()
	if t != nil {
		t.Fatal("efft.UnsupportedParallelTesting")
	}
	t = tt
	Note = ""
	t.Cleanup(func() { t = nil })
	defaultReplacer.Incomplete = map[internal.Location]bool{}
	defaultReplacer.Replacements = map[internal.Location]string{}
	t.Cleanup(func() {
		t.Helper()
		incomplete, replacements := defaultReplacer.Incomplete, defaultReplacer.Replacements
		if !updatemode && len(incomplete) > 0 {
			t.Errorf("efft.IncompleteExpectations: run with EFFUP=1 envvar to complete them")
		} else if len(incomplete) > 0 {
			t.Errorf("efft.IncompleteExpectations: will update them at end")
		}
		if !updatemode && len(replacements) > len(incomplete) {
			t.Errorf("efft.WrongExpectations: run with EFFUP=1 envvar to fix them")
		} else if len(replacements) > len(incomplete) {
			t.Errorf("efft.WrongExpectations: will update them at end")
		}
		if !updatemode || len(replacements) == 0 {
			return
		}
		if rewriterPipe == nil {
			cmd := exec.Command(os.Args[0])
			cmd.Env = []string{"EFFTESTING_REWRITE=1"}
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			p, err := cmd.StdinPipe()
			if err != nil {
				t.Errorf("efft.CreateRewriterPipe: %v", err)
			}
			rewriterPipe = p
			if err := cmd.Start(); err != nil {
				t.Errorf("efft.StartRewriter: %v", err)
			}
		}
		for loc, newstr := range replacements {
			fmt.Fprintf(rewriterPipe, "%q %d %q\n", loc.Fname, loc.Line, newstr)
		}
	})
}

func checkT() {
	if t == nil {
		pc, filename, _, _ := runtime.Caller(2)
		funcname := runtime.FuncForPC(pc).Name()
		if i := strings.LastIndexByte(funcname, '.'); i != -1 {
			funcname = funcname[i+1:]
		}
		funcname = path.Base(filename) + ":" + funcname
		panic(fmt.Sprintf("efft.MissingInit func=%q: make sure to call efft.Init(t) at the beginning", funcname))
	}
}

type result struct {
	got   string
	loc   internal.Location
	fatal bool
}

func (r result) Equals(wanted expectationString) {
	got, want := r.got, internal.Detab(string(wanted))
	delete(defaultReplacer.Incomplete, r.loc)
	if got == want {
		delete(defaultReplacer.Replacements, r.loc)
		return
	}
	t.Helper()
	var note string
	if Note != "" {
		note = "note=`" + Note + "` "
	}
	if updatemode || !r.fatal {
		t.Errorf("efft.EffectDiff %s-want +got:\n%s", note, Diff(want, got))
	} else {
		t.Fatalf("efft.FatalEffectDiff %s-want +got:\n%s", note, Diff(string(want), got))
	}
}

// Effect sets up an expectation.
// Effect accepts a list of any args so it can be used with functions that return multiple values.
// This is why the expectation has to be given in a separate function.
// See the package comment how to use this.
func Effect(args ...any) result { //revive:disable-line:unexported-return
	checkT()
	t.Helper()
	got := Stringify(args...)
	return result{got, defaultReplacer.Replace(got), false}
}

// FatalEffect is same as Effect but aborts the test if the expectation doesn't match.
func FatalEffect(args ...any) result { //revive:disable-line:unexported-return
	checkT()
	t.Helper()
	got := Stringify(args...)
	return result{got, defaultReplacer.Replace(got), true}
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
