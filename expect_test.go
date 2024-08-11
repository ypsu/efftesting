package efftesting

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	et.Expect("intslice", []int{1, 2, 3, 4, 5}, "[1 2 3 4 5]")
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
		 }
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
		 }
		 
	`)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 || os.Getenv("EFFTESTING_UPDATE") != "1" {
		if len(defaultReplacer.replacements) != 0 {
			fmt.Fprintf(os.Stderr, "Expectations need updating, use `EFFTESTING_UPDATE=1 go test ./...` for that.\n")
		}
		os.Exit(code)
	}
	if len(defaultReplacer.replacements) != 0 {
		_, testfile, _, _ := runtime.Caller(0)
		if err := defaultReplacer.apply(testfile); err != nil {
			fmt.Fprintf(os.Stderr, "efftesting update failed: %v.\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Expectations updated.\n")
	}
	os.Exit(code)
}
