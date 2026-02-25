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

func TestEffect(t *testing.T) {
	efft.Init(t)
	efft.Effect(false).Equals("false")
	efft.Effect(true).Equals("true")
	efft.Effect(0).Equals("0")
	efft.Effect(-43).Equals("-43")
	efft.Effect("blah").Equals("blah")
	efft.Effect([]int{1, 2, 3, 4, 5}).Equals(`
		[
		  1,
		  2,
		  3,
		  4,
		  5
		]`)
	efft.Effect(map[int]int{1: 2, 2: 4, 4: 8, 8: 16}).Equals(`
		{
		  "1": 2,
		  "2": 4,
		  "4": 8,
		  "8": 16
		}`)
	efft.Effect("hello\nworld\n").Equals(`
		hello
		world
		`)
	efft.Effect("\nhello\nworld\n").Equals(`
		
		hello
		world
		`)
	efft.Effect("hello\nworld\n").Equals(`
		hello
		world
		`)
	efft.Effect("\nhello\nworld\n").Equals(`
		
		hello
		world
		`)
	efft.Effect("\nhello\nworld\n\t").Equals(`
		
		hello
		world
			`)
	efft.Effect("\nhello\nworld\n\t\n").Equals(`
		
		hello
		world
			
		`)

	structure := []struct {
		I       int
		V       []string
		private int
	}{{1, []string{"a", "b"}, 7}, {2, []string{"multiline\nstring"}, 9}}
	efft.Effect(structure).Equals(`
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
	efft.Effect(fmt.Errorf("SomeError")).Equals("SomeError")
	efft.Effect("result1", true).Equals("result1")
	efft.Effect("result1", false).Equals(`
		[
		  "result1",
		  false
		]`)
	efft.Effect("result1", nil).Equals("result1")
	efft.Effect("result1", fmt.Errorf("SomeError")).Equals("SomeError")
	efft.Effect("result1", "result2", nil).Equals(`
		[
		  "result1",
		  "result2"
		]`,
	)
	efft.Effect("result1", "result2", fmt.Errorf("SomeError")).Equals("SomeError")
	efft.Effect(func() (int, string, error) { return 1, "result2", nil }()).Equals(`
		[
		  1,
		  "result2"
		]`)
	efft.Effect(func() (int, string, error) { return 1, "result2", fmt.Errorf("SomeError") }()).Equals("SomeError")
}

func TestReplacer(t *testing.T) {
	efft.Init(t)
	tmpfile := filepath.Join(t.TempDir(), "test.go")
	testfile := internal.Detab(strings.ReplaceAll(`
		package main

		func TestSomething() {
			efft.Init(t)
			// line 5
			efft.Effect("somevalue").Equals("somevalue")
			efft.Effect("newvalue")
			efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
			efft.Effect("new\nvalue").Equals("oldvalue") // line 9
			efft.Effect("new value").Equals(!
				one
				two
				three
			!) // line 14
			efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
			go func() {
				efft.Effect("newvalue").Equals("oldvalue") // line 17
			}()
			efft.Effect("a", "b", "c").Equals("oldvalue")
			efft.Effect("a", "b", "c").Equals()
			efft.Effect("a", "b", "c").Equals("a", "b")
			efft.Effect("a", "b", "c").Equals(3)
			// some comment before
			efft.Effect("y\nx").Equals("x\ny") // line 24
			efft.Effect("y\nx").Equals("x\ny")
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

	efft.Note = "no replacement"
	efft.Effect(apply(6, "somevalue")).Equals("")

	efft.Note = "add expectation"
	efft.Effect(apply(7, "newvalue")).Equals(`
		 	// line 5
		 	efft.Effect("somevalue").Equals("somevalue")
		-	efft.Effect("newvalue")
		+	efft.Effect("newvalue").Equals("newvalue")
		 	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		 	efft.Effect("new\nvalue").Equals("oldvalue") // line 9
		`)

	efft.Note = "simple replacement"
	efft.Effect(apply(6, "newvalue")).Equals(`
		 	efft.Init(t)
		 	// line 5
		-	efft.Effect("somevalue").Equals("somevalue")
		+	efft.Effect("somevalue").Equals("newvalue")
		 	efft.Effect("newvalue")
		 	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		`)

	efft.Note = "quote change"
	efft.Effect(apply(8, "newvalue")).Equals(`
		 	efft.Effect("somevalue").Equals("somevalue")
		 	efft.Effect("newvalue")
		-	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		+	efft.Effect( /* line 8 */ "newvalue").Equals("newvalue")
		 	efft.Effect("new\nvalue").Equals("oldvalue") // line 9
		 	efft.Effect("new value").Equals(!
		`)

	efft.Note = "add newline"
	efft.Effect(apply(8, "new\nvalue")).Equals(`
		 	efft.Effect("somevalue").Equals("somevalue")
		 	efft.Effect("newvalue")
		-	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		+	efft.Effect( /* line 8 */ "newvalue").Equals(!
		+		new
		+		value!)
		 	efft.Effect("new\nvalue").Equals("oldvalue") // line 9
		 	efft.Effect("new value").Equals(!
		`)

	efft.Note = "remove single internal newline"
	efft.Effect(apply(10, "one\nthree\n")).Equals(`
		 	efft.Effect("new value").Equals(!
		 		one
		-		two
		-		three
		-	!) // line 14
		+		three
		+		!,
		+	) // line 14
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		`)

	efft.Note = "remove two internal newlines"
	efft.Effect(apply(10, "three\n")).Equals(`
		 	efft.Effect("new\nvalue").Equals("oldvalue") // line 9
		 	efft.Effect("new value").Equals(!
		-		one
		-		two
		-		three
		-	!) // line 14
		+		three
		+		!,
		+	) // line 14
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		`)

	efft.Note = "remove last newline"
	efft.Effect(apply(10, "one\ntwo\nthree")).Equals(`
		 		one
		 		two
		-		three
		-	!) // line 14
		+		three!,
		+	) // line 14
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		`)

	efft.Note = "remove all newlines"
	efft.Effect(apply(10, "one two three")).Equals(`
		 	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		 	efft.Effect("new\nvalue").Equals("oldvalue") // line 9
		-	efft.Effect("new value").Equals(!
		-		one
		-		two
		-		three
		-	!) // line 14
		+	efft.Effect("new value").Equals("one two three",
		+	) // line 14
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		`)

	efft.Note = "add a newline"
	efft.Effect(apply(10, "one\ntwo\nnewline\nthree\n")).Equals(`
		 		one
		 		two
		-		three
		-	!) // line 14
		+		newline
		+		three
		+		!) // line 14
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		`)

	efft.Note = "update in goroutine"
	efft.Effect(apply(17, "newvalue")).Equals(`
		 	efft.Effect("\nnew\n\nvalue").Equals("oldvalue")
		 	go func() {
		-		efft.Effect("newvalue").Equals("oldvalue") // line 17
		+		efft.Effect("newvalue").Equals("newvalue") // line 17
		 	}()
		 	efft.Effect("a", "b", "c").Equals("oldvalue")
		`)

	efft.Note = "expect has multiple arguments"
	efft.Effect(apply(19, "a,b,c")).Equals(`
		 		efft.Effect("newvalue").Equals("oldvalue") // line 17
		 	}()
		-	efft.Effect("a", "b", "c").Equals("oldvalue")
		+	efft.Effect("a", "b", "c").Equals("a,b,c")
		 	efft.Effect("a", "b", "c").Equals()
		 	efft.Effect("a", "b", "c").Equals("a", "b")
		`)

	efft.Note = "expectation is empty"
	efft.Effect(apply(20, "a,b,c")).Equals(`
		 	}()
		 	efft.Effect("a", "b", "c").Equals("oldvalue")
		-	efft.Effect("a", "b", "c").Equals()
		+	efft.Effect("a", "b", "c").Equals("a,b,c")
		 	efft.Effect("a", "b", "c").Equals("a", "b")
		 	efft.Effect("a", "b", "c").Equals(3)
		`)

	efft.Note = "expectation has multiple arguments"
	efft.Effect(apply(21, "a,b,c")).Equals(`
		 	efft.Effect("a", "b", "c").Equals("oldvalue")
		 	efft.Effect("a", "b", "c").Equals()
		-	efft.Effect("a", "b", "c").Equals("a", "b")
		+	efft.Effect("a", "b", "c").Equals("a,b,c")
		 	efft.Effect("a", "b", "c").Equals(3)
		 	// some comment before
		`)

	efft.Note = "expectation is a number"
	efft.Effect(apply(22, "a,b,c")).Equals(`
		 	efft.Effect("a", "b", "c").Equals()
		 	efft.Effect("a", "b", "c").Equals("a", "b")
		-	efft.Effect("a", "b", "c").Equals(3)
		+	efft.Effect("a", "b", "c").Equals("a,b,c")
		 	// some comment before
		 	efft.Effect("y\nx").Equals("x\ny") // line 24
		`)

	efft.Note = "adding expectation keeps the post-comment intact"
	efft.Effect(apply(24, "x\ny")).Equals(`
		 	efft.Effect("a", "b", "c").Equals(3)
		 	// some comment before
		-	efft.Effect("y\nx").Equals("x\ny") // line 24
		+	efft.Effect("y\nx").Equals(!
		+		x
		+		y!) // line 24
		 	efft.Effect("y\nx").Equals("x\ny")
		 	// some comment after
		`)

	efft.Note = "adding expectation keeps the next comment intact"
	efft.Effect(apply(25, "x\ny")).Equals(`
		 	// some comment before
		 	efft.Effect("y\nx").Equals("x\ny") // line 24
		-	efft.Effect("y\nx").Equals("x\ny")
		+	efft.Effect("y\nx").Equals(!
		+		x
		+		y!)
		 	// some comment after
		 }
		`)

	efft.Note = "backtick in the string means quoted string"
	efft.Effect(apply(6, "x\n`\ny")).Equals(`
		 	efft.Init(t)
		 	// line 5
		-	efft.Effect("somevalue").Equals("somevalue")
		+	efft.Effect("somevalue").Equals("x\n!\ny")
		 	efft.Effect("newvalue")
		 	efft.Effect( /* line 8 */ "newvalue").Equals(!oldvalue!)
		`)

	efft.Note = "bad replacement"
	efft.Effect(apply(1, "")).Equals("efft.ReplacementsFailed file=test.go lines=[1]")
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
			t.Errorf("effttest.UnexpectedXValue got=%d want=%d", got, want)
		}
	}
	t.Cleanup(func() { checkx(4) })

	efft.Init(t)
	efft.Override(&x, 5)
	checkx(5)
}
