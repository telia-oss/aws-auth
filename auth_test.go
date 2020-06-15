package awsauth_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

// This test exists to verify that the AWS SDK Go version prioritises
// the credential_process over other providers (where we need it to).
func TestCredentialProcessPriority(t *testing.T) {
	tests := []struct {
		description string
		profile     string
		config      string
	}{
		{
			description: "works with sso profiles",
			profile:     "sso-profile",
			config: strings.TrimSpace(`
[profile sso-profile]
sso_start_url      = https://example.awsapps.com/start
sso_region         = eu-west-1
sso_account_id     = 111122223333
sso_role_name      = ReadOnly
credential_process = echo %q
			`),
		},
		{
			description: "works with empty profiles",
			profile:     "default",
			config: strings.TrimSpace(`
[default]
region             = eu-west-1
credential_process = echo %q
			`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			credentialsProcessOutput := `{"Version":1,"AccessKeyId":"KEY","SecretAccessKey":"SECRET"}`

			f := createTestConfig(t, fmt.Sprintf(tc.config, credentialsProcessOutput))
			defer os.Remove(f.Name())

			c, err := external.LoadDefaultAWSConfig(
				external.WithSharedConfigProfile(tc.profile),
				external.WithSharedConfigFiles([]string{f.Name()}),
			)
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.Credentials.Retrieve(context.TODO())
			if err != nil {
				t.Fatal(err)
			}
			want := aws.Credentials{
				AccessKeyID:     "KEY",
				SecretAccessKey: "SECRET",
				Source:          "ProcessProvider",
			}
			eq(t, want, got)
		})
	}
}
