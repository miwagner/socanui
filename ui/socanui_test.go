package ui

import (
	"regexp"
	"testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestToHex(t *testing.T) {
	name := "Gladys"

	want := regexp.MustCompile(`\b` + name + `\b`)
	msg := ""
	if !want.MatchString(msg) {
		t.Fatalf(`toHex = %q, want match for %#q, nil`, msg, want)
	}
}
