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
}
