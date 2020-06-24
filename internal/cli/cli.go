package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	awsauth "github.com/telia-oss/aws-auth"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Options for the CLI.
type Options struct {
	Writer io.Writer
}

// New returns a new kingpin.Application.
func New(opts *Options) *kingpin.Application {
	app := kingpin.New("aws-auth", "CLI and process helper for authenticating against AWS")
	app.DefaultEnvars().UsageWriter(opts.Writer).ErrorWriter(opts.Writer)

	var (
		configPath   = app.Flag("config", "Path to the config file where the AWS profiles are defined").Default(external.DefaultSharedConfigFilename()).String()
		get          = app.Command("get", "Get credentials for the specified AWS profile")
		getProfile   = get.Arg("profile", "Profile to target").Required().String()
		login        = app.Command("login", "Login to the AWS Console")
		loginProfile = login.Arg("profile", "Profile to use when creating the AWS Credentials").Required().String()
		exec         = app.Command("exec", "Execute a command after populating AWS Credentials")
		execProfile  = exec.Arg("profile", "Profile to use when creating the AWS Credentials").Required().String()
		execCommand  = exec.Arg("command", "Commands to execute").Strings()
	)

	keyringConfig := keyring.Config{
		ServiceName:              "aws-auth",
		FileDir:                  "~/.aws-auth/keys/",
		FilePasswordFunc:         fileKeyringPassphrasePrompt,
		LibSecretCollectionName:  "aws-auth",
		KWalletAppID:             "aws-auth",
		KWalletFolder:            "aws-auth",
		WinCredPrefix:            "aws-auth",
		KeychainName:             "aws-auth",
		KeychainTrustApplication: true,
	}

	get.Action(func(_ *kingpin.ParseContext) error {
		keyring, err := keyring.Open(keyringConfig)
		if err != nil {
			return err
		}
		creds, err := awsauth.GetCredentials(*configPath, *getProfile, keyring)
		if err != nil {
			return err
		}
		output := struct {
			Version         int    `json:"Version"`
			AccessKeyID     string `json:"AccessKeyId"`
			SecretAccessKey string `json:"SecretAccessKey"`
			SessionToken    string `json:"SessionToken"`
			Expiration      string `json:"Expiration"`
		}{
			Version:         1,
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
			Expiration:      creds.Expires.Format(time.RFC3339),
		}
		return json.NewEncoder(opts.Writer).Encode(output)
	})

	exec.Action(func(_ *kingpin.ParseContext) error {
		keyring, err := keyring.Open(keyringConfig)
		if err != nil {
			return err
		}
		return awsauth.Exec(*configPath, *execProfile, keyring, *execCommand)
	})

	login.Action(func(_ *kingpin.ParseContext) error {
		keyring, err := keyring.Open(keyringConfig)
		if err != nil {
			return err
		}
		return awsauth.Login(*configPath, *loginProfile, keyring)
	})

	return app
}

func fileKeyringPassphrasePrompt(prompt string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	b, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "\n")
	return string(b), nil
}
