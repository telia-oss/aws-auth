package awsauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/skratchdot/open-golang/open"
)

const (
	awsFederationEndpoint = "https://signin.aws.amazon.com/federation"
)

// GetCredentials ...
func GetCredentials(configPath, profileName string, store keyring.Keyring) (aws.Credentials, error) {
	profile, err := GetAWSConfigProfile(configPath, profileName)
	if err != nil {
		return aws.Credentials{}, err
	}

	var provider aws.CredentialsProvider

	switch {
	case profile.IsSSOProfile():
		config := defaults.Config()
		config.Region = profile.SSORegion
		provider = &SSOProvider{
			OIDCClient:     ssooidc.New(config),
			SSOClient:      sso.New(config),
			Profile:        profile,
			Store:          store,
			NoBrowser:      false,
			RotationWindow: 30 * time.Minute,
		}
	default:
		return aws.Credentials{}, fmt.Errorf("unknown type for profile: %s", profileName)
	}
	return provider.Retrieve(context.TODO())
}

// Exec a command using the specified AWSProfile.
func Exec(configPath, profileName string, args []string) error {
	c, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigFiles([]string{configPath}),
		external.WithSharedConfigProfile(profileName),
	)
	if err != nil {
		return fmt.Errorf("load config: %s", err)
	}

	creds, err := c.Credentials.Retrieve(context.TODO())
	if err != nil {
		return fmt.Errorf("get credentials: %s", err)
	}

	env := make(map[string]string)
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		env[kv[0]] = kv[1]
	}

	for k, v := range map[string]string{
		"AWS_ACCESS_KEY_ID":      creds.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY":  creds.SecretAccessKey,
		"AWS_SESSION_TOKEN":      creds.SessionToken,
		"AWS_SESSION_EXPIRATION": creds.Expires.Format(time.RFC3339),
		"AWS_REGION":             c.Region,
	} {
		env[k] = v
	}

	cmd := newCommand(args, env)
	return cmd.Run()
}

func newCommand(args []string, envmap map[string]string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	for k, v := range envmap {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd
}

// Login to the AWS Console using credentials for the specified AWSProfile.
func Login(configPath, profileName string) error {
	c, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigFiles([]string{configPath}),
		external.WithSharedConfigProfile(profileName),
	)
	if err != nil {
		return fmt.Errorf("load config: %s", err)
	}

	creds, err := c.Credentials.Retrieve(context.TODO())
	if err != nil {
		return fmt.Errorf("get credentials: %s", err)
	}

	session, err := json.Marshal(map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	})
	if err != nil {
		return fmt.Errorf("marshal credentials: %s", err)
	}

	signinURL, err := buildURL(awsFederationEndpoint, map[string]string{
		"Action":  "getSigninToken",
		"Session": string(session),
	})
	if err != nil {
		return fmt.Errorf("build signin url: %s", err)
	}

	req, err := http.NewRequest("GET", signinURL, nil)
	if err != nil {
		return fmt.Errorf("new request: %s", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 response: %s", res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var token struct {
		Token string `json:"SigninToken"`
	}
	err = json.Unmarshal([]byte(body), &token)
	if err != nil {
		return err
	}

	destinationURL, err := buildURL("https://console.aws.amazon.com/console/home", map[string]string{"region": c.Region})
	if err != nil {
		return fmt.Errorf("build destination url: %s", err)
	}

	loginURL, err := buildURL(awsFederationEndpoint, map[string]string{
		"Action":      "login",
		"Issuer":      "aws-auth",
		"Destination": destinationURL,
		"SigninToken": token.Token,
	})
	if err != nil {
		return fmt.Errorf("build login url: %s", err)
	}
	if err := open.Run(loginURL); err != nil {
		return fmt.Errorf("open browser: %s", err)
	}
	return nil
}

func buildURL(base string, query map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
