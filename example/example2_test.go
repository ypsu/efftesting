package example_test

import (
	"strings"
	"testing"

	"github.com/ypsu/efftesting/efft"
)

func TestSplit(t *testing.T) {
	efft.Init(t)

	// simple cut
	efft.Expect(strings.CutPrefix("hello world", "hello"))(" world")

	// failing cut
	efft.Expect(strings.CutPrefix("hello world", "world"))(`
		[
		  "hello world",
		  false
		]`)
}
