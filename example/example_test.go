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
	efft.Effect(MyStringLength("tükör")).Equals("7")
	efft.Effect(strings.Split("This is a sentence.", " ")).Equals(`
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
	efft.Effect(msg).Equals("")
	efft.Effect(err).Equals("empty name")

	msg, err = hello("Glady")
	efft.Effect(err).Equals("null")
	efft.Effect(regexp.MustCompile(`\bGlady\b`).MatchString(msg)).Equals("true")
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
