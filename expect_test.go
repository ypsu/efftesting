package efftesting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestExpect(t *testing.T) {
	et := New(t)
	et.Expect("bool1", false, "false")
	et.Expect("bool2", true, "true")
	et.Expect("int1", 0, "0")
	et.Expect("int2", -43, "-43")
	et.Expect("string", "blah", "blah")
	et.Expect("intslice", []int{1, 2, 3, 4, 5}, `
		[
		  1,
		  2,
		  3,
		  4,
		  5
		]`)
	et.Expect("map", map[int]int{1: 2, 2: 4, 4: 8, 8: 16}, `
		{
		  "1": 2,
		  "2": 4,
		  "4": 8,
		  "8": 16
		}`)
	et.Expect("multiline_quoted1", "hello\nworld\n", "hello\nworld\n")
	et.Expect("multiline_quoted2", "\nhello\nworld\n", "\n\nhello\nworld\n")
	et.Expect("multiline_backticked1", "hello\nworld\n", `
		hello
		world
	`)
	et.Expect("multiline_backticked2", "\nhello\nworld\n", `
		
		hello
		world
	`)

	structure := []struct {
		I       int
		V       []string
		private int
	}{{1, []string{"a", "b"}, 7}, {2, []string{"multiline\nstring"}, 9}}
	et.Expect("struct", structure, `
		[
		  {
		    "I": 1,
		    "V": [
		      "a",
		      "b"
		    ]
		  },
		  {
		    "I": 2,
		    "V": [
		      "multiline\nstring"
		    ]
		  }
		]`)
}

func TestReplacer(t *testing.T) {
	tmpfile := filepath.Join(t.TempDir(), "test.go")
	testfile := detab(strings.ReplaceAll(`
		package main

		func main() {
			et := efftesting.New(nil)
			// line 5
			et.Expect("no replacement", "somevalue", "somevalue")
			et.Expect("simple string", "newvalue", "oldvalue")
			et.Expect( /* line 8 */ "quote change", "newvalue", !oldvalue!)
			et.Expect("add newline", "new\nvalue", "oldvalue") // line 9
			et.Expect("remove newline", "new value", !
				new
				value
			!) // line 13
			et.Expect("add more newlines", "\nnew\n\nvalue", "oldvalue")
			go func() {
				et.Expect("in goroutine", "newvalue", "oldvalue") // line 16
			}()
		}
	`, "!", "`"))

	apply := func(line int, s string) string {
		must(t, os.WriteFile(tmpfile, []byte(testfile), 0644))
		replacer := replacer{replacements: map[location]string{}}
		replacer.replacements[location{tmpfile, line}] = s
		must(t, replacer.apply(tmpfile))
		newfile, err := os.ReadFile(tmpfile)
		must(t, err)
		return strings.ReplaceAll(dummydiff(string(testfile), string(newfile)), "`", "!")
	}

	et := New(t)
	et.Expect("no replacement", apply(6, "somevalue"), "")
	et.Expect("simple replacement", apply(7, "newvalue"), `
		 	// line 5
		 	et.Expect("no replacement", "somevalue", "somevalue")
		-	et.Expect("simple string", "newvalue", "oldvalue")
		+	et.Expect("simple string", "newvalue", "newvalue")
		 	et.Expect( /* line 8 */ "quote change", "newvalue", !oldvalue!)
		 	et.Expect("add newline", "new\nvalue", "oldvalue") // line 9
	`)
	et.Expect("quote change", apply(8, "newvalue"), `
		 	et.Expect("no replacement", "somevalue", "somevalue")
		 	et.Expect("simple string", "newvalue", "oldvalue")
		-	et.Expect( /* line 8 */ "quote change", "newvalue", !oldvalue!)
		+	et.Expect( /* line 8 */ "quote change", "newvalue", "newvalue")
		 	et.Expect("add newline", "new\nvalue", "oldvalue") // line 9
		 	et.Expect("remove newline", "new value", !
	`)
	et.Expect("add newline", apply(9, "new\nvalue"), `
		 	et.Expect("simple string", "newvalue", "oldvalue")
		 	et.Expect( /* line 8 */ "quote change", "newvalue", !oldvalue!)
		-	et.Expect("add newline", "new\nvalue", "oldvalue") // line 9
		+	et.Expect("add newline", "new\nvalue", !
		+		new
		+		value!) // line 9
		 	et.Expect("remove newline", "new value", !
		 		new
	`)
	et.Expect("remove newline", apply(10, "new value"), `
		 	et.Expect( /* line 8 */ "quote change", "newvalue", !oldvalue!)
		 	et.Expect("add newline", "new\nvalue", "oldvalue") // line 9
		-	et.Expect("remove newline", "new value", !
		-		new
		-		value
		-	!) // line 13
		+	et.Expect("remove newline", "new value", "new value",
		+	) // line 13
		 	et.Expect("add more newlines", "\nnew\n\nvalue", "oldvalue")
		 	go func() {
	`)
	et.Expect("add more newline", apply(14, "\nnew\n\nvalue"), `
		 		value
		 	!) // line 13
		-	et.Expect("add more newlines", "\nnew\n\nvalue", "oldvalue")
		+	et.Expect("add more newlines", "\nnew\n\nvalue", !
		+		
		+		new
		+
		+		value!)
		 	go func() {
		 		et.Expect("in goroutine", "newvalue", "oldvalue") // line 16
	`)
	et.Expect("update in goroutine", apply(16, "newvalue"), `
		 	et.Expect("add more newlines", "\nnew\n\nvalue", "oldvalue")
		 	go func() {
		-		et.Expect("in goroutine", "newvalue", "oldvalue") // line 16
		+		et.Expect("in goroutine", "newvalue", "newvalue") // line 16
		 	}()
		 }
	`)
}

func TestMust(t *testing.T) {
	New(t)
	Must(true)
	Must(nil)
}

func TestOverride(t *testing.T) {
	x := 4

	checkx := func(want int) {
		t.Helper()
		if got := x; got != want {
			t.Errorf("efftetsing_test.UnexpectedXValue got=%d want=%d", got, want)
		}
	}
	t.Cleanup(func() { checkx(4) })

	New(t)
	Override(&x, 5)
	checkx(5)
}

func TestStringifySyntax(t *testing.T) {
	et := New(t)
	et.Expect("Empty", Stringify(), "")
	et.Expect("Number", Stringify(2), "2")
	et.Expect("False", Stringify(false), "false")
	et.Expect("True", Stringify(true), "true")
	et.Expect("Error", Stringify(fmt.Errorf("SomeError")), "SomeError")
	et.Expect("True1", Stringify("result1", true), "result1")
	et.Expect("False1", Stringify("result1", false), "failed")
	et.Expect("Success1", Stringify("result1", nil), "result1")
	et.Expect("Error1", Stringify("result1", fmt.Errorf("SomeError")), "SomeError")
	et.Expect("Success2", Stringify("result1", "result2", nil), `
		[
		  "result1",
		  "result2"
		]`,
	)
	et.Expect("Error2", Stringify("result1", "result2", fmt.Errorf("SomeError")), "SomeError")
	et.Expect("FuncSuccess1", Stringify(
		func() (int, string, error) { return 1, "result2", nil }(),
	), `
		[
		  1,
		  "result2"
		]`,
	)
	et.Expect("FuncError1", Stringify(
		func() (int, string, error) { return 1, "result2", fmt.Errorf("SomeError") }(),
	), "SomeError")
}

func TestMain(m *testing.M) {
	os.Exit(Main(m))
}
