package awsauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/skratchdot/open-golang/open"
)

const authorizationTemplate = `
Attempting to automatically open the SSO authorization page in your default
browser. If the browser does not open or you wish to use a different device to
authorize this request, open the following URL:

%s
(Use Ctrl-C to abort)

`

var _ aws.CredentialsProvider = &SSOProvider{}

// SSOProvider for temporary SSO Credentials.
type SSOProvider struct {
	OIDCClient     OIDCAPI
	SSOClient      SSOAPI
	Store          keyring.Keyring
	Profile        *Profile
	NoBrowser      bool
	RotationWindow time.Duration
}

type oidcClientToken struct {
	Token      string
	Expiration time.Time
}

type oidcClientCredentials struct {
	ID         string
	Secret     string
	Expiration time.Time
}

// Retrieve implements Provider.
func (p *SSOProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	token, err := p.getAccessToken(ctx, p.Profile)
	if err != nil {
		return aws.Credentials{}, err
	}

	resp, err := p.SSOClient.GetRoleCredentialsRequest(&sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token.Token),
		AccountId:   aws.String(p.Profile.SSOAccountID),
		RoleName:    aws.String(p.Profile.SSORoleName),
	}).Send(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return aws.Credentials{
		AccessKeyID:     aws.StringValue(resp.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.StringValue(resp.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.StringValue(resp.RoleCredentials.SessionToken),
		CanExpire:       true,
		Expires:         aws.MillisecondsTimeValue(resp.RoleCredentials.Expiration).Add(-p.RotationWindow),
	}, nil
}

func (p *SSOProvider) getAccessToken(ctx context.Context, profile *Profile) (*oidcClientToken, error) {
	var (
		creds = &struct {
			Token  *oidcClientToken
			Client *oidcClientCredentials
		}{
			Token:  &oidcClientToken{},
			Client: &oidcClientCredentials{},
		}
		credsUpdated bool
		now          = time.Now()
	)

	item, err := p.Store.Get(profile.SSOStartURL)
	if err != nil && err != keyring.ErrKeyNotFound {
		return nil, err
	}

	if item.Data != nil {
		if err = json.Unmarshal(item.Data, &creds); err != nil {
			return nil, fmt.Errorf("invalid data in keyring: %v", err)
		}
	}

	if creds.Client.Expiration.Add(-p.RotationWindow).Before(now) {
		creds.Client, err = p.registerNewClient(ctx)
		if err != nil {
			return nil, err
		}
		credsUpdated = true
	}

	if creds.Token.Expiration.Add(-p.RotationWindow).Before(now) {
		creds.Token, err = p.createNewClientToken(ctx, profile.SSOStartURL, creds.Client)
		if err != nil {
			return nil, err
		}
		credsUpdated = true
	}

	if credsUpdated {
		bytes, err := json.Marshal(creds)
		if err != nil {
			return nil, err
		}
		err = p.Store.Set(keyring.Item{
			Key:                         profile.SSOStartURL,
			Label:                       fmt.Sprintf("aws-auth (%s)", profile.SSOStartURL),
			Data:                        bytes,
			KeychainNotTrustApplication: true,
		})
		if err != nil {
			return nil, err
		}
	}

	return creds.Token, nil
}

func (p *SSOProvider) registerNewClient(ctx context.Context) (*oidcClientCredentials, error) {
	c, err := p.OIDCClient.RegisterClientRequest(&ssooidc.RegisterClientInput{
		ClientName: aws.String("aws-auth"),
		ClientType: aws.String("public"),
	}).Send(ctx)
	if err != nil {
		return nil, err
	}
	return &oidcClientCredentials{
		ID:         aws.StringValue(c.ClientId),
		Secret:     aws.StringValue(c.ClientSecret),
		Expiration: time.Unix(aws.Int64Value(c.ClientSecretExpiresAt), 0),
	}, nil

}

func (p *SSOProvider) createNewClientToken(ctx context.Context, startURL string, creds *oidcClientCredentials) (*oidcClientToken, error) {
	auth, err := p.OIDCClient.StartDeviceAuthorizationRequest(&ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(creds.ID),
		ClientSecret: aws.String(creds.Secret),
		StartUrl:     aws.String(startURL),
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, authorizationTemplate, aws.StringValue(auth.VerificationUriComplete))
	if !p.NoBrowser {
		if err := open.Run(aws.StringValue(auth.VerificationUriComplete)); err != nil {
			log.Printf("failed to open browser: %s", err)
		}
	}

	// The are the default value defined in the following RFC:
	// https://tools.ietf.org/html/draft-ietf-oauth-device-flow-15#section-3.5
	var (
		retryInterval = 5 * time.Second
		slowDownDelay = 5 * time.Second
	)
	if i := aws.Int64Value(auth.Interval); i > 0 {
		retryInterval = time.Duration(i) * time.Second
	}

	for {
		t, err := p.OIDCClient.CreateTokenRequest(&ssooidc.CreateTokenInput{
			ClientId:     aws.String(creds.ID),
			ClientSecret: aws.String(creds.Secret),
			DeviceCode:   auth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		}).Send(ctx)
		if err != nil {
			e, ok := err.(awserr.Error)
			if !ok {
				return nil, err
			}
			switch e.Code() {
			case ssooidc.ErrCodeSlowDownException:
				retryInterval += slowDownDelay
				fallthrough
			case ssooidc.ErrCodeAuthorizationPendingException:
				time.Sleep(retryInterval)
				continue
			default:
				return nil, err
			}
		}
		return &oidcClientToken{
			Token:      aws.StringValue(t.AccessToken),
			Expiration: time.Now().Add(time.Duration(aws.Int64Value(t.ExpiresIn)) * time.Second),
		}, nil
	}
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_oidc_api.go . OIDCAPI
type OIDCAPI interface {
	RegisterClientRequest(*ssooidc.RegisterClientInput) ssooidc.RegisterClientRequest
	StartDeviceAuthorizationRequest(*ssooidc.StartDeviceAuthorizationInput) ssooidc.StartDeviceAuthorizationRequest
	CreateTokenRequest(*ssooidc.CreateTokenInput) ssooidc.CreateTokenRequest
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_sso_api.go . SSOAPI
type SSOAPI interface {
	GetRoleCredentialsRequest(*sso.GetRoleCredentialsInput) sso.GetRoleCredentialsRequest
}
