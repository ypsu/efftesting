package example_test

import (
	"strings"
	"testing"

	"github.com/ypsu/efftesting/efft"
)

func TestSplit(t *testing.T) {
	efft.Init(t)

	// simple cut
	efft.Effect(strings.CutPrefix("hello world", "hello")).Equals(" world")

	// failing cut
	efft.Effect(strings.CutPrefix("hello world", "world")).Equals(`
		[
		  "hello world",
		  false
		]`)
}
