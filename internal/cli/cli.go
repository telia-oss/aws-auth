package cli

import (
	"io"

	auth "github.com/telia-oss/aws-auth"

	"gopkg.in/alecthomas/kingpin.v2"
)

// New returns a new kingpin.Application.
func New(w io.Writer) *kingpin.Application {
	app := kingpin.New("aws-auth", "CLI for authenticating against AWS").DefaultEnvars()
	var (
		messages = app.Arg("message", "Message(s) to print").Strings()
		times    = app.Flag("times", "Number of times to print the messages").Default("1").Int()
	)
	app.Action(func(_ *kingpin.ParseContext) error {
		for i := 0; i < *times; i++ {
			for _, m := range *messages {
				auth.Print(w, m)
			}
		}
		return nil
	})
	return app
}
