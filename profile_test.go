package awsauth_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	awsauth "github.com/telia-oss/aws-auth"
)

var defaultTestConfig = strings.TrimSpace(`
[default]
region = eu-west-1
output = json

[profile root-profile]
mfa_serial         = arn:aws:iam::111122223333:mfa/first.last
credential_process = aws-auth get root-profile

[profile role-profile]
source_profile = root-profile
role_arn       = arn:aws:iam::111122223333:role/read-only
mfa_serial     = arn:aws:iam::111122223333:mfa/first.last

[profile sso-profile]
sso_start_url  = https://example.awsapps.com/start
sso_region     = eu-west-1
sso_account_id = 111122223333
sso_role_name  = ReadOnly

[unknown]
region = eu-north-1
`)

func createTestConfig(t *testing.T, content string) *os.File {
	f, err := ioutil.TempFile("", "aws-auth-test-config")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}
	return f
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		description string
		profile     string
		want        *awsauth.Profile
		wantSSO     bool
	}{
		{
			description: "works for SSO profiles",
			profile:     "sso-profile",
			want: &awsauth.Profile{
				Name:         "sso-profile",
				SSOStartURL:  "https://example.awsapps.com/start",
				SSORegion:    "eu-west-1",
				SSOAccountID: "111122223333",
				SSORoleName:  "ReadOnly",
			},
			wantSSO: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			f := createTestConfig(t, defaultTestConfig)
			defer os.Remove(f.Name())

			profile, err := awsauth.GetAWSConfigProfile(f.Name(), tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			eq(t, tc.want, profile)
			eq(t, tc.wantSSO, profile.IsSSOProfile())
		})
	}
}

func eq(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nexpected:\n%v\n\ngot:\n%v", expected, got)
	}
}
