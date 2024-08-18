package example_test

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
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

// TestHello demonstrates https://go.dev/doc/tutorial/add-a-test in efftesting format.
func TestHello(t *testing.T) {
	et := efftesting.New(t)

	msg, err := hello("")
	et.Expect("empty hello is empty msg", msg, "")
	et.Expect("empty hello's error", err, "empty name")

	msg, err = hello("Glady")
	et.Expect("no error for non-empty name", err, "null")
	et.Expect("name is in msg", regexp.MustCompile(`\bGlady\b`).MatchString(msg), "true")
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

func TestMain(m *testing.M) {
	os.Exit(efftesting.Main(m))
}
