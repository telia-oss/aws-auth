package main

import (
	"os"

	"github.com/telia-oss/aws-auth/internal/cli"

	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "dev"

func main() {
	app := cli.New(os.Stdout).Version(version)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
