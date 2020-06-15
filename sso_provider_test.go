package awsauth_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/99designs/keyring"
	awsauth "github.com/telia-oss/aws-auth"
	"github.com/telia-oss/aws-auth/fakes"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

func TestSSOProvider(t *testing.T) {
	type expectations struct {
		registerClientCalls     int
		createTokenCalls        int
		startAuthorizationCalls int
		getRoleCredentialsCalls int
	}

	tests := []struct {
		name             string
		profile          *awsauth.Profile
		tokenExpiration  time.Time
		clientExpiration time.Time
		expectations     *expectations
	}{
		{
			name: "works",
			profile: &awsauth.Profile{
				SSOStartURL: "https://login.awsapps.com/start",
			},
			expectations: &expectations{
				registerClientCalls:     1,
				createTokenCalls:        1,
				startAuthorizationCalls: 1,
				getRoleCredentialsCalls: 1,
			},
		},
		{
			name: "uses cache if it exists",
			profile: &awsauth.Profile{
				SSOStartURL: "https://cached.awsapps.com/start",
			},
			tokenExpiration:  time.Now().Add(1 * time.Hour),
			clientExpiration: time.Now().Add(1 * time.Hour),
			expectations: &expectations{
				getRoleCredentialsCalls: 1,
			},
		},
		{
			name: "registers new client on expiration",
			profile: &awsauth.Profile{
				SSOStartURL: "https://cached.awsapps.com/start",
			},
			tokenExpiration: time.Now().Add(1 * time.Hour),
			expectations: &expectations{
				registerClientCalls:     1,
				getRoleCredentialsCalls: 1,
			},
		},
		{
			name: "refreshes token on expiration",
			profile: &awsauth.Profile{
				SSOStartURL: "https://cached.awsapps.com/start",
			},
			clientExpiration: time.Now().Add(1 * time.Hour),
			expectations: &expectations{
				createTokenCalls:        1,
				startAuthorizationCalls: 1,
				getRoleCredentialsCalls: 1,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			fakeSSOClient := &fakes.FakeSSOAPI{}
			fakeOIDCClient := &fakes.FakeOIDCAPI{}

			fakeOIDCClient.RegisterClientRequestReturns(ssooidc.RegisterClientRequest{
				Request: fakeRequest(&ssooidc.RegisterClientOutput{
					ClientId:              aws.String("id"),
					ClientSecret:          aws.String("secret"),
					ClientSecretExpiresAt: aws.Int64(time.Now().Unix()),
				}),
			})

			fakeOIDCClient.CreateTokenRequestReturns(ssooidc.CreateTokenRequest{
				Request: fakeRequest(&ssooidc.CreateTokenOutput{
					AccessToken: aws.String("token"),
					ExpiresIn:   aws.Int64(3600),
				}),
			})

			fakeOIDCClient.StartDeviceAuthorizationRequestReturns(ssooidc.StartDeviceAuthorizationRequest{
				Request: fakeRequest(&ssooidc.StartDeviceAuthorizationOutput{
					DeviceCode:              aws.String("1234"),
					VerificationUriComplete: aws.String("https://device.sso.eu-west-1.amazonaws.com/?user_code=HZZB-FPRL"),
				}),
			})

			fakeSSOClient.GetRoleCredentialsRequestReturns(sso.GetRoleCredentialsRequest{
				Request: fakeRequest(&sso.GetRoleCredentialsOutput{
					RoleCredentials: &sso.RoleCredentials{
						AccessKeyId:     aws.String("accesskeyid"),
						SecretAccessKey: aws.String("secret"),
						SessionToken:    aws.String("token"),
						Expiration:      aws.Int64(time.Now().Add(1*time.Hour).Unix() * 1000),
					},
				}),
			})

			fakeKeyring := keyring.NewArrayKeyring([]keyring.Item{
				{
					Key:  "https://cached.awsapps.com/start",
					Data: []byte(newTestCredentialsData(t, tc.tokenExpiration, tc.clientExpiration)),
				},
			})

			p := &awsauth.SSOProvider{
				Profile:    tc.profile,
				OIDCClient: fakeOIDCClient,
				SSOClient:  fakeSSOClient,
				Store:      fakeKeyring,
				NoBrowser:  true,
			}

			creds, err := p.Retrieve(context.TODO())
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if creds.AccessKeyID != "accesskeyid" {
				t.Fatalf("unexpected credentials: %v", creds)
			}

			eqCount(t, fakeOIDCClient.RegisterClientRequestCallCount(), tc.expectations.registerClientCalls, "register client")
			eqCount(t, fakeOIDCClient.CreateTokenRequestCallCount(), tc.expectations.createTokenCalls, "create token")
			eqCount(t, fakeOIDCClient.StartDeviceAuthorizationRequestCallCount(), tc.expectations.startAuthorizationCalls, "start authorization")
			eqCount(t, fakeSSOClient.GetRoleCredentialsRequestCallCount(), tc.expectations.getRoleCredentialsCalls, "get role credentials")

		})
	}
}

func eqCount(t *testing.T, want int, got int, message string) {
	if got != want {
		t.Errorf("%s: call count %d != %d", message, got, want)
	}
}

func fakeRequest(data interface{}) *aws.Request {
	return &aws.Request{
		HTTPRequest:  &http.Request{},
		HTTPResponse: &http.Response{},
		Data:         data,
		Error:        nil,
		Retryer:      aws.NoOpRetryer{},
	}
}

func newTestCredentialsData(t *testing.T, tokenExpiration, clientExpiration time.Time) []byte {
	tpl := `{"Token":{"Token":"token","Expiration":"%s"},"Client":{"ID":"id","Secret":"secret","Expiration":"%s"}}`
	out := fmt.Sprintf(tpl, tokenExpiration.Format(time.RFC3339), clientExpiration.Format(time.RFC3339))
	return []byte(out)
}
