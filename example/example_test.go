package example_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ypsu/efftesting"
)

func MyStringLength(s string) int {
	return len(s)
}

func TestLength(t *testing.T) {
	et := efftesting.New(t)
	et.Expect("string length of tükör", MyStringLength("tükör"), "7")
	et.Expect("slice example", strings.Split("This is a sentence.", " "), `
		[
		  "This",
		  "is",
		  "a",
		  "sentence."
		]`)
}

func TestMain(m *testing.M) {
	os.Exit(efftesting.Main(m))
}
