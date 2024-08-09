package efftesting_test

import (
	"testing"

	"github.com/ypsu/efftesting"
)

func TestExpectOK(t *testing.T) {
	et := efftesting.New(t)
	et.Expect("bool1", false, "false")
	et.Expect("bool2", true, "true")
	et.Expect("int1", 0, "0")
	et.Expect("int2", -43, "-43")
	et.Expect("string", "blah", "blah")
	et.Expect("multiline1_quoted", "hello\nworld\n", "hello\nworld\n")
	et.Expect("multiline2_quoted", "\nhello\nworld\n", "\n\nhello\nworld\n")
	et.Expect("multiline1_backticked", "hello\nworld\n", `
		hello
		world
	`)
	et.Expect("multiline2_backticked", "\nhello\nworld\n", `

		hello
		world
	`)
}
