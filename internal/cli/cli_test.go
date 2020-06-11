package cli_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/itsdalmo/gotemplate/internal/cli"
)

func TestCLI(t *testing.T) {
	tests := []struct {
		description string
		command     []string
		expected    string
	}{
		{
			description: "works",
			command:     []string{"hello", "there", "--times", "2"},
			expected: strings.TrimSpace(`
hello
there
hello
there
			 `),
		},
		{
			description: "help message",
			command:     []string{"--help"},
			expected: strings.TrimSpace(`
usage: gotemplate [<flags>] [<message>...]

Template for go CLIs

Flags:
  --help     Show context-sensitive help (also try --help-long and --help-man).
  --times=1  Number of times to print the messages

Args:
  [<message>]  Message(s) to print
			 `),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var b bytes.Buffer
			app := cli.New(&b)
			app.Writer(&b)
			app.ErrorWriter(&b)
			app.Terminate(nil)

			_, err := app.Parse(tc.command)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			eq(t, tc.expected, strings.TrimSpace(b.String()))
		})
	}
}

func eq(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nexpected:\n%v\n\ngot:\n%v", expected, got)
	}
}
