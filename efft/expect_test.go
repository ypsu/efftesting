package efft_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ypsu/efftesting/efft"
	"github.com/ypsu/efftesting/efft/internal"
)

func TestExpect(t *testing.T) {
	efft.Init(t)
	efft.Expect(false)("false")
	efft.Expect(true)("true")
	efft.Expect(0)("0")
	efft.Expect(-43)("-43")
	efft.Expect("blah")("blah")
	efft.Expect([]int{1, 2, 3, 4, 5})(`
		[
		  1,
		  2,
		  3,
		  4,
		  5
		]`)
	efft.Expect(map[int]int{1: 2, 2: 4, 4: 8, 8: 16})(`
		{
		  "1": 2,
		  "2": 4,
		  "4": 8,
		  "8": 16
		}`)
	efft.Expect("hello\nworld\n")(`
		hello
		world
	`)
	efft.Expect("\nhello\nworld\n")(`
		
		hello
		world
	`)
	efft.Expect("hello\nworld\n")(`
		hello
		world
	`)
	efft.Expect("\nhello\nworld\n")(`
		
		hello
		world
	`)

	structure := []struct {
		I       int
		V       []string
		private int
	}{{1, []string{"a", "b"}, 7}, {2, []string{"multiline\nstring"}, 9}}
	efft.Expect(structure)(`
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

	// multiarg and error testcases
	efft.Expect(fmt.Errorf("SomeError"))("SomeError")
	efft.Expect("result1", true)("result1")
	efft.Expect("result1", false)(`
		[
		  "result1",
		  false
		]`)
	efft.Expect("result1", nil)("result1")
	efft.Expect("result1", fmt.Errorf("SomeError"))("SomeError")
	efft.Expect("result1", "result2", nil)(`
		[
		  "result1",
		  "result2"
		]`,
	)
	efft.Expect("result1", "result2", fmt.Errorf("SomeError"))("SomeError")
	efft.Expect(func() (int, string, error) { return 1, "result2", nil }())(`
		[
		  1,
		  "result2"
		]`)
	efft.Expect(func() (int, string, error) { return 1, "result2", fmt.Errorf("SomeError") }())("SomeError")
}

func TestReplacer(t *testing.T) {
	efft.Init(t)
	tmpfile := filepath.Join(t.TempDir(), "test.go")
	testfile := internal.Detab(strings.ReplaceAll(`
		package main

		func TestSomething() {
			efft.Init(t)
			// line 5
			efft.Expect("somevalue")("somevalue")
			efft.Expect("somevalue")
			efft.Expect("newvalue")("oldvalue")
			efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
			efft.Expect("new\nvalue")("oldvalue") // line 10
			efft.Expect("new value")(!
				new
				value
			!) // line 14
			efft.Expect("\nnew\n\nvalue")("oldvalue")
			go func() {
				efft.Expect("newvalue")("oldvalue") // line 17
			}()
			efft.Expect("a", "b", "c")("oldvalue")
			efft.Expect("a", "b", "c")()
			efft.Expect("a", "b", "c")("a", "b")
			efft.Expect("a", "b", "c")(3)
			// some comment before
			efft.Expect("x\ny") // line 24
			efft.Expect("x\ny")
			// some comment after
		}
	`, "!", "`"))

	apply := func(line int, s string) (string, error) {
		t.Helper()
		efft.Must(os.WriteFile(tmpfile, []byte(testfile), 0644))
		replacer := internal.Replacer{Replacements: map[internal.Location]string{}}
		replacer.Replacements[internal.Location{Fname: tmpfile, Line: line}] = s
		if err := replacer.Apply(tmpfile); err != nil {
			return "", err
		}
		newfile, err := os.ReadFile(tmpfile)
		efft.Must(err)
		return strings.ReplaceAll(efft.Diff(string(testfile), string(newfile)), "`", "!"), nil
	}

	// no replacement
	efft.Expect(apply(6, "somevalue"))("")

	// add expectation
	efft.Expect(apply(7, "newvalue"))(`
		 	// line 5
		 	efft.Expect("somevalue")("somevalue")
		-	efft.Expect("somevalue")
		+	efft.Expect("somevalue")("newvalue")
		 	efft.Expect("newvalue")("oldvalue")
		 	efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
	`)

	// simple replacement
	efft.Expect(apply(8, "newvalue"))(`
		 	efft.Expect("somevalue")("somevalue")
		 	efft.Expect("somevalue")
		-	efft.Expect("newvalue")("oldvalue")
		+	efft.Expect("newvalue")("newvalue")
		 	efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
		 	efft.Expect("new\nvalue")("oldvalue") // line 10
	`)

	// quote change
	efft.Expect(apply(9, "newvalue"))(`
		 	efft.Expect("somevalue")
		 	efft.Expect("newvalue")("oldvalue")
		-	efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
		+	efft.Expect( /* line 9 */ "newvalue")("newvalue")
		 	efft.Expect("new\nvalue")("oldvalue") // line 10
		 	efft.Expect("new value")(!
	`)

	// add newline
	efft.Expect(apply(9, "new\nvalue"))(`
		 	efft.Expect("somevalue")
		 	efft.Expect("newvalue")("oldvalue")
		-	efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
		+	efft.Expect( /* line 9 */ "newvalue")(!
		+		new
		+		value!)
		 	efft.Expect("new\nvalue")("oldvalue") // line 10
		 	efft.Expect("new value")(!
	`)

	// remove newline
	efft.Expect(apply(11, "new value"))(`
		 	efft.Expect( /* line 9 */ "newvalue")(!oldvalue!)
		 	efft.Expect("new\nvalue")("oldvalue") // line 10
		-	efft.Expect("new value")(!
		-		new
		-		value
		-	!) // line 14
		+	efft.Expect("new value")("new value",
		+	) // line 14
		 	efft.Expect("\nnew\n\nvalue")("oldvalue")
		 	go func() {
	`)

	// add more newline
	efft.Expect(apply(11, "\nnew\n\nvalue"))(`
		 	efft.Expect("new\nvalue")("oldvalue") // line 10
		 	efft.Expect("new value")(!
		-		new
		-		value
		-	!) // line 14
		+		
		+		new
		+
		+		value!) // line 14
		 	efft.Expect("\nnew\n\nvalue")("oldvalue")
		 	go func() {
	`)

	// update in goroutine
	efft.Expect(apply(17, "newvalue"))(`
		 	efft.Expect("\nnew\n\nvalue")("oldvalue")
		 	go func() {
		-		efft.Expect("newvalue")("oldvalue") // line 17
		+		efft.Expect("newvalue")("newvalue") // line 17
		 	}()
		 	efft.Expect("a", "b", "c")("oldvalue")
	`)

	// expect has multiple arguments
	efft.Expect(apply(19, "a,b,c"))(`
		 		efft.Expect("newvalue")("oldvalue") // line 17
		 	}()
		-	efft.Expect("a", "b", "c")("oldvalue")
		+	efft.Expect("a", "b", "c")("a,b,c")
		 	efft.Expect("a", "b", "c")()
		 	efft.Expect("a", "b", "c")("a", "b")
	`)

	// expectation is empty
	efft.Expect(apply(20, "a,b,c"))(`
		 	}()
		 	efft.Expect("a", "b", "c")("oldvalue")
		-	efft.Expect("a", "b", "c")()
		+	efft.Expect("a", "b", "c")("a,b,c")
		 	efft.Expect("a", "b", "c")("a", "b")
		 	efft.Expect("a", "b", "c")(3)
	`)

	// expectation has multiple arguments
	efft.Expect(apply(21, "a,b,c"))(`
		 	efft.Expect("a", "b", "c")("oldvalue")
		 	efft.Expect("a", "b", "c")()
		-	efft.Expect("a", "b", "c")("a", "b")
		+	efft.Expect("a", "b", "c")("a,b,c")
		 	efft.Expect("a", "b", "c")(3)
		 	// some comment before
	`)

	// expectation is a number
	efft.Expect(apply(22, "a,b,c"))(`
		 	efft.Expect("a", "b", "c")()
		 	efft.Expect("a", "b", "c")("a", "b")
		-	efft.Expect("a", "b", "c")(3)
		+	efft.Expect("a", "b", "c")("a,b,c")
		 	// some comment before
		 	efft.Expect("x\ny") // line 24
	`)

	// adding expectation keeps comments intact 1
	efft.Expect(apply(24, "x\ny"))(`
		 	efft.Expect("a", "b", "c")(3)
		 	// some comment before
		-	efft.Expect("x\ny") // line 24
		+	efft.Expect("x\ny")(!
		+		x
		+		y!) // line 24
		 	efft.Expect("x\ny")
		 	// some comment after
	`)

	// adding expectation keeps comments intact 2
	efft.Expect(apply(25, "x\ny"))(`
		 	// some comment before
		 	efft.Expect("x\ny") // line 24
		-	efft.Expect("x\ny")
		+	efft.Expect("x\ny")(!
		+		x
		+		y!)
		 	// some comment after
		 }
	`,
	)

	// backtick in the string means quoted string
	efft.Expect(apply(6, "x\n`\ny"))(`
		 	efft.Init(t)
		 	// line 5
		-	efft.Expect("somevalue")("somevalue")
		+	efft.Expect("somevalue")("x\n!\ny")
		 	efft.Expect("somevalue")
		 	efft.Expect("newvalue")("oldvalue")
	`)

	// bad replacement
	efft.Expect(apply(1, ""))("efft.ReplacementsFailed file=test.go lines=[1]")
}

func TestMust(t *testing.T) {
	efft.Init(t)
	efft.Must(true)
	efft.Must(nil)
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

	efft.Init(t)
	efft.Override(&x, 5)
	checkx(5)
}
