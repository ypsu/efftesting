package example_test

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/ypsu/efftesting/efft"
)

func MyStringLength(s string) int {
	return len(s)
}

func TestLength(t *testing.T) {
	efft.Init(t)
	efft.Expect(MyStringLength("tükör"))("7")
	efft.Expect(strings.Split("This is a sentence.", " "))(`
		[
		  "This",
		  "is",
		  "a",
		  "sentence."
		]`)
}

// TestHello demonstrates https://go.dev/doc/tutorial/add-a-test in efftesting format.
func TestHello(t *testing.T) {
	efft.Init(t)

	msg, err := hello("")
	efft.Expect(msg)("")
	efft.Expect(err)("empty name")

	msg, err = hello("Glady")
	efft.Expect(err)("null")
	efft.Expect(regexp.MustCompile(`\bGlady\b`).MatchString(msg))("true")
}

// randomFormat returns one of a set of greeting messages. The returned
// message is selected at random.
// From https://go.dev/doc/tutorial/greetings-multiple-people.
func randomFormat() string {
	// A slice of message formats.
	formats := []string{
		"Hi, %v. Welcome!",
		"Great to see you, %v!",
		"Hail, %v! Well met!",
	}

	// Return one of the message formats selected at random.
	return formats[rand.Intn(len(formats))]
}

// hello returns a greeting for the named person.
// From https://go.dev/doc/tutorial/greetings-multiple-people.
func hello(name string) (string, error) {
	// If no name was given, return an error with a message.
	if name == "" {
		return name, errors.New("empty name")
	}
	// Create a message using a random format.
	message := fmt.Sprintf(randomFormat(), name)
	return message, nil
}
