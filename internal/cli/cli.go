package cli

import (
	"io"

	"github.com/itsdalmo/gotemplate"

	"gopkg.in/alecthomas/kingpin.v2"
)

// New returns a new kingpin.Application.
func New(w io.Writer) *kingpin.Application {
	app := kingpin.New("gotemplate", "Template for go CLIs").DefaultEnvars()
	var (
		messages = app.Arg("message", "Message(s) to print").Strings()
		times    = app.Flag("times", "Number of times to print the messages").Default("1").Int()
	)
	app.Action(func(_ *kingpin.ParseContext) error {
		for i := 0; i < *times; i++ {
			for _, m := range *messages {
				gotemplate.Print(w, m)
			}
		}
		return nil
	})
	return app
}
