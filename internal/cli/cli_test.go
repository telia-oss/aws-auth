package cli_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/telia-oss/aws-auth/internal/cli"
)

func TestCLI(t *testing.T) {
	tests := []struct {
		description string
		command     []string
		expected    string
	}{
		{
			description: "works",
			command:     []string{"exec", "test-profile"},
			expected:    strings.TrimSpace(``),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var b bytes.Buffer
			app := cli.New(&cli.Options{Writer: &b})
			app.Terminate(nil)

			// _, err := app.Parse(tc.command)
			// if err != nil && err != kingpin.ErrCommandNotSpecified {
			// 	t.Fatalf("unexpected error: %s", err)
			// }
			// eq(t, tc.expected, strings.TrimSpace(b.String()))
		})
	}
}

func eq(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nexpected:\n%v\n\ngot:\n%v", expected, got)
	}
}
